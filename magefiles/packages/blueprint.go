package packages

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/samber/lo"
	"gitlab.com/gitlab-org/gitlab-runner/magefiles/build"
	"gitlab.com/gitlab-org/gitlab-runner/magefiles/env"
)

const (
	Deb     Type = "deb"
	DebSlim Type = "deb-slim"
	Rpm     Type = "rpm"
	RpmSlim Type = "rpm-slim"
	RpmFips Type = "rpm-fips"
)

var (
	gPGKeyID      = env.New("GPG_KEYID")
	gPGPassphrase = env.New("GPG_PASSPHRASE")
	iteration     = env.New(iterationVar)
)

// translatePackageType translates various package types into their bare type which can then be passed
// to fpm to create the package.
func translatePackageType(p Type) Type {
	switch p {
	case DebSlim:
		return Deb
	case RpmSlim, RpmFips:
		return Rpm
	default:
		return p
	}
}

func postfix(p Type) string {
	if p == RpmFips {
		return "-fips"
	}

	return ""
}

type Blueprint = build.TargetBlueprint[build.Component, build.Component, blueprintParams]

type blueprintImpl struct {
	build.BlueprintBase

	fileDependencies                 []string
	osBinaryDependencies             []string
	prebuiltImageArchiveDependencies []string
	macOSDependencies                []build.Component

	artifacts []string
	params    blueprintParams
}

type blueprintParams struct {
	pkgType        Type
	packageArch    string
	postfix        string
	runnerBinary   string
	pkgFile        string
	prebuiltImages []string
}

func (b blueprintImpl) Dependencies() []build.Component {
	fileDeps := lo.Map(b.fileDependencies, func(s string, _ int) build.Component {
		return build.NewFile(s).WithRequired()
	})

	binDeps := lo.Map(b.osBinaryDependencies, func(s string, _ int) build.Component {
		return build.NewOSBinary(s).WithRequired()
	})

	imageDebs := lo.Map(b.prebuiltImageArchiveDependencies, func(s string, _ int) build.Component {
		return build.NewDockerImageArchive(s).WithRequired()
	})

	var deps []build.Component
	deps = append(deps, fileDeps...)
	deps = append(deps, binDeps...)
	deps = append(deps, imageDebs...)
	deps = append(deps, b.macOSDependencies...)

	return deps
}

func (b blueprintImpl) Artifacts() []build.Component {
	return lo.Map(b.artifacts, func(s string, _ int) build.Component {
		return build.NewFile(s)
	})
}

func (b blueprintImpl) Data() blueprintParams {
	return b.params
}

func Assemble(pkgType Type, arch, packageArch string) Blueprint {
	base := build.NewBlueprintBase(gPGKeyID, gPGPassphrase, iteration)

	runnerBinary := fmt.Sprintf("out/binaries/%s-linux-%s", build.AppName, arch)

	pkgName := build.AppName
	pkgFile := fmt.Sprintf("out/%s/%s_%s%s.%s", pkgType, pkgName, packageArch, postfix(pkgType), pkgType)

	prebuiltImages := prebuiltImages(pkgType, arch)

	params := blueprintParams{
		pkgType:        translatePackageType(pkgType),
		packageArch:    packageArch,
		postfix:        postfix(pkgType),
		runnerBinary:   runnerBinary,
		pkgFile:        pkgFile,
		prebuiltImages: prebuiltImages,
	}

	fileDependencies, osBinaryDependencies, imagesDependencies, macosDependencies := assembleDependencies(params, base.Env())

	return blueprintImpl{
		BlueprintBase: base,

		fileDependencies:                 fileDependencies,
		osBinaryDependencies:             osBinaryDependencies,
		prebuiltImageArchiveDependencies: imagesDependencies,
		macOSDependencies:                macosDependencies,

		artifacts: []string{pkgFile},

		params: params,
	}
}

func assembleDependencies(p blueprintParams, env build.BlueprintEnv) ([]string, []string, []string, []build.Component) {
	fileDependencies := []string{p.runnerBinary}

	binaryDependencies := []string{"fpm"}

	if env.Value(gPGKeyID) != "" {
		switch p.pkgType {
		case Deb:
			binaryDependencies = append(binaryDependencies, "dpkg-sig", "gpg")
		case Rpm, RpmFips:
			binaryDependencies = append(binaryDependencies, "rpm", "gpg")
		}
	}

	imagesDependencies := lo.Map(p.prebuiltImages, func(s string, _ int) string {
		return strings.Split(s, "=")[0]
	})

	var macosDependencies []build.Component
	if runtime.GOOS == "darwin" {
		macosDependencies = append(macosDependencies,
			build.NewMacOSPackage("gtar").WithDescription("from the brew package gnu-tar").WithRequired(),
			build.NewMacOSPackage("rpmbuild").WithDescription("from the brew package rpm").WithRequired(),
		)
	}

	return fileDependencies, binaryDependencies, imagesDependencies, macosDependencies
}

func prebuiltImages(t Type, archFilter string) []string {
	const (
		baseHelperInputPart  = "out/helper-images/prebuilt-"
		baseHelperOutputPart = "/usr/lib/gitlab-runner/helper-images/prebuilt-"
	)

	if t == RpmFips {
		return []string{
			fmt.Sprintf("%subi-fips-x86_64.tar.xz=%subi-fips-x86_64.tar.xz", baseHelperInputPart, baseHelperOutputPart),
		}
	}

	if archFilter == "amd64" {
		archFilter = "x86_64"
	}

	suffixes := []string{
		"alpine-arm.tar.xz",
		"alpine-arm64.tar.xz",
		"alpine-edge-riscv64.tar.xz",
		"alpine-s390x.tar.xz",
		"alpine-x86_64-pwsh.tar.xz",
		"alpine-x86_64.tar.xz",
		"ubuntu-arm.tar.xz",
		"ubuntu-arm64.tar.xz",
		"ubuntu-ppc64le.tar.xz",
		"ubuntu-s390x.tar.xz",
		"ubuntu-x86_64-pwsh.tar.xz",
		"ubuntu-x86_64.tar.xz",
	}

	if archFilter != "" {
		suffixes = lo.Filter(suffixes, func(s string, _ int) bool {
			if t == DebSlim || t == RpmSlim {
				return strings.Contains(s, archFilter) && strings.Contains(s, "alpine")
			}

			return strings.Contains(s, archFilter)
		})
	}

	return lo.Map(suffixes, func(s string, _ int) string {
		return fmt.Sprintf("%s=%s", baseHelperInputPart+s, baseHelperOutputPart+s)
	})
}
