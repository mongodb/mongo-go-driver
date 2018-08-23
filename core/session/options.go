// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package session

import (
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
)

// ClientOptioner is the interface implemented by types that can be used as options for configuring a client session.
type ClientOptioner interface {
	Option(*Client) error
}

// OptCausalConsistency specifies if a session should be causally consistent.
type OptCausalConsistency bool

// Option implements the ClientOptioner interface.
func (opt OptCausalConsistency) Option(c *Client) error {
	c.Consistent = bool(opt)
	return nil
}

// OptDefaultReadConcern specifies the read concern that should be used for transactions started from this session.
type OptDefaultReadConcern struct {
	*readconcern.ReadConcern
}

// Option implements the ClientOptioner interface.
func (opt OptDefaultReadConcern) Option(c *Client) error {
	c.transactionRc = opt.ReadConcern
	return nil
}

// OptDefaultWriteConcern specifies the read concern that should be used for transactions started from this session.
type OptDefaultWriteConcern struct {
	*writeconcern.WriteConcern
}

// Option implements the ClientOptioner interface.
func (opt OptDefaultWriteConcern) Option(c *Client) error {
	c.transactionWc = opt.WriteConcern
	return nil
}

// OptDefaultReadPreference specifies the read concern that should be used for transactions started from this session.
type OptDefaultReadPreference struct {
	*readpref.ReadPref
}

// Option implements the ClientOptioner interface.
func (opt OptDefaultReadPreference) Option(c *Client) error {
	c.transactionRp = opt.ReadPref
	return nil
}

// OptCurrentReadConcern specifies the read concern to be used for the current transaction.
type OptCurrentReadConcern struct {
	*readconcern.ReadConcern
}

// Option implements the ClientOptioner interface.
func (opt OptCurrentReadConcern) Option(c *Client) error {
	c.CurrentRc = opt.ReadConcern
	return nil
}

// OptCurrentWriteConcern specifies the read concern to be used for the current transaction.
type OptCurrentWriteConcern struct {
	*writeconcern.WriteConcern
}

// Option implements the ClientOptioner interface.
func (opt OptCurrentWriteConcern) Option(c *Client) error {
	c.CurrentWc = opt.WriteConcern
	return nil
}

// OptCurrentReadPreference specifies the read concern to be used for the current transaction.
type OptCurrentReadPreference struct {
	*readpref.ReadPref
}

// Option implements the ClientOptioner interface.
func (opt OptCurrentReadPreference) Option(c *Client) error {
	c.CurrentRp = opt.ReadPref
	return nil
}
