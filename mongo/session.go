// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"errors"

	"context"

	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/dispatch"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/core/topology"
	"github.com/mongodb/mongo-go-driver/mongo/aggregateopt"
	"github.com/mongodb/mongo-go-driver/mongo/changestreamopt"
	"github.com/mongodb/mongo-go-driver/mongo/countopt"
	"github.com/mongodb/mongo-go-driver/mongo/dbopt"
	"github.com/mongodb/mongo-go-driver/mongo/deleteopt"
	"github.com/mongodb/mongo-go-driver/mongo/distinctopt"
	"github.com/mongodb/mongo-go-driver/mongo/dropcollopt"
	"github.com/mongodb/mongo-go-driver/mongo/findopt"
	"github.com/mongodb/mongo-go-driver/mongo/indexopt"
	"github.com/mongodb/mongo-go-driver/mongo/insertopt"
	"github.com/mongodb/mongo-go-driver/mongo/listcollectionopt"
	"github.com/mongodb/mongo-go-driver/mongo/listdbopt"
	"github.com/mongodb/mongo-go-driver/mongo/replaceopt"
	"github.com/mongodb/mongo-go-driver/mongo/runcmdopt"
	"github.com/mongodb/mongo-go-driver/mongo/transactionopt"
	"github.com/mongodb/mongo-go-driver/mongo/updateopt"
)

// ErrWrongClient is returned when a user attempts to pass in a session created by a different client than
// the method call is using.
var ErrWrongClient = errors.New("session was not created by this client")

// Session represents a set of sequential operations executed by an application that are related in some way.
type Session struct {
	aggregateopt.AggregateSessionOpt
	changestreamopt.ChangeStreamSessionOpt
	countopt.CountSessionOpt
	deleteopt.DeleteSessionOpt
	distinctopt.DistinctSessionOpt
	dbopt.DropDBSessionOpt
	findopt.FindSessionOpt
	dropcollopt.DropCollSessionOpt
	listcollectionopt.ListCollectionsSessionOpt
	updateopt.UpdateSessionOpt
	replaceopt.ReplaceSessionOpt
	indexopt.IndexSessionOpt
	insertopt.InsertSessionOpt
	runcmdopt.RunCmdSessionOpt
	listdbopt.ListDatabasesSessionOpt
	*session.Client
	topo                *topology.Topology
	didCommitAfterStart bool // true if commit was called after start with no other operations
}

var (
	_ aggregateopt.Aggregate            = (*Session)(nil)
	_ countopt.Count                    = (*Session)(nil)
	_ changestreamopt.ChangeStream      = (*Session)(nil)
	_ deleteopt.Delete                  = (*Session)(nil)
	_ distinctopt.Distinct              = (*Session)(nil)
	_ dbopt.DropDB                      = (*Session)(nil)
	_ findopt.Find                      = (*Session)(nil)
	_ findopt.One                       = (*Session)(nil)
	_ findopt.UpdateOne                 = (*Session)(nil)
	_ findopt.ReplaceOne                = (*Session)(nil)
	_ findopt.DeleteOne                 = (*Session)(nil)
	_ indexopt.Create                   = (*Session)(nil)
	_ indexopt.Drop                     = (*Session)(nil)
	_ indexopt.List                     = (*Session)(nil)
	_ dropcollopt.DropColl              = (*Session)(nil)
	_ listcollectionopt.ListCollections = (*Session)(nil)
	_ insertopt.Many                    = (*Session)(nil)
	_ insertopt.One                     = (*Session)(nil)
	_ updateopt.Update                  = (*Session)(nil)
	_ replaceopt.Replace                = (*Session)(nil)
	_ runcmdopt.Option                  = (*Session)(nil)
	_ listdbopt.ListDatabases           = (*Session)(nil)
)

// EndSession ends the session.
func (s *Session) EndSession(ctx context.Context) {
	if s.TransactionInProgress() {
		// ignore all errors aborting during an end session
		_ = s.AbortTransaction(ctx)
	}
	s.Client.EndSession()
}

// StartTransaction starts a transaction for this session.
func (s *Session) StartTransaction(opts ...transactionopt.Transaction) error {
	err := s.CheckStartTransaction()
	if err != nil {
		return err
	}

	s.didCommitAfterStart = false

	tranOpts, err := transactionopt.BundleTransaction(opts...).Unbundle(true)
	if err != nil {
		return err
	}

	return s.Client.StartTransaction(tranOpts...)
}

