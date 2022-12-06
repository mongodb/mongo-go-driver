package logger

import (
	"io"
	"log"
)

type osSink struct {
	log *log.Logger
}

func newOSSink(out io.Writer) *osSink {
	return &osSink{
		log: log.New(out, "", log.LstdFlags),
	}
}

func (osSink *osSink) Info(_ int, msg string, _ ...interface{}) {
	osSink.log.Print(msg)
}
