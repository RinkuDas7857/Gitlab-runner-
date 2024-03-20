package images

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	"github.com/samber/lo"
	"gitlab.com/gitlab-org/gitlab-runner/magefiles/build"
	"gitlab.com/gitlab-org/gitlab-runner/magefiles/ci"
	"gitlab.com/gitlab-org/gitlab-runner/magefiles/docker"
	"gitlab.com/gitlab-org/gitlab-runner/magefiles/env"
)

var helperImageName = env.NewDefault("HELPER_IMAGE_NAME", "gitlab-runner-helper")

var platformMap = map[string]docker.PlatformSpec{
	"x86_64":  {Os: "linux", Architecture: "amd64"},
	"arm":     {Os: "linux", Architecture: "arm", Variant: "v7"},
	"arm64":   {Os: "linux", Architecture: "arm64", Variant: "v8"},
	"s390x":   {Os: "linux", Architecture: "s390x"},
	"ppc64le": {Os: "linux", Architecture: "ppc64le"},
	"riscv64": {Os: "linux", Architecture: "riscv64"},
}

var flavorsSupportingPWSH = []string{
	"alpine",
	"alpine3.16",
	"alpine3.17",
	"alpine3.18",
	"ubuntu",
}

type helperBlueprint build.TargetBlueprint[build.Component, build.Component, []helperBuildSet]

// Collects the architecture-specific variants for a given "flavor"
// (e.g. alpine3.19, ubuntu, etc) of the helper together in one set to
// facilitate building a single manifest list for each flavor.
type helperBuildSet struct {
	componentBuilds    []helperBuild
	manifestTagSpec    helperTagSpec
	manifestAliasSpecs []helperTagSpec
	manifestType       string
}

func (bs helperBuildSet) tags() []string {
	tags := lo.Flatten(lo.Map(bs.componentBuilds, func(item helperBuild, _ int) []string {
		return append(lo.Map(item.aliasSpecs, func(item helperTagSpec, _ int) string {
			return item.render()
		}), item.tagSpec.render())
	}))
	tags = append(tags, bs.manifestTagSpec.render())
	tags = append(tags, lo.Map(bs.manifestAliasSpecs, func(item helperTagSpec, _ int) string {
		return item.render()
	})[:]...)
	return tags
}

func (bs helperBuildSet) renderManifestToolYaml() (string, error) {
	manifest := docker.ManifestToolSpec{
		Image: bs.manifestTagSpec.render(),
		Manifests: lo.Map(bs.componentBuilds, func(build helperBuild, _ int) docker.ManifestImage {
			return docker.ManifestImage{
				Image:    build.tagSpec.render(),
				Platform: build.platform,
			}
		}),
	}
	yaml, err := manifest.Render()
	return yaml, err
}

func (bs helperBuildSet) renderManifestToolAliasYaml(i int) (string, error) {
	if i >= len(bs.manifestAliasSpecs) {
		return "", fmt.Errorf("no alias %d", i)
	}
	manifest := docker.ManifestToolSpec{
		Image: bs.manifestAliasSpecs[i].render(),
		Manifests: lo.Map(bs.componentBuilds, func(build helperBuild, _ int) docker.ManifestImage {
			return docker.ManifestImage{
				Image:    build.aliasSpecs[i].render(),
				Platform: build.platform,
			}
		}),
	}
	yaml, err := manifest.Render()
	return yaml, err
}

// Describes the architecture-specific artifacts for a given flavor.
type helperBuild struct {
	archive    string
	platform   docker.PlatformSpec
	tagSpec    helperTagSpec
	aliasSpecs []helperTagSpec
}

type helperTagSpec struct {
	Suffix        string
	Prefix        string
	Arch          string
	ImageName     string
	RegistryImage string
	Version       string
	baseTemplate  string
	specTemplate  string
}

func newHelperTagSpec(prefix, suffix, arch, imageName, registryImage, version string) helperTagSpec {
	return helperTagSpec{
		Prefix:        prefix,
		Suffix:        suffix,
		Arch:          arch,
		RegistryImage: registryImage,
		ImageName:     imageName,
		Version:       version,
		baseTemplate:  "{{ .RegistryImage }}/{{ .ImageName }}:{{ if .Prefix }}{{ .Prefix }}-{{ end }}{{ if .Arch }}{{ .Arch }}-{{ end }}{{ .Version }}{{ if .Suffix }}-{{ .Suffix }}{{ end }}",
		specTemplate:  "spec-{{ if .Prefix }}{{ .Prefix }}-{{ end }}{{ .Version }}{{ if .Suffix }}-{{ .Suffix }}{{ end }}.yml",
	}
}

