// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	var line string
	var suppress bool
	var foundChange = false
	var foundSummary = false

	// open file to read
	fRead, err := os.Open("api-report.txt")
	if err != nil {
		log.Fatal(err)
	}
	// remember to close the file at the end of the program
	defer fRead.Close()

	// open file to write
	fWrite, err := os.Create("api-report.md")
	if err != nil {
		log.Fatal(err)
	}
	// remember to close the file at the end of the program
	defer fWrite.Close()

	fmt.Fprint(fWrite, "## API Change Report\n")

	// read the file line by line using scanner
	scanner := bufio.NewScanner(fRead)

	for scanner.Scan() {
		// do something with a line
		line = scanner.Text()
		if strings.Index(line, "## ") == 0 {
			line = "##" + line
		}

		if strings.Contains(line, "/mongo/integration/") {
			suppress = true
		}
		if strings.Index(line, "# summary") == 0 {
			suppress = true
			foundSummary = true
		}

		if strings.Contains(line, "go.mongodb.org/mongo-driver") {
			line = strings.Replace(line, "go.mongodb.org/mongo-driver", ".", -1)
			line = "##" + line
		}
		if !suppress {
			fmt.Fprintf(fWrite, "%s\n", line)
			foundChange = true
		}
		if len(line) == 0 {
			suppress = false
		}
	}

	if !foundChange {
		fmt.Fprint(fWrite, "No changes found!\n")
	}

	if !foundSummary {
		log.Fatal("Could not parse api summary")
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

}
