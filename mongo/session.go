// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/mongo/aggregateopt"
	"github.com/mongodb/mongo-go-driver/mongo/countopt"
	"github.com/mongodb/mongo-go-driver/mongo/deleteopt"
	"github.com/mongodb/mongo-go-driver/mongo/distinctopt"
	"github.com/mongodb/mongo-go-driver/mongo/findopt"
	"github.com/mongodb/mongo-go-driver/mongo/insertopt"
	"github.com/mongodb/mongo-go-driver/mongo/replaceopt"
	"github.com/mongodb/mongo-go-driver/mongo/updateopt"
)

// Session represents a set of sequential operations executed by an application that are related in some way.
type Session struct {
	aggregateopt.AggregateSessionOpt
	countopt.CountSessionOpt
	deleteopt.DeleteSessionOpt
	distinctopt.DistinctSessionOpt
	findopt.FindSessionOpt
	updateopt.UpdateSessionOpt
	replaceopt.ReplaceSessionOpt
	insertopt.InsertSessionOpt
	Sess *session.Client
}

var (
	_ aggregateopt.Aggregate = (*Session)(nil)
	_ countopt.Count         = (*Session)(nil)
	_ deleteopt.Delete       = (*Session)(nil)
	_ distinctopt.Distinct   = (*Session)(nil)
	_ findopt.Find           = (*Session)(nil)
	_ findopt.One            = (*Session)(nil)
	_ findopt.UpdateOne      = (*Session)(nil)
	_ findopt.ReplaceOne     = (*Session)(nil)
	_ findopt.DeleteOne      = (*Session)(nil)
	_ insertopt.Many         = (*Session)(nil)
	_ insertopt.One          = (*Session)(nil)
	_ updateopt.Update       = (*Session)(nil)
	_ replaceopt.Replace     = (*Session)(nil)
)

// ConvertAggregateSession implements the AggregateSession interface.
func (s *Session) ConvertAggregateSession() *session.Client {
	return s.Sess
}

// ConvertCountSession implements the CountSession interface.
func (s *Session) ConvertCountSession() *session.Client {
	return s.Sess
}

// ConvertDeleteSession implements the DeleteSession interface.
func (s *Session) ConvertDeleteSession() *session.Client {
	return s.Sess
}

// ConvertDistinctSession implements the DistinctSession interface.
func (s *Session) ConvertDistinctSession() *session.Client {
	return s.Sess
}

// ConvertFindSession implements the FindSession interface.
func (s *Session) ConvertFindSession() *session.Client {
	return s.Sess
}

// ConvertFindOneSession implements the FindOneSession interface.
func (s *Session) ConvertFindOneSession() *session.Client {
	return s.Sess
}

// ConvertUpdateOneSession implements the UpdateOneSession interface.
func (s *Session) ConvertUpdateOneSession() *session.Client {
	return s.Sess
}

// ConvertReplaceOneSession implements the ReplaceOneSession interface.
func (s *Session) ConvertReplaceOneSession() *session.Client {
	return s.Sess
}

// ConvertDeleteOneSession implements the DeleteOneSession interface.
func (s *Session) ConvertDeleteOneSession() *session.Client {
	return s.Sess
}

// ConvertInsertSession implements the InsertManySession and InsertOneSession interfaces.
func (s *Session) ConvertInsertSession() *session.Client {
	return s.Sess
}

// ConvertUpdateSession implements the UpdateSession interface.
func (s *Session) ConvertUpdateSession() *session.Client {
	return s.Sess
}

// ConvertReplaceSession implements the ReplaceSession interface.
func (s *Session) ConvertReplaceSession() *session.Client {
	return s.Sess
}