func (l helperTagSpec) renderTemplate(tmplSrc string) string {
	var out bytes.Buffer
	tmpl := lo.Must(template.New("tmpl").Parse(tmplSrc))

	lo.Must0(tmpl.Execute(&out, &l))

	return out.String()
}

func (l helperTagSpec) render() string {
	return l.renderTemplate(l.baseTemplate)
}

func (l helperTagSpec) renderSpecFileName() string {
	return l.renderTemplate(l.specTemplate)
}

type helperBlueprintImpl struct {
	build.BlueprintBase
	buildSets []helperBuildSet
}

func (b helperBlueprintImpl) Dependencies() []build.Component {
	return lo.Flatten(lo.Map(b.buildSets, func(item helperBuildSet, _ int) []build.Component {
		return lo.Map(item.componentBuilds, func(item helperBuild, _ int) build.Component {
			return build.NewDockerImageArchive(item.archive)
		})
	}))
}

func (b helperBlueprintImpl) Artifacts() []build.Component {
	return lo.Flatten(lo.Map(b.buildSets, func(item helperBuildSet, _ int) []build.Component {
		return lo.Map(item.tags(), func(item string, _ int) build.Component {
			return build.NewDockerImage(item)
		})
	}))
}

func (b helperBlueprintImpl) Data() []helperBuildSet {
	return b.buildSets
}

func AssembleReleaseHelper(flavor, prefix string) helperBlueprint {
	var archs []string
	switch flavor {
	case "ubi-fips":
		archs = []string{"x86_64"}
	case "alpine-edge":
		archs = []string{"x86_64", "arm", "arm64", "s390x", "ppc64le", "riscv64"}
	default:
		archs = []string{"x86_64", "arm", "arm64", "s390x", "ppc64le"}
	}

	builds := helperBlueprintImpl{
		BlueprintBase: build.NewBlueprintBase(ci.RegistryImage, ci.RegistryAuthBundle, docker.BuilderEnvBundle, helperImageName),
		buildSets:     []helperBuildSet{},
	}

	imageName := builds.Env().Value(helperImageName)
	registryImage := builds.Env().Value(ci.RegistryImage)

	primaryBuildSet := helperBuildSet{
		componentBuilds:    []helperBuild{},
		manifestTagSpec:    newHelperTagSpec(prefix, "", "", imageName, registryImage, build.Revision()),
		manifestAliasSpecs: []helperTagSpec{newHelperTagSpec(prefix, "", "", imageName, registryImage, build.RefTag())},
		manifestType:       "oci",
	}

	if build.IsLatest() {
		primaryBuildSet.manifestAliasSpecs = append(primaryBuildSet.manifestAliasSpecs, newHelperTagSpec(prefix, "", "", imageName, registryImage, "latest"))
	}

	for _, arch := range archs {
		b := helperBuild{
			archive:    fmt.Sprintf("out/helper-images/prebuilt-%s-%s.tar.xz", flavor, arch),
			platform:   platformMap[arch],
			tagSpec:    newHelperTagSpec(prefix, "", arch, imageName, registryImage, build.Revision()),
			aliasSpecs: []helperTagSpec{newHelperTagSpec(prefix, "", arch, imageName, registryImage, build.RefTag())},
		}
		if build.IsLatest() {
			b.aliasSpecs = append(b.aliasSpecs, newHelperTagSpec(prefix, "", arch, imageName, registryImage, "latest"))
		}
		primaryBuildSet.componentBuilds = append(primaryBuildSet.componentBuilds, b)
	}
	builds.buildSets = append(builds.buildSets, primaryBuildSet)

	if lo.Contains(flavorsSupportingPWSH, flavor) {
		pwshBuildSet := helperBuildSet{
			componentBuilds:    []helperBuild{},
			manifestTagSpec:    newHelperTagSpec(prefix, "pwsh", "", imageName, registryImage, build.Revision()),
			manifestAliasSpecs: []helperTagSpec{newHelperTagSpec(prefix, "pwsh", "", imageName, registryImage, build.RefTag())},
		}
		if build.IsLatest() {
			pwshBuildSet.manifestAliasSpecs = append(pwshBuildSet.manifestAliasSpecs, newHelperTagSpec(prefix, "pwsh", "", imageName, registryImage, "latest"))
		}
		pwshBuild := helperBuild{
			archive:    fmt.Sprintf("out/helper-images/prebuilt-%s-x86_64-pwsh.tar.xz", flavor),
			platform:   platformMap["x86_64"],
			tagSpec:    newHelperTagSpec(prefix, "pwsh", "x86_64", imageName, registryImage, build.Revision()),
			aliasSpecs: []helperTagSpec{newHelperTagSpec(prefix, "pwsh", "x86_64", imageName, registryImage, build.RefTag())},
		}
		if build.IsLatest() {
			pwshBuild.aliasSpecs = append(pwshBuild.aliasSpecs, newHelperTagSpec(prefix, "pwsh", "x86_64", imageName, registryImage, "latest"))
		}
		pwshBuildSet.componentBuilds = append(pwshBuildSet.componentBuilds, pwshBuild)
		builds.buildSets = append(builds.buildSets, pwshBuildSet)
	}

	return builds
}

