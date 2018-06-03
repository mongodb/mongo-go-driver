package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mongodb/mongo-go-driver/bson"
)

func main() {
	err := mainReal()
	if err != nil {
		os.Stderr.Write([]byte(err.Error()))
		os.Exit(-1)
	}
}

func mainReal() error {
	fileName := "-"

	flag.Parse()
	if flag.NArg() > 0 {
		fileName = flag.Arg(0)
	}

	var file *os.File
	var err error

	if fileName == "-" {
		file = os.Stdin
	} else {
		file, err = os.Open(fileName)
		if err != nil {
			return fmt.Errorf("cannot open file (%s) because: %s", fileName, err)
		}
		defer file.Close()
	}

	lineNumber := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineNumber++

		line := scanner.Text()
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		b, err := bson.ParseExtJSONObject(line)
		if err != nil {
			return fmt.Errorf("error parsing line %d: %s", lineNumber, err)
		}
		by, err := b.MarshalBSON()
		if err != nil {
			// this should be impossible
			panic(err)
		}

		n, err := os.Stdout.Write(by)
		if n != len(by) {
			return fmt.Errorf("error writing bson, only wrote %d of %d bytes", n, len(by))
		}
		if err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
