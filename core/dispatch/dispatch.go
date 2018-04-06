package dispatch

import (
	"errors"

	"github.com/mongodb/mongo-go-driver/core/options"
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
)

// ErrUnacknowledgedWrite is returned from functions that have an unacknowledged
// write concern.
var ErrUnacknowledgedWrite = errors.New("unacknowledged write")

func writeConcernOption(wc *writeconcern.WriteConcern) (options.OptWriteConcern, error) {
	elem, err := wc.MarshalBSONElement()
	if err != nil {
		return options.OptWriteConcern{}, err
	}
	return options.OptWriteConcern{WriteConcern: elem, Acknowledged: wc.Acknowledged()}, nil
}

func readConcernOption(rc *readconcern.ReadConcern) (options.OptReadConcern, error) {
	elem, err := rc.MarshalBSONElement()
	if err != nil {
		return options.OptReadConcern{}, err
	}
	return options.OptReadConcern{ReadConcern: elem}, nil
}
