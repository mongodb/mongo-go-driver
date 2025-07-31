// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/mongoutil"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// transactionOptions is a wrapper for *options.transactionOptions. This type implements the bson.Unmarshaler interface
// to convert BSON documents to a transactionOptions instance.
type transactionOptions struct {
	*options.TransactionOptionsBuilder
}

var _ bson.Unmarshaler = (*transactionOptions)(nil)

func (to *transactionOptions) UnmarshalBSON(data []byte) error {
	var temp struct {
		RC    *readConcern    `bson:"readConcern"`
		RP    *ReadPreference `bson:"readPreference"`
		WC    *writeConcern   `bson:"writeConcern"`
		Extra map[string]any  `bson:",inline"`
	}
	if err := bson.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("error unmarshalling to temporary transactionOptions object: %v", err)
	}
	if len(temp.Extra) > 0 {
		return fmt.Errorf("unrecognized fields for transactionOptions: %v", mapKeys(temp.Extra))
	}

	to.TransactionOptionsBuilder = options.Transaction()
	if rc := temp.RC; rc != nil {
		to.SetReadConcern(rc.toReadConcernOption())
	}
	if rp := temp.RP; rp != nil {
		converted, err := rp.ToReadPrefOption()
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
	*options.SessionOptionsBuilder
}

var _ bson.Unmarshaler = (*sessionOptions)(nil)

func (so *sessionOptions) UnmarshalBSON(data []byte) error {
	var temp struct {
		Causal     *bool               `bson:"causalConsistency"`
		TxnOptions *transactionOptions `bson:"defaultTransactionOptions"`
		Snapshot   *bool               `bson:"snapshot"`
		Extra      map[string]any      `bson:",inline"`
	}
	if err := bson.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("error unmarshalling to temporary sessionOptions object: %v", err)
	}
	if len(temp.Extra) > 0 {
		return fmt.Errorf("unrecognized fields for sessionOptions: %v", mapKeys(temp.Extra))
	}

	so.SessionOptionsBuilder = options.Session()
	if temp.Causal != nil {
		so.SetCausalConsistency(*temp.Causal)
	}
	if temp.TxnOptions != nil {
		txnArgs, err := mongoutil.NewOptions[options.TransactionOptions](temp.TxnOptions)
		if err != nil {
			return fmt.Errorf("failed to construct options from builder: %w", err)
		}

		txnOpts := options.Transaction()
		if rc := txnArgs.ReadConcern; rc != nil {
			txnOpts.SetReadConcern(rc)
		}
		if rp := txnArgs.ReadPreference; rp != nil {
			txnOpts.SetReadPreference(rp)
		}
		if wc := txnArgs.WriteConcern; wc != nil {
			txnOpts.SetWriteConcern(wc)
		}

		so.SetDefaultTransactionOptions(txnOpts)
	}
	if temp.Snapshot != nil {
		so.SetSnapshot(*temp.Snapshot)
	}
	return nil
}
