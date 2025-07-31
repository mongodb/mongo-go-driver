// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
)

const parsePerfCompDir = "./internal/cmd/perfcomp/parseperfcomp/"
const perfReportFileTxt = "perf-report.txt"
const perfReportFileMd = "perf-report.md"
const perfVariant = "^perf$"
const hscoreDefLink = "https://en.wikipedia.org/wiki/Energy_distance#:~:text=E%2Dcoefficient%20of%20inhomogeneity"
const zscoreDefLink = "https://en.wikipedia.org/wiki/Standard_score#Calculation"

func main() {
	var line string

	// open file to read
	fRead, err := os.Open(parsePerfCompDir + perfReportFileTxt)
	if err != nil {
		log.Fatalf("Could not open %s: %v", perfReportFileTxt, err)
	}
	defer fRead.Close()

	// open file to write
	fWrite, err := os.Create(perfReportFileMd)
	if err != nil {
		log.Fatalf("Could not create %s: %v", perfReportFileMd, err)
	}
	defer fWrite.Close()

	fmt.Fprintf(fWrite, "## 👋 GoDriver Performance\n")

	// read the file line by line using scanner
	scanner := bufio.NewScanner(fRead)

	var version string
	var evgLink string

	for scanner.Scan() {
		line = scanner.Text()
		if strings.Contains(line, "Version ID:") {
			// parse version
			version = strings.Split(line, " ")[2]
		} else if strings.Contains(line, "Commit SHA:") {
			// parse commit SHA and write header
			fmt.Fprintf(fWrite, "\n<details>\n<summary><b>%s</b></summary>\n\t<br>\n\n", line)
		} else if strings.Contains(line, "version "+version) {
			// dynamic Evergreen perf task link
			evgLink, err = generateEvgLink(version, perfVariant)
			if err != nil {
				log.Println(err)
				fmt.Fprintf(fWrite, "%s\n", line)
			} else {
				printUrlToLine(fWrite, line, evgLink, "version", -1)
			}
		} else if strings.Contains(line, "For a comprehensive view of all microbenchmark results for this PR's commit, please check out the Evergreen perf task for this patch.") {
			// last line of comment
			evgLink, err = generateEvgLink(version, "")
			if err != nil {
				log.Println(err)
				fmt.Fprintf(fWrite, "%s\n", line)
			} else {
				printUrlToLine(fWrite, line, evgLink, "Evergreen", 0)
			}
		} else if strings.Contains(line, ", ") {
			line = strings.ReplaceAll(line, ", ", "<br>")
			fmt.Fprintf(fWrite, "%s\n", line)
		} else if strings.Contains(line, "H-Score") {
			linkedWord := "[H-Score](" + hscoreDefLink + ")"
			line = strings.ReplaceAll(line, "H-Score", linkedWord)
			linkedWord = "[Z-Score](" + zscoreDefLink + ")"
			line = strings.ReplaceAll(line, "Z-Score", linkedWord)
			fmt.Fprintf(fWrite, "%s\n", line)
		} else {
			// all other regular lines
			fmt.Fprintf(fWrite, "%s\n", line)
		}
	}

	fmt.Fprintf(fWrite, "</details>\n")
}

func generateEvgLink(version string, variant string) (string, error) {
	baseUrl := "https://spruce.mongodb.com"
	page := "0"
	sorts := "STATUS:ASC;BASE_STATUS:DESC"

	u, err := url.Parse(baseUrl)
	if err != nil {
		return "", fmt.Errorf("Error parsing URL: %v", err)
	}

	u.Path = fmt.Sprintf("version/%s/tasks", version)

	// construct query parameters
	queryParams := url.Values{}
	queryParams.Add("page", page)
	queryParams.Add("sorts", sorts)
	if variant != "" {
		queryParams.Add("variant", variant)
	}

	u.RawQuery = queryParams.Encode()
	return u.String(), nil
}

func printUrlToLine(fWrite *os.File, line string, link string, targetWord string, step int) {
	words := strings.Split(line, " ")
	for i, w := range words {
		if i > 0 && words[i+step] == targetWord {
			fmt.Fprintf(fWrite, "[%s](%s)", w, link)
		} else {
			fmt.Fprint(fWrite, w)
		}

		if i < len(words)-1 {
			fmt.Fprint(fWrite, " ")
		} else {
			fmt.Fprint(fWrite, "\n")
		}
	}
}