// CommitTransaction commits the sesson's transaction.
func (s *Session) CommitTransaction(ctx context.Context) error {
	err := s.CheckCommitTransaction()
	if err != nil {
		return err
	}

	// Do not run the commit command if the transaction is in started state
	if s.TransactionStarting() || s.didCommitAfterStart {
		s.didCommitAfterStart = true
		return s.Client.CommitTransaction()
	}

	if s.Client.TransactionCommitted() {
		s.RetryingCommit = true
	}

	cmd := command.CommitTransaction{
		Session: s.Client,
	}

	// Hack to ensure that session stays in committed state
	if s.TransactionCommitted() {
		s.Committing = true
		defer func() {
			s.Committing = false
		}()
	}
	_, err = dispatch.CommitTransaction(ctx, cmd, s.topo, description.WriteSelector())
	if err == nil {
		return s.Client.CommitTransaction()
	}
	return err
}

// AbortTransaction aborts the session's transaction, returning any errors and error codes
func (s *Session) AbortTransaction(ctx context.Context) error {
	err := s.CheckAbortTransaction()
	if err != nil {
		return err
	}

	cmd := command.AbortTransaction{
		Session: s.Client,
	}

	s.Aborting = true
	_, err = dispatch.AbortTransaction(ctx, cmd, s.topo, description.WriteSelector())

	_ = s.Client.AbortTransaction()
	return err
}

// ConvertAggregateSession implements the AggregateSession interface.
func (s *Session) ConvertAggregateSession() *session.Client {
	return s.Client
}

// ConvertChangeStreamSession implements the ChangeStreamSession interface.
func (s *Session) ConvertChangeStreamSession() *session.Client {
	return s.Client
}

// ConvertCountSession implements the CountSession interface.
func (s *Session) ConvertCountSession() *session.Client {
	return s.Client
}

// ConvertDeleteSession implements the DeleteSession interface.
func (s *Session) ConvertDeleteSession() *session.Client {
	return s.Client
}

// ConvertDropDBSession implements the DropDBSession interface.
func (s *Session) ConvertDropDBSession() *session.Client {
	return s.Client
}

// ConvertDropCollSession implements the DropCollSession interface
func (s *Session) ConvertDropCollSession() *session.Client {
	return s.Client
}

// ConvertDistinctSession implements the DistinctSession interface.
func (s *Session) ConvertDistinctSession() *session.Client {
	return s.Client
}

// ConvertFindSession implements the FindSession interface.
func (s *Session) ConvertFindSession() *session.Client {
	return s.Client
}

// ConvertFindOneSession implements the FindOneSession interface.
func (s *Session) ConvertFindOneSession() *session.Client {
	return s.Client
}

// ConvertListCollectionsSession implements the ListCollectionsSession interface.
func (s *Session) ConvertListCollectionsSession() *session.Client {
	return s.Client
}

// ConvertUpdateOneSession implements the UpdateOneSession interface.
func (s *Session) ConvertUpdateOneSession() *session.Client {
	return s.Client
}

// ConvertReplaceOneSession implements the ReplaceOneSession interface.
func (s *Session) ConvertReplaceOneSession() *session.Client {
	return s.Client
}

// ConvertDeleteOneSession implements the DeleteOneSession interface.
func (s *Session) ConvertDeleteOneSession() *session.Client {
	return s.Client
}

// ConvertIndexSession implements the IndexSession interface.
func (s *Session) ConvertIndexSession() *session.Client {
	return s.Client
}

// ConvertInsertSession implements the InsertManySession and InsertOneSession interfaces.
func (s *Session) ConvertInsertSession() *session.Client {
	return s.Client
}

// ConvertUpdateSession implements the UpdateSession interface.
func (s *Session) ConvertUpdateSession() *session.Client {
	return s.Client
}

// ConvertReplaceSession implements the ReplaceSession interface.
func (s *Session) ConvertReplaceSession() *session.Client {
	return s.Client
}

// ConvertRunCmdSession implements the RunCmdSession interface.
func (s *Session) ConvertRunCmdSession() *session.Client {
	return s.Client
}

// ConvertListDatabasesSession implements the ListDatabasesSession interface
func (s *Session) ConvertListDatabasesSession() *session.Client {
	return s.Client
}
