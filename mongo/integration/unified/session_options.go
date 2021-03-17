// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// transactionOptions is a wrapper for *options.transactionOptions. This type implements the bson.Unmarshaler interface
// to convert BSON documents to a transactionOptions instance.
type transactionOptions struct {
	*options.TransactionOptions
}

var _ bson.Unmarshaler = (*transactionOptions)(nil)

func (to *transactionOptions) UnmarshalBSON(data []byte) error {
	var temp struct {
		RC              *readConcern           `bson:"readConcern"`
		RP              *readPreference        `bson:"readPreference"`
		WC              *writeConcern          `bson:"writeConcern"`
		MaxCommitTimeMS *int64                 `bson:"maxCommitTimeMS"`
		Extra           map[string]interface{} `bson:",inline"`
	}
	if err := bson.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("error unmarshalling to temporary transactionOptions object: %v", err)
	}
	if len(temp.Extra) > 0 {
		return fmt.Errorf("unrecognized fields for transactionOptions: %v", mapKeys(temp.Extra))
	}

	to.TransactionOptions = options.Transaction()
	if temp.MaxCommitTimeMS != nil {
		mctms := time.Duration(*temp.MaxCommitTimeMS) * time.Millisecond
		to.SetMaxCommitTime(&mctms)
	}
	if rc := temp.RC; rc != nil {
		to.SetReadConcern(rc.toReadConcernOption())
	}
	if rp := temp.RP; rp != nil {
		converted, err := rp.toReadPrefOption()
		if err != nil {
			return fmt.Errorf("error parsing read preference document: %v", err)
		}
		to.SetReadPreference(converted)
	}
	if wc := temp.WC; wc != nil {
		converted, err := wc.toWriteConcernOption()
		if err != nil {
			return fmt.Errorf("error parsing write concern document: %v", err)
		}
		to.SetWriteConcern(converted)
	}
	return nil
}

// sessionOptions is a wrapper for *options.sessionOptions. This type implements the bson.Unmarshaler interface to
// convert BSON documents to a sessionOptions instance.
type sessionOptions struct {
	*options.SessionOptions
}

var _ bson.Unmarshaler = (*sessionOptions)(nil)

func (so *sessionOptions) UnmarshalBSON(data []byte) error {
	var temp struct {
		Causal          *bool                  `bson:"causalConsistency"`
		MaxCommitTimeMS *int64                 `bson:"maxCommitTimeMS"`
		TxnOptions      *transactionOptions    `bson:"defaultTransactionOptions"`
		Extra           map[string]interface{} `bson:",inline"`
	}
	if err := bson.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("error unmarshalling to temporary sessionOptions object: %v", err)
	}
	if len(temp.Extra) > 0 {
		return fmt.Errorf("unrecognized fields for sessionOptions: %v", mapKeys(temp.Extra))
	}

	so.SessionOptions = options.Session()
	if temp.Causal != nil {
		so.SetCausalConsistency(*temp.Causal)
	}
	if temp.MaxCommitTimeMS != nil {
		mctms := time.Duration(*temp.MaxCommitTimeMS) * time.Millisecond
		so.SetDefaultMaxCommitTime(&mctms)
	}
	if rc := temp.TxnOptions.ReadConcern; rc != nil {
		so.SetDefaultReadConcern(rc)
	}
	if rp := temp.TxnOptions.ReadPreference; rp != nil {
		so.SetDefaultReadPreference(rp)
	}
	if wc := temp.TxnOptions.WriteConcern; wc != nil {
		so.SetDefaultWriteConcern(wc)
	}
	return nil
}
