package helpers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gitlab.com/gitlab-org/gitlab-runner/common"
)

const (
	artifactsMetadataFormat   = "%v-artifacts-metadata.json"
	attestationType           = "https://in-toto.io/Statement/v0.1"
	attestationPredicateType  = "https://slsa.dev/provenance/v0.2"
	attestationTypeFormat     = "https://gitlab.com/gitlab-org/gitlab-runner/-/blob/%v/PROVENANCE.md"
	attestationRunnerIDFormat = "%v/-/runners/%v"
)

type artifactMetadataGenerator struct {
	GenerateArtifactsMetadata bool     `long:"generate-artifacts-metadata"`
	RunnerID                  int64    `long:"runner-id"`
	RepoURL                   string   `long:"repo-url"`
	RepoDigest                string   `long:"repo-digest"`
	JobName                   string   `long:"job-name"`
	ExecutorName              string   `long:"executor-name"`
	RunnerName                string   `long:"runner-name"`
	Parameters                []string `long:"metadata-parameter"`
	StartedAtRFC3339          string   `long:"started-at"`
	EndedAtRFC3339            string   `long:"ended-at"`
}

type AttestationMetadata struct {
	Type          string                  `json:"_type"`
	Subject       []AttestationSubject    `json:"subject"`
	PredicateType string                  `json:"predicateType"`
	Predicate     AttestationPredicate    `json:"predicate"`
	Metadata      AttestationMetadataInfo `json:"metadata"`
	// Materials are currently intentionally empty
	// https://gitlab.com/gitlab-org/gitlab-runner/-/issues/28940#note_976823431
	Materials []interface{} `json:"materials"`
}

type AttestationSubject struct {
	Name   string            `json:"name"`
	Digest AttestationDigest `json:"digest"`
}

type AttestationDigest struct {
	Sha256 string `json:"sha256"`
}

type AttestationPredicate struct {
	BuildType  string                         `json:"buildType"`
	Builder    AttestationPredicateBuilder    `json:"builder"`
	Invocation AttestationPredicateInvocation `json:"invocation"`
}

type AttestationPredicateBuilder struct {
	ID string `json:"id"`
}

type AttestationPredicateInvocation struct {
	ConfigSource AttestationPredicateInvocationConfigSource `json:"configSource"`
	Environment  AttestationPredicateInvocationEnvironment  `json:"environment"`
	Parameters   AttestationPredicateInvocationParameters   `json:"parameters"`
}

type AttestationPredicateInvocationConfigSource struct {
	URI        string            `json:"uri"`
	Digest     AttestationDigest `json:"digest"`
	EntryPoint string            `json:"entryPoint"`
}

type AttestationPredicateInvocationEnvironment struct {
	Name         string `json:"name"`
	Executor     string `json:"executor"`
	Architecture string `json:"architecture"`
}

type AttestationPredicateInvocationParameters map[string]string

type AttestationMetadataInfo struct {
	BuildStartedOn  common.TimeRFC3339                  `json:"buildStartedOn"`
	BuildFinishedOn common.TimeRFC3339                  `json:"buildFinishedOn"`
	Reproducible    bool                                `json:"reproducible"`
	Completeness    AttestationMetadataInfoCompleteness `json:"completeness"`
}

type AttestationMetadataInfoCompleteness struct {
	Parameters  bool `json:"parameters"`
	Environment bool `json:"environment"`
	Materials   bool `json:"materials"`
}

type generateMetadataOptions struct {
	files map[string]os.FileInfo
	wd    string
	jobID int64
}

func (g *artifactMetadataGenerator) generateMetadataToFile(opts generateMetadataOptions) (string, error) {
	metadata, err := g.metadata(opts)
	if err != nil {
		return "", err
	}

	file := filepath.Join(opts.wd, fmt.Sprintf(artifactsMetadataFormat, opts.jobID))

	b, err := json.MarshalIndent(metadata, "", " ")
	if err != nil {
		return "", err
	}

	err = ioutil.WriteFile(file, b, 0700)
	return file, err
}

func (g *artifactMetadataGenerator) metadata(opts generateMetadataOptions) (AttestationMetadata, error) {
	subjects, err := g.generateSubjects(opts.files)
	if err != nil {
		return AttestationMetadata{}, err
	}

	parameters := AttestationPredicateInvocationParameters{}
	for _, param := range g.Parameters {
		parameters[param] = ""
	}

	startedAt, endedAt, err := g.parseTimings()
	if err != nil {
		return AttestationMetadata{}, err
	}

	return AttestationMetadata{
		Type:          attestationType,
		Subject:       subjects,
		PredicateType: attestationPredicateType,
		Predicate: AttestationPredicate{
			BuildType: fmt.Sprintf(attestationTypeFormat, g.version()),
			Builder:   AttestationPredicateBuilder{ID: fmt.Sprintf(attestationRunnerIDFormat, g.RepoURL, g.RunnerID)},
			Invocation: AttestationPredicateInvocation{
				ConfigSource: AttestationPredicateInvocationConfigSource{
					URI:        g.RepoURL,
					Digest:     AttestationDigest{Sha256: g.RepoDigest},
					EntryPoint: g.JobName,
				},
				Environment: AttestationPredicateInvocationEnvironment{
					Name:         g.RunnerName,
					Executor:     g.ExecutorName,
					Architecture: common.AppVersion.Architecture,
				},
				Parameters: parameters,
			},
		},
		Metadata: AttestationMetadataInfo{
			BuildStartedOn:  common.TimeRFC3339{Time: startedAt},
			BuildFinishedOn: common.TimeRFC3339{Time: endedAt},
			Reproducible:    false,
			Completeness: AttestationMetadataInfoCompleteness{
				Parameters:  true,
				Environment: true,
				Materials:   false,
			},
		},
		Materials: make([]interface{}, 0),
	}, nil
}

func (g *artifactMetadataGenerator) version() string {
	if strings.HasPrefix(common.AppVersion.Version, "v") {
		return common.AppVersion.Version
	}

	return common.AppVersion.Revision
}

func (g *artifactMetadataGenerator) parseTimings() (time.Time, time.Time, error) {
	startedAt, err := time.Parse(time.RFC3339, g.StartedAtRFC3339)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	endedAt, err := time.Parse(time.RFC3339, g.EndedAtRFC3339)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	return startedAt, endedAt, nil
}

func (g *artifactMetadataGenerator) generateSubjects(files map[string]os.FileInfo) ([]AttestationSubject, error) {
	subjects := make([]AttestationSubject, 0, len(files))

	for file := range files {
		subject, err := func(file string) (AttestationSubject, error) {
			f, err := os.Open(file)
			if err != nil {
				return AttestationSubject{}, err
			}
			defer f.Close()

			h := sha256.New()
			if _, err := io.Copy(h, f); err != nil {
				return AttestationSubject{}, err
			}

			return AttestationSubject{
				Name:   file,
				Digest: AttestationDigest{Sha256: hex.EncodeToString(h.Sum(nil))},
			}, nil
		}(file)

		if err != nil {
			return nil, err
		}

		subjects = append(subjects, subject)
	}

	return subjects, nil
}
