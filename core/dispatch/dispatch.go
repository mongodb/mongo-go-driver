// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

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
