package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/klauspost/compress/s2"
)

var (
	safe   = flag.Bool("safe", false, "Do not overwrite output files")
	stdout = flag.Bool("c", false, "Write all output to stdout. Multiple input files will be concatenated")
	remove = flag.Bool("rm", false, "Delete source file(s) after successful decompression")
	quiet  = flag.Bool("q", false, "Don't write any output to terminal, except errors")
	bench  = flag.Int("bench", 0, "Run benchmark n times. No output will be written")
	help   = flag.Bool("help", false, "Display help")
)

func main() {
	flag.Parse()
	r := s2.NewReader(nil)

	// No args, use stdin/stdout
	args := flag.Args()
	if len(args) == 0 || *help {
		_, _ = fmt.Fprintln(os.Stderr, `Usage: s2d [options] file1 file2

Decompresses all files supplied as input. Input files must end with '.s2' or '.snappy'.
Output file names have the extension removed. By default output files will be overwritten.
Use - as the only file name to read from stdin and write to stdout.

Wildcards are accepted: testdir/*.txt will compress all files in testdir ending with .txt
Directories can be wildcards as well. testdir/*/*.txt will match testdir/subdir/b.txt

Options:`)
		flag.PrintDefaults()
	}
	if len(args) == 1 && args[0] == "-" {
		r.Reset(os.Stdin)
		_, err := io.Copy(os.Stdout, r)
		exitErr(err)
		return
	}
	var files []string

	for _, pattern := range args {
		found, err := filepath.Glob(pattern)
		exitErr(err)
		if len(found) == 0 {
			exitErr(fmt.Errorf("unable to find file %v", pattern))
		}
		files = append(files, found...)
	}

	*quiet = *quiet || *stdout
	allFiles := files
	for i := 0; i < *bench; i++ {
		files = append(files, allFiles...)
	}

	for _, filename := range files {
		dstFilename := filename
		switch {
		case strings.HasSuffix(filename, ".s2"):
			dstFilename = strings.TrimSuffix(filename, ".s2")
		case strings.HasSuffix(filename, ".snappy"):
			dstFilename = strings.TrimSuffix(filename, ".snappy")
		default:
			fmt.Println("Skipping", filename)
			continue
		}

		func() {
			var closeOnce sync.Once
			if !*quiet {
				fmt.Println("Decompressing", filename, "->", dstFilename)
			}
			// Input file.
			file, err := os.Open(filename)
			exitErr(err)
			defer closeOnce.Do(func() { file.Close() })
			rc := rCounter{in: file}
			src := bufio.NewReaderSize(&rc, 4<<20)
			finfo, err := file.Stat()
			exitErr(err)
			mode := finfo.Mode() // use the same mode for the output file
			if *safe {
				_, err := os.Stat(dstFilename)
				if !os.IsNotExist(err) {
					exitErr(errors.New("destination files exists"))
				}
			}
			var out io.Writer
			switch {
			case *bench > 0:
				out = ioutil.Discard
			case *stdout:
				out = os.Stdout
			default:
				dstFile, err := os.OpenFile(dstFilename, os.O_CREATE|os.O_WRONLY, mode)
				exitErr(err)
				defer dstFile.Close()
				bw := bufio.NewWriterSize(dstFile, 4<<20)
				defer bw.Flush()
				out = bw
			}
			r.Reset(src)
			start := time.Now()
			output, err := io.Copy(out, r)
			exitErr(err)
			if !*quiet {
				elapsed := time.Since(start)
				mbPerSec := (float64(output) / (1024 * 1024)) / (float64(elapsed) / (float64(time.Second)))
				pct := float64(output) * 100 / float64(rc.n)
				fmt.Printf("%d -> %d [%.02f%%]; %.01fMB/s\n", rc.n, output, pct, mbPerSec)
			}
			if *remove {
				closeOnce.Do(func() {
					file.Close()
					err := os.Remove(filename)
					exitErr(err)
				})
			}
		}()
	}
}

func exitErr(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err.Error())
		os.Exit(2)
	}
}

type rCounter struct {
	n  int
	in io.Reader
}

func (w *rCounter) Read(p []byte) (n int, err error) {
	n, err = w.in.Read(p)
	w.n += n
	return n, err

}
