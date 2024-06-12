package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"text/template"

	"github.com/coreos/go-semver/semver"
	"github.com/fatih/color"
)

//go:embed github.md.tmpl
var githubTmpl string

//go:embed forum.md.tmpl
var forumTmpl string

// generateGithubNotes generates a partially filled out release notes template
// for a Github release and writes it to a file named "github.md". It
// also prints the gh command to create a new draft release with those release
// notes.
func generateGithubNotes(release, previous *semver.Version) {
	filename := fmt.Sprintf("github.md", release)

	writeTemplate(
		filename,
		githubTmpl,
		map[string]any{
			"ReleaseVersion":  release.String(),
			"PreviousVersion": previous.String(),
		})

	fmt.Println()
	fmt.Print(
		color.BlueString(`Wrote Github notes template to "`),
		color.GreenString(filename),
		color.BlueString(`".`),
		"\n")
	color.Blue("Fill out any missing information and run the following command to create the Github release:")
	fmt.Println()
	color.Green(
		"\tgh auth refresh && gh release create v%[1]s --verify-tag --draft -R 'mongodb/mongo-go-driver' -t 'MongoDB Go Driver %[1]s' -F '%[2]s'",
		release,
		filename)
}

// generateForumNotes generates a partially filled out release notes template
// for a MongoDB community forum release post and writes it to a file named
// "forum.md".
func generateForumNotes(version *semver.Version) {
	data := map[string]any{
		"ReleaseVersion": version.String(),
	}

	forumFilename := fmt.Sprintf("forum.md", version)
	writeTemplate(
		forumFilename,
		forumTmpl,
		data)

	fmt.Println()
	fmt.Print(
		color.BlueString(`Wrote MongoDB community forum notes template to "`),
		color.GreenString(forumFilename),
		color.BlueString(`".`),
		"\n")
	color.Blue("Fill out any missing information and paste the contents into a new MongoDB community forum post in section:")
	color.Blue("https://www.mongodb.com/community/forums/c/announcements/driver-releases/110")
}

func writeTemplate(filename, tmplText string, data any) {
	tmpl, err := template.New(filename).Parse(tmplText)
	if err != nil {
		log.Fatalf("Error creating new template for %q: %v", filename, err)
	}

	f, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Error creating file %q: %v", filename, err)
	}
	defer f.Close()

	err = tmpl.Execute(f, data)
	if err != nil {
		log.Fatalf("Error executing template for %q: %v", filename, err)
	}
}

func main() {
	version := os.Args[1]
	prevVersion := os.Args[2]
	generateGithubNotes(version, prevVersion)
	generateForumNotes(version)
}
