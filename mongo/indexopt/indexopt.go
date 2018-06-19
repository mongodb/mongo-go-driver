package indexopt

import (
	"time"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
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

// WriteConcern specifies a write concern.
// Create, Drop
func WriteConcern(wc *writeconcern.WriteConcern) OptWriteConcern {
	return OptWriteConcern{
		WriteConcern: wc,
	}
}

// OptWriteConcern specifies a write concern.
type OptWriteConcern option.OptWriteConcern

func (OptWriteConcern) create() {}
func (OptWriteConcern) drop()   {}

// ConvertCreateOption implements the Create interface.
func (opt OptWriteConcern) ConvertCreateOption() option.CreateIndexesOptioner {
	return option.OptWriteConcern{
		WriteConcern: opt.WriteConcern,
	}
}

// ConvertDropOption implements the Drop interface.
func (opt OptWriteConcern) ConvertDropOption() option.DropIndexesOptioner {
	return option.OptWriteConcern{
		WriteConcern: opt.WriteConcern,
	}
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
