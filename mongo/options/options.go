// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
)

// Optioner is the interface implemented by types that can be used as options
// to a command.
type Optioner interface {
	Option(*bson.Document)
}

// FindOptioner is the interface implemented by types that can be used as
// Options for Find commands.
type FindOptioner interface {
	Optioner
	findOption()
}

// CountOptioner is the interface implemented by types that can be used as
// Options for Count commands.
type CountOptioner interface {
	Optioner
	countOption()
}

// DeleteOptioner is the interface implemented by types that can be used as
// Options for Delete commands.
type DeleteOptioner interface {
	Optioner
	deleteOption()
}

// InsertOptioner is the interface implemented by types that can be used as
// Options for Insert commands.
type InsertOptioner interface {
	Optioner
	insertOption()
}

// UpdateOptioner is the interface implemented by types that can be used as
// Options for Update commands.
type UpdateOptioner interface {
	Optioner
	updateOption()
}

// ReplaceOptioner is the interface implemented by types that can be used as
// Options for Update commands.
type ReplaceOptioner interface {
	UpdateOptioner
	replaceOption()
}

// DistinctOptioner is the interface implemented by types that can be used as
// Options for Distinct commands.
type DistinctOptioner interface {
	Optioner
	distinctOption()
}

// AggregateOptioner is the interface implemented by types that can be used
// as Options for an Aggregate command.
type AggregateOptioner interface {
	Optioner
	aggregateOption()
}

// InsertOneOptioner is the interface implemented by types that can be used as
// Options for InsertOne commands.
type InsertOneOptioner interface {
	InsertOptioner
	insertOneOption()
}

// InsertManyOptioner is the interface implemented by types that can be used as
// Options for InsertMany commands.
type InsertManyOptioner interface {
	InsertOptioner
	insertManyOption()
}

// FindOneAndDeleteOptioner is the interface implemented by types that can be
// used as Options for FindOneAndDelete commands.
type FindOneAndDeleteOptioner interface {
	Optioner
	findOneAndDeleteOption()
}

// FindOneAndUpdateOptioner is the interface implemented by types that can be
// used as Options for FindOneAndUpdate commands.
type FindOneAndUpdateOptioner interface {
	Optioner
	findOneAndUpdateOption()
}

// FindOneAndReplaceOptioner is the interface implemented by types that can be
// used as Options for FindOneAndReplace commands.
type FindOneAndReplaceOptioner interface {
	Optioner
	findOneAndReplaceOption()
}

