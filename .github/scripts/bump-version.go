package main

import (
	"fmt"
	"os"
	"path"
)

const (
	versionFile = "version/version.go"
	prefix      = `// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package version defines the Go Driver version.
package version // import "go.mongodb.org/mongo-driver/version"

// Driver is the current version of the driver.
`
)

func main() {
	repoDir, err := os.Getwd()
	versionFilePath := path.Join(repoDir, versionFile)

	writeFile, err := os.OpenFile(versionFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println(err)
	}
	text := prefix + "var Driver = \"" + os.Args[1] + "\"\n"
	fmt.Fprint(writeFile, text)
	writeFile.Close()
}
