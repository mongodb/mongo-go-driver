package main

import (
	"bufio"
	"fmt"
	"os"
	"path"
)

const (
	versionFile = "version/version.go"
)

func main() {
	repoDir, err := os.Getwd()
	versionFilePath := path.Join(repoDir, versionFile)
	readFile, err := os.Open(versionFilePath)

	if err != nil {
		fmt.Println(err)
	}
	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)
	var fileLines []string

	for fileScanner.Scan() {
		fileLines = append(fileLines, fileScanner.Text())
	}

	readFile.Close()

	fileLines[len(fileLines)-1] = "var Driver = \"" + os.Args[1] + "\""

	writeFile, err := os.OpenFile(versionFilePath, os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
	}
	for _, line := range fileLines {
		fmt.Fprintln(writeFile, line)
	}

	writeFile.Close()
}
