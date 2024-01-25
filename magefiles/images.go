//go:build mage

package main

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/magefile/mage/mg"
	"gitlab.com/gitlab-org/gitlab-runner/magefiles/build"
	"gitlab.com/gitlab-org/gitlab-runner/magefiles/images"
)

type Images mg.Namespace

// BuildRunnerDefault builds gitlab-runner images for ubuntu amd64 without pushing the resulting tags
func (Images) BuildRunnerDefault() error {
	return runRunnerBuild(images.DefaultFlavor, images.DefaultArchs, false)
}

// BuildRunnerHostArch builds gitlab-runner images for the host arch without pushing the resulting tags
func (Images) BuildRunnerHostArch(flavor string) error {
	return runRunnerBuild(flavor, runtime.GOARCH, false)
}

// BuildRunner builds gitlab-runner images for the specified flavor and target archs without pushing the resulting tags
func (Images) BuildRunner(flavor, targetArchs string) error {
	return runRunnerBuild(flavor, targetArchs, false)
}

// ReleaseRunner builds gitlab-runner images for the specified flavor and target archs and pushes the resulting
// tags to the configured repository
func (Images) ReleaseRunner(flavor, targetArchs string) error {
	return runRunnerBuild(flavor, targetArchs, true)
}

// ReleaseRunnerHostArch builds gitlab-runner images for the host arch and pushes the resulting tags to the configured repository
func (Images) ReleaseRunnerHostArch(flavor string) error {
	return runRunnerBuild(flavor, runtime.GOARCH, true)
}

func runRunnerBuild(flavor, targetArchs string, publish bool) error {
	blueprint, err := build.PrintBlueprint(images.AssembleBuildRunner(flavor, targetArchs))
	if err != nil {
		return err
	}

	artifactsFile := fmt.Sprintf("runner_images_%s_%s", flavor, strings.Join(strings.Split(targetArchs, ","), "_"))
	if err := build.Export(blueprint.Artifacts(), build.ReleaseArtifactsPath(artifactsFile)); err != nil {
		return err
	}

	return images.BuildRunner(blueprint, publish)
}

// TagHelperHostArch generates gitlab-runner-helper images tags for the specified flavor and host arch
func (Images) TagHelperHostArch(flavor string) error {
	return runHelperBuild(flavor, "", runtime.GOARCH, false)
}

// ReleaseHelper generates gitlab-runner-helper images tags from already generated image archives for the specified flavor.
// Prefix is automatically generated. The resulting tags are pushed to the configured repository
func (Images) ReleaseHelper(flavor string) error {
	return runHelperBuild(flavor, "", "", true)
}

// ReleaseHelperPrefix generates gitlab-runner-helper images tags from already generated image archives for the specified flavor and prefix
// and pushes the resulting tags to the configured repository
func (Images) ReleaseHelperPrefix(flavor, prefix string) error {
	return runHelperBuild(flavor, prefix, "", true)
}

func runHelperBuild(flavor, prefix, arch string, publish bool) error {
	blueprint, err := build.PrintBlueprint(images.AssembleReleaseHelper(flavor, prefix, arch))
	if err != nil {
		return err
	}

	artifactsFile := fmt.Sprintf("helper_images_%s_%s", flavor, prefix)
	if err := build.Export(blueprint.Artifacts(), build.ReleaseArtifactsPath(artifactsFile)); err != nil {
		return err
	}

	return images.ReleaseHelper(blueprint, publish)
}