var (
	_ AggregateOptioner         = (*OptAllowDiskUse)(nil)
	_ AggregateOptioner         = (*OptBatchSize)(nil)
	_ AggregateOptioner         = (*OptBypassDocumentValidation)(nil)
	_ AggregateOptioner         = (*OptCollation)(nil)
	_ AggregateOptioner         = (*OptComment)(nil)
	_ AggregateOptioner         = (*OptMaxTime)(nil)
	_ CountOptioner             = (*OptCollation)(nil)
	_ CountOptioner             = (*OptHint)(nil)
	_ CountOptioner             = (*OptLimit)(nil)
	_ CountOptioner             = (*OptMaxTime)(nil)
	_ CountOptioner             = (*OptSkip)(nil)
	_ DeleteOptioner            = (*OptCollation)(nil)
	_ DistinctOptioner          = (*OptCollation)(nil)
	_ DistinctOptioner          = (*OptMaxTime)(nil)
	_ FindOneAndDeleteOptioner  = (*OptCollation)(nil)
	_ FindOneAndDeleteOptioner  = (*OptMaxTime)(nil)
	_ FindOneAndDeleteOptioner  = (*OptProjection)(nil)
	_ FindOneAndDeleteOptioner  = (*OptSort)(nil)
	_ FindOneAndReplaceOptioner = (*OptBypassDocumentValidation)(nil)
	_ FindOneAndReplaceOptioner = (*OptCollation)(nil)
	_ FindOneAndReplaceOptioner = (*OptMaxTime)(nil)
	_ FindOneAndReplaceOptioner = (*OptProjection)(nil)
	_ FindOneAndReplaceOptioner = (*OptReturnDocument)(nil)
	_ FindOneAndReplaceOptioner = (*OptSort)(nil)
	_ FindOneAndReplaceOptioner = (*OptUpsert)(nil)
	_ FindOneAndUpdateOptioner  = (*OptArrayFilters)(nil)
	_ FindOneAndUpdateOptioner  = (*OptBypassDocumentValidation)(nil)
	_ FindOneAndUpdateOptioner  = (*OptCollation)(nil)
	_ FindOneAndUpdateOptioner  = (*OptMaxTime)(nil)
	_ FindOneAndUpdateOptioner  = (*OptProjection)(nil)
	_ FindOneAndUpdateOptioner  = (*OptReturnDocument)(nil)
	_ FindOneAndUpdateOptioner  = (*OptSort)(nil)
	_ FindOneAndUpdateOptioner  = (*OptUpsert)(nil)
	_ FindOptioner              = (*OptAllowPartialResults)(nil)
	_ FindOptioner              = (*OptBatchSize)(nil)
	_ FindOptioner              = (*OptCollation)(nil)
	_ FindOptioner              = OptCursorType(0)
	_ FindOptioner              = (*OptComment)(nil)
	_ FindOptioner              = (*OptHint)(nil)
	_ FindOptioner              = (*OptLimit)(nil)
	_ FindOptioner              = (*OptLimit)(nil)
	_ FindOptioner              = (*OptMaxAwaitTime)(nil)
	_ FindOptioner              = (*OptMaxScan)(nil)
	_ FindOptioner              = (*OptMaxTime)(nil)
	_ FindOptioner              = (*OptMin)(nil)
	_ FindOptioner              = (*OptNoCursorTimeout)(nil)
	_ FindOptioner              = (*OptOplogReplay)(nil)
	_ FindOptioner              = (*OptProjection)(nil)
	_ FindOptioner              = (*OptReturnKey)(nil)
	_ FindOptioner              = (*OptShowRecordID)(nil)
	_ FindOptioner              = (*OptSkip)(nil)
	_ FindOptioner              = (*OptSnapshot)(nil)
	_ FindOptioner              = (*OptSort)(nil)
	_ InsertManyOptioner        = (*OptBypassDocumentValidation)(nil)
	_ InsertManyOptioner        = (*OptOrdered)(nil)
	_ InsertOneOptioner         = (*OptBypassDocumentValidation)(nil)
	_ InsertOptioner            = (*OptBypassDocumentValidation)(nil)
	_ InsertOptioner            = (*OptOrdered)(nil)
	_ ReplaceOptioner           = (*OptBypassDocumentValidation)(nil)
	_ ReplaceOptioner           = (*OptCollation)(nil)
	_ ReplaceOptioner           = (*OptUpsert)(nil)
	_ UpdateOptioner            = (*OptUpsert)(nil)
	_ UpdateOptioner            = (*OptArrayFilters)(nil)
	_ UpdateOptioner            = (*OptBypassDocumentValidation)(nil)
	_ UpdateOptioner            = (*OptCollation)(nil)
)

// OptAllowDiskUse is for internal use.
type OptAllowDiskUse bool

// Option implements the Optioner interface.
func (opt OptAllowDiskUse) Option(d *bson.Document) {
	d.Append(bson.C.Boolean("allowDiskUse", bool(opt)))
}

func (opt OptAllowDiskUse) aggregateOption() {}

// OptAllowPartialResults is for internal use.
type OptAllowPartialResults bool

// Option implements the Optioner interface.
func (opt OptAllowPartialResults) Option(d *bson.Document) {
	d.Append(bson.C.Boolean("allowPartialResults", bool(opt)))
}

func (opt OptAllowPartialResults) findOption() {}

// OptArrayFilters is for internal use.
type OptArrayFilters []*bson.Document

// Option implements the Optioner interface.
func (opt OptArrayFilters) Option(d *bson.Document) {
	arr := bson.NewArray()
	for _, af := range opt {
		arr.Append(bson.AC.Document(af))
	}
	d.Append(bson.C.Array("arrayFilters", arr))
}

func (OptArrayFilters) findOneAndUpdateOption() {}
func (OptArrayFilters) updateOption()           {}

// OptBatchSize is for internal use.
type OptBatchSize int32

// Option implements the Optioner interface.
func (opt OptBatchSize) Option(d *bson.Document) {
	d.Append(bson.C.Int32("batchSize", int32(opt)))
}

func (OptBatchSize) aggregateOption() {}
func (OptBatchSize) findOption()      {}

// OptBypassDocumentValidation is for internal use.
type OptBypassDocumentValidation bool

// Option implements the Optioner interface.
func (opt OptBypassDocumentValidation) Option(d *bson.Document) {
	d.Append(bson.C.Boolean("bypassDocumentValidation", bool(opt)))
}

