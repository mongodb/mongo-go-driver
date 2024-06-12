package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"text/template"
)

//go:embed github.md.tmpl
var githubTmpl string

//go:embed forum.md.tmpl
var forumTmpl string

// generateGithubNotes generates a partially filled out release notes template
// for a Github release and writes it to a file named "github.md". It
// also prints the gh command to create a new draft release with those release
// notes.
func generateGithubNotes(release, previous string) {
	filename := "github.md"

	writeTemplate(
		filename,
		githubTmpl,
		map[string]any{
			"ReleaseVersion":  release,
			"PreviousVersion": previous,
		})

	fmt.Println()
	fmt.Print(
		`Wrote Github notes template to "`,
		filename,
		`".`,
		"\n")
	fmt.Println("Fill out any missing information and run the following command to create the Github release:")
	fmt.Println()
	fmt.Println(
		"\tgh auth refresh && gh release create v%[1]s --verify-tag --draft -R 'mongodb/mongo-go-driver' -t 'MongoDB Go Driver %[1]s' -F '%[2]s'",
		release,
		filename)
}

// generateForumNotes generates a partially filled out release notes template
// for a MongoDB community forum release post and writes it to a file named
// "forum.md".
func generateForumNotes(version string) {
	data := map[string]any{
		"ReleaseVersion": version,
	}

	forumFilename := "forum.md"
	writeTemplate(
		forumFilename,
		forumTmpl,
		data)

	fmt.Println()
	fmt.Print(
		`Wrote MongoDB community forum notes template to "`,
		forumFilename,
		`".`,
		"\n")
	fmt.Println("Fill out any missing information and paste the contents into a new MongoDB community forum post in section:")
	fmt.Println("https://www.mongodb.com/community/forums/c/announcements/driver-releases/110")
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
