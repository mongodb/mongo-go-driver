package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/bitfield/script"
	"github.com/coreos/go-semver/semver"
	"github.com/urfave/cli/v2"
)

const (
	versionFile      = "version/version.go"
	prereleaseSuffix = "-prerelease"
	remote           = "origin"
	minorBaseBranch  = "master"
)

func releaseMinor(version *semver.Version) {
	if version.Patch != 0 {
		log.Fatalf("Expected minor version to have patch version 0, but got %d", version.Patch)
	}

	// Make sure there are no changes, staged or unstaged.
	exec("git diff --quiet --exit-code")
	exec("git diff --quiet --exit-code --cached")

	if branch := strings.TrimSpace(exec("git branch --show-current")); branch != minorBaseBranch {
		log.Fatalf("Current branch must be %q, but is %q", minorBaseBranch, branch)
	}

	// Create the new release branch.
	releaseBranch := fmt.Sprintf("release/%d.%d", version.Major, version.Minor)
	execf("git checkout -b %s", releaseBranch)

	// Check that version.go has the expected prerelease version.
	prerelease := semver.New(version.String() + prereleaseSuffix)
	if must(script.File(versionFile).Match(prerelease.String()).String()) == "" {
		log.Fatalf(
			"Expected version/version.go to contain version %q, but it does not.",
			prerelease)
	}

	// Update the release version in version.go and commit the change.
	log.Printf("Updating version in %q from %q to %q", versionFile, prerelease, version)
	contents := must(script.File(versionFile).Replace(prerelease.String(), version.String()).String())
	must(script.Echo(contents).WriteFile(versionFile))
	execf("git add %s", versionFile)
	execf(`git commit -m "Update version to v%s"`, version.String())

	// Tag the release commit.
	releaseTag := fmt.Sprintf("v%s", version)
	execf(`git tag -s %[1]s -m "%[1]s"`, releaseTag)

	// Update the release version in version.go to the next patch prerelease tag
	// and commit the change.
	nextPatchPrerelease := semver.New(fmt.Sprintf(
		"%d.%d.%d%s",
		version.Major,
		version.Minor,
		version.Patch+1,
		prereleaseSuffix))
	log.Printf("Updating version in %q from %q to %q", versionFile, version, nextPatchPrerelease)
	contents = must(script.File(versionFile).Replace(version.String(), nextPatchPrerelease.String()).String())
	must(script.Echo(contents).WriteFile(versionFile))
	execf("git add %s", versionFile)
	execf(`git commit -m "Update version to v%s"`, nextPatchPrerelease)

	log.Printf(
		`Done! Run the following commands to persist the release to the remote:
	git push %[1]s %s
	git push %[1]s %s`,
		remote,
		releaseBranch,
		releaseTag)
}

func releasePatch(version *semver.Version) {
	if version.Patch == 0 {
		log.Fatal("Expected patch version to have non-zero patch version")
	}

	// Make sure there are no changes, staged or unstaged. Then check out the
	// appropriate release branch.
	exec("git diff --quiet --exit-code")
	exec("git diff --quiet --exit-code --cached")

	releaseBranch := fmt.Sprintf("release/%d.%d", version.Major, version.Minor)
	if branch := strings.TrimSpace(exec("git branch --show-current")); branch != releaseBranch {
		log.Fatalf("Current branch must be %q, but is %q", releaseBranch, branch)
	}

	// Check that version.go has the expected prerelease version.
	prerelease := semver.New(version.String() + prereleaseSuffix)
	if must(script.File(versionFile).Match(prerelease.String()).String()) == "" {
		log.Fatalf(
			"Expected version/version.go to contain version %q, but it does not.",
			prerelease)
	}

	// Update the release version in version.go and commit the change.
	log.Printf("Updating version in %q from %q to %q", versionFile, prerelease, version)
	contents := must(script.File(versionFile).Replace(prerelease.String(), version.String()).String())
	must(script.Echo(contents).WriteFile(versionFile))
	execf("git add %s", versionFile)
	execf(`git commit -m "Update version to v%s"`, version.String())

	// Tag the release commit.
	releaseTag := fmt.Sprintf("v%s", version)
	execf(`git tag -s %[1]s -m "%[1]s"`, releaseTag)

	// Update the release version in version.go to the next patch prerelease tag
	// and commit the change.
	nextPatchPrerelease := semver.New(fmt.Sprintf("%d.%d.%d%s", version.Major, version.Minor, version.Patch+1, prereleaseSuffix))
	log.Printf("Updating version in %q from %q to %q", versionFile, version, nextPatchPrerelease)
	contents = must(script.File(versionFile).Replace(version.String(), nextPatchPrerelease.String()).String())
	must(script.Echo(contents).WriteFile(versionFile))
	execf("git add %s", versionFile)
	execf(`git commit -m "Update version to v%s"`, nextPatchPrerelease)

	log.Printf(
		`Done! Run the following commands to persist the release to the remote:
	git push %[1]s %s
	git push %[1]s %s`,
		remote,
		releaseBranch,
		releaseTag)
}

// execf assembles and runs the given shell command and returns any printed
// shell output. It panics if the command exits with a non-zero exit code.
func execf(format string, a ...any) string {
	return exec(fmt.Sprintf(format, a...))
}

// exec runs the given shell command and returns any printed shell output. It
// panics if the command exits with a non-zero exit code.
func exec(command string) string {
	log.Print(command)
	out := must(script.Exec(command).String())
	if out != "" {
		log.Print(out)
	}
	return out
}

func must[T any](x T, err error) T {
	if err != nil {
		log.Fatal(err)
	}
	return x
}

func main() {
	app := &cli.App{
		Name:  "MongoDB Go Driver releaser.",
		Usage: "release patch v1.2.3",
		Commands: []*cli.Command{
			{
				Name:  "minor",
				Usage: "Release a minor version.",
				Action: func(c *cli.Context) error {
					version := strings.TrimSpace(c.Args().First())
					if version == "" {
						return errors.New("must specify a version to release")
					}

					// If the version string starts with "v", remove it because
					// the semver package expects version to only contain
					// numbers and periods. It will be re-added when appropriate
					// in ReleaseMinor.
					if version[0] == 'v' {
						version = version[1:]
					}

					releaseMinor(semver.New(version))
					return nil
				},
			},
			{
				Name:  "patch",
				Usage: "Release a patch version.",
				Flags: []cli.Flag{
					// &cli.StringFlag{
					// 	Name:  "main-branch",
					// 	Value: "master",
					// 	Usage: "the Git branch to release minor versions from",
					// },
				},
				Action: func(c *cli.Context) error {
					version := strings.TrimSpace(c.Args().First())
					if version == "" {
						return errors.New("must specify a version to release")
					}

					// If the version string starts with "v", remove it because
					// the semver package expects version to only contain
					// numbers and periods. It will be re-added when appropriate
					// in ReleasePatch.
					if version[0] == 'v' {
						version = version[1:]
					}

					releasePatch(semver.New(version))
					return nil
				},
			},
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