func (OptBypassDocumentValidation) aggregateOption()         {}
func (OptBypassDocumentValidation) findOneAndReplaceOption() {}
func (OptBypassDocumentValidation) findOneAndUpdateOption()  {}
func (OptBypassDocumentValidation) insertManyOption()        {}
func (OptBypassDocumentValidation) insertOption()            {}
func (OptBypassDocumentValidation) insertOneOption()         {}
func (OptBypassDocumentValidation) replaceOption()           {}
func (OptBypassDocumentValidation) updateOption()            {}

// OptCollation is for internal use.
type OptCollation struct{ Collation *CollationOptions }

// Option implements the Optioner interface.
func (opt OptCollation) Option(d *bson.Document) {
	d.Append(bson.C.SubDocument("collation", opt.Collation.toDocument()))
}

func (OptCollation) aggregateOption()         {}
func (OptCollation) countOption()             {}
func (OptCollation) deleteOption()            {}
func (OptCollation) distinctOption()          {}
func (OptCollation) findOption()              {}
func (OptCollation) findOneAndDeleteOption()  {}
func (OptCollation) findOneAndReplaceOption() {}
func (OptCollation) findOneAndUpdateOption()  {}
func (OptCollation) replaceOption()           {}
func (OptCollation) updateOption()            {}

// OptComment is for internal use.
type OptComment string

// Option implements the Optioner interface.
func (opt OptComment) Option(d *bson.Document) {
	d.Append(bson.C.String("comment", string(opt)))
}

func (OptComment) aggregateOption() {}
func (OptComment) findOption()      {}

// OptCursorType is for internal use.
type OptCursorType CursorType

// Option implements the Optioner interface.
func (opt OptCursorType) Option(d *bson.Document) {
	switch CursorType(opt) {
	case Tailable:
		d.Append(bson.C.Boolean("tailable", true))
	case TailableAwait:
		d.Append(bson.C.Boolean("tailable", true), bson.C.Boolean("awaitData", true))
	}
}

func (OptCursorType) findOption() {}

// OptHint is for internal use.
type OptHint struct{ Hint interface{} }

// Option implements the Optioner interface.
func (opt OptHint) Option(d *bson.Document) {
	switch t := (opt).Hint.(type) {
	case string:
		d.Append(bson.C.String("hint", t))
	case *bson.Document:
		d.Append(bson.C.SubDocument("hint", t))
	}
}

func (OptHint) countOption() {}
func (OptHint) findOption()  {}

// OptLimit is for internal use.
type OptLimit int64

// Option implements the Optioner interface.
func (opt OptLimit) Option(d *bson.Document) {
	d.Append(bson.C.Int64("limit", int64(opt)))
}

func (OptLimit) countOption() {}
func (OptLimit) findOption()  {}

// OptMax is for internal use.
type OptMax struct{ Max *bson.Document }

// Option implements the Optioner interface.
func (opt OptMax) Option(d *bson.Document) {
	d.Append(bson.C.SubDocument("max", opt.Max))
}

func (OptMax) findOption() {}

// OptMaxAwaitTime is for internal use.
type OptMaxAwaitTime time.Duration

// Option implements the Optioner interface.
func (opt OptMaxAwaitTime) Option(d *bson.Document) {
	d.Append(bson.C.Int64("maxAwaitTimeMS", int64(time.Duration(opt)/time.Millisecond)))
}

func (OptMaxAwaitTime) findOption() {}

// OptMaxScan is for internal use.
type OptMaxScan int64

// Option implements the Optioner interface.
func (opt OptMaxScan) Option(d *bson.Document) {
	d.Append(bson.C.Int64("maxScan", int64(opt)))
}

func (OptMaxScan) findOption() {}

// OptMaxTime is for internal use.
type OptMaxTime time.Duration

// Option implements the Optioner interface.
func (opt OptMaxTime) Option(d *bson.Document) {
	d.Append(bson.C.Int64("maxTimeMS", int64(time.Duration(opt)/time.Millisecond)))
}

func (OptMaxTime) aggregateOption()         {}
func (OptMaxTime) countOption()             {}
func (OptMaxTime) distinctOption()          {}
func (OptMaxTime) findOption()              {}
func (OptMaxTime) findOneAndDeleteOption()  {}
func (OptMaxTime) findOneAndReplaceOption() {}
func (OptMaxTime) findOneAndUpdateOption()  {}

