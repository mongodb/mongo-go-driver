package main

import (
	"flag"
	"fmt"
	"io"
	"os"

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

	for {
		rdr, err := bson.NewFromIOReader(file)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		doc, err := bson.ReadDocument(rdr)
		if err != nil {
			return err
		}

		fmt.Println(doc.ToExtJSON(true))
	}

	return nil
}