func ReleaseHelper(blueprint helperBlueprint, publish bool) error {
	env := blueprint.Env()
	builder := docker.NewBuilder(
		env.Value(docker.Host),
		env.Value(docker.CertPath),
	)
	manifestTool := docker.NewManifestTool()

	logout, err := builder.Login(
		env.Value(ci.RegistryUser),
		env.Value(ci.RegistryPassword),
		env.Value(ci.Registry),
	)
	if err != nil {
		return err
	}
	defer logout()

	for _, build := range blueprint.Data() {
		if err := releaseImageTagSet(
			manifestTool,
			builder,
			build,
			publish,
		); err != nil {
			return err
		}
	}

	return nil
}

func releaseImageTagSet(manifestTool *docker.ManifestToolContext, builder *docker.Builder, buildSet helperBuildSet, publish bool) error {
	for _, build := range buildSet.componentBuilds {
		if err := releaseImageTags(
			builder,
			build,
			publish,
		); err != nil {
			return err
		}
	}

	specFile := fmt.Sprintf("out/helper-images/%s", buildSet.manifestTagSpec.renderSpecFileName())
	specContent, err := buildSet.renderManifestToolYaml()
	if err != nil {
		return err
	}
	if err := os.WriteFile(specFile, []byte(specContent), 0o644); err != nil {
		return err
	}

	if !publish {
		return nil
	}

	if err := manifestTool.Push("--type", buildSet.manifestType, "from-spec", specFile); err != nil {
		return err
	}
	// For manifest aliasing, manifest-tool pushes directly to the repo, without creating
	// the manifest in the local docker context. Additionally, docker pull is going to pull
	// the single appropriate image for the docker context, not the whole manifest.
	// For those reasons, on the manifest list side, we create manifest specs for each alias
	// individually
	for i, alias := range buildSet.manifestAliasSpecs {
		specFile := fmt.Sprintf("out/helper-images/%s", alias.renderSpecFileName())
		aliasManifest, err := buildSet.renderManifestToolAliasYaml(i)
		if err != nil {
			return err
		}
		if err := os.WriteFile(specFile, []byte(aliasManifest), 0o644); err != nil {
			return err
		}
		if err := manifestTool.Push("--type", buildSet.manifestType, "from-spec", specFile); err != nil {
			return err
		}
	}
	return nil
}

func releaseImageTags(builder *docker.Builder, build helperBuild, publish bool) error {
	baseTag := build.tagSpec.render()
	tagsToPush := []string{baseTag}

	if err := builder.Import(build.archive, baseTag, build.platform.String()); err != nil {
		return err
	}

	for _, alias := range build.aliasSpecs {
		aliasTag := alias.render()
		tagsToPush = append(tagsToPush, aliasTag)
		if err := builder.Tag(baseTag, aliasTag); err != nil {
			return err
		}
	}

	if !publish {
		return nil
	}

	for _, tag := range tagsToPush {
		if err := builder.Push(tag); err != nil {
			return err
		}
	}

	return nil
}