// OptMin is for internal use.
type OptMin struct{ Min *bson.Document }

// Option implements the Optioner interface.
func (opt OptMin) Option(d *bson.Document) {
	d.Append(bson.C.SubDocument("min", opt.Min))
}

func (OptMin) findOption() {}

// OptNoCursorTimeout is for internal use.
type OptNoCursorTimeout bool

// Option implements the Optioner interface.
func (opt OptNoCursorTimeout) Option(d *bson.Document) {
	d.Append(bson.C.Boolean("noCursorTimeout", bool(opt)))
}

func (OptNoCursorTimeout) findOption() {}

// OptOplogReplay is for internal use.
type OptOplogReplay bool

// Option implements the Optioner interface.
func (opt OptOplogReplay) Option(d *bson.Document) {
	d.Append(bson.C.Boolean("oplogReplay", bool(opt)))
}

func (OptOplogReplay) findOption() {}

// OptOrdered is for internal use.
type OptOrdered bool

// Option implements the Optioner interface.
func (opt OptOrdered) Option(d *bson.Document) {
	d.Append(bson.C.Boolean("ordered", bool(opt)))
}

func (OptOrdered) insertManyOption() {}
func (OptOrdered) insertOption()     {}

// OptProjection is for internal use.
type OptProjection struct {
	Projection *bson.Document
	find       bool
}

// Option implements the Optioner interface.
func (opt OptProjection) Option(d *bson.Document) {
	var key = "fields"
	if opt.find {
		key = "projection"
	}
	d.Append(bson.C.SubDocument(key, opt.Projection))
}

// IsFind is for internal use.
func (opt OptProjection) IsFind() OptProjection {
	opt.find = true

	return opt
}

func (OptProjection) findOption()              {}
func (OptProjection) findOneAndDeleteOption()  {}
func (OptProjection) findOneAndReplaceOption() {}
func (OptProjection) findOneAndUpdateOption()  {}

// OptReturnDocument is for internal use.
type OptReturnDocument ReturnDocument

// Option implements the Optioner interface.
func (opt OptReturnDocument) Option(d *bson.Document) {
	d.Append(bson.C.Boolean("new", ReturnDocument(opt) == After))
}

func (OptReturnDocument) findOneAndReplaceOption() {}
func (OptReturnDocument) findOneAndUpdateOption()  {}

// OptReturnKey is for internal use.
type OptReturnKey bool

// Option implements the Optioner interface.
func (opt OptReturnKey) Option(d *bson.Document) {
	d.Append(bson.C.Boolean("returnKey", bool(opt)))
}

func (OptReturnKey) findOption() {}

// OptShowRecordID is for internal use.
type OptShowRecordID bool

// Option implements the Optioner interface.
func (opt OptShowRecordID) Option(d *bson.Document) {
	d.Append(bson.C.Boolean("showRecordId", bool(opt)))
}

func (OptShowRecordID) findOption() {}

// OptSkip is for internal use.
type OptSkip int64

// Option implements the Optioner interface.
func (opt OptSkip) Option(d *bson.Document) {
	d.Append(bson.C.Int64("skip", int64(opt)))
}

func (OptSkip) countOption() {}
func (OptSkip) findOption()  {}

// OptSnapshot is for internal use.
type OptSnapshot bool

// Option implements the Optioner interface.
func (opt OptSnapshot) Option(d *bson.Document) {
	d.Append(bson.C.Boolean("snapshot", bool(opt)))
}

func (OptSnapshot) findOption() {}

// OptSort is for internal use.
type OptSort struct{ Sort *bson.Document }

// Option implements the Optioner interface.
func (opt OptSort) Option(d *bson.Document) {
	d.Append(bson.C.SubDocument("sort", opt.Sort))
}

func (OptSort) findOption()              {}
func (OptSort) findOneAndDeleteOption()  {}
func (OptSort) findOneAndReplaceOption() {}
func (OptSort) findOneAndUpdateOption()  {}

// OptUpsert is for internal use.
type OptUpsert bool

// Option implements the Optioner interface.
func (opt OptUpsert) Option(d *bson.Document) {
	d.Append(bson.C.Boolean("upsert", bool(opt)))
}

func (OptUpsert) findOneAndReplaceOption() {}
func (OptUpsert) findOneAndUpdateOption()  {}
func (OptUpsert) replaceOption()           {}
func (OptUpsert) updateOption()            {}
