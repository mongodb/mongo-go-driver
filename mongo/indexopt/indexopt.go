package indexopt

import (
	"time"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/core/session"
)

// BatchSize specifes the number of documents to return in each batch.
// List
func BatchSize(i int32) OptBatchSize {
	return OptBatchSize(i)
}

// MaxTime specifies the amount of time to allow the query to run.
// Create, Drop, List
func MaxTime(d time.Duration) OptMaxTime {
	return OptMaxTime(d)
}

// OptMaxTime specifies the maximum amount of time to allow the query to run.
type OptMaxTime option.OptMaxTime

func (OptMaxTime) create() {}
func (OptMaxTime) drop()   {}
func (OptMaxTime) list()   {}

// ConvertCreateOption implements the Create interface.
func (opt OptMaxTime) ConvertCreateOption() option.CreateIndexesOptioner {
	return option.OptMaxTime(opt)
}

// ConvertDropOption implements the Drop interface.
func (opt OptMaxTime) ConvertDropOption() option.DropIndexesOptioner {
	return option.OptMaxTime(opt)
}

// ConvertListOption implements the List interface.
func (opt OptMaxTime) ConvertListOption() option.ListIndexesOptioner {
	return option.OptMaxTime(opt)
}

// OptBatchSize specifies the number of documents to return in every batch.
type OptBatchSize option.OptBatchSize

func (OptBatchSize) list() {}

// ConvertListOption implements the List interface.
func (opt OptBatchSize) ConvertListOption() option.ListIndexesOptioner {
	return option.OptBatchSize(opt)
}

// IndexSessionOpt is an indexSession option.
type IndexSessionOpt struct{}

func (IndexSessionOpt) create() {}
func (IndexSessionOpt) drop()   {}
func (IndexSessionOpt) list()   {}

// ConvertIndexSession implements the DropIndexSession interface.
func (IndexSessionOpt) ConvertIndexSession() *session.Client {
	return nil
}
