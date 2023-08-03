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
	var found_change bool = false
	var found_summary bool = false

	// open file to read
	f_read, err := os.Open("api-report.txt")
	if err != nil {
		log.Fatal(err)
	}
	// remember to close the file at the end of the program
	defer f_read.Close()

	// open file to write
	f_write, err := os.Create("api-report.md")
	if err != nil {
		log.Fatal(err)
	}
	// remember to close the file at the end of the program
	defer f_write.Close()

	fmt.Fprint(f_write, "## API Change Report\n")

	// read the file line by line using scanner
	scanner := bufio.NewScanner(f_read)

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
			found_summary = true
		}

		if strings.Contains(line, "go.mongodb.org/mongo-driver") {
			line = strings.Replace(line, "go.mongodb.org/mongo-driver", ".", -1)
			line = "##" + line
		}
		if !suppress {
			fmt.Fprintf(f_write, "%s\n", line)
			found_change = true
		}
		if len(line) == 0 {
			suppress = false
		}
	}

	if !found_change {
		fmt.Fprint(f_write, "No changes found!\n")
	}

	if !found_summary {
		log.Fatal("Could not parse api summary")
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

}
