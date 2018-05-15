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
	Option(*bson.Document) error
}

// FindOptioner is the interface implemented by types that can be used as
// Options for Find commands.
type FindOptioner interface {
	Optioner
	findOption()
}

// FindOneOptioner is the interface implemented by types that can be used as
// Options for FindOne operations.
type FindOneOptioner interface {
	Optioner
	findOneOption()
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

// InsertOptioner is the interface implemented by types that can be used as
// Options for insert commands.
type InsertOptioner interface {
	Optioner
	insertOption()
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

// ChangeStreamOptioner is the interface implemented by types that can be used as Options for
// change stream operations.
type ChangeStreamOptioner interface {
	Optioner
	changeStreamOption()
}

// ListCollectionsOptioner is the interface implemented by types that can be used as
// Options for ListCollections operations.
type ListCollectionsOptioner interface {
	Optioner
	listCollectionsOption()
}

// ListDatabasesOptioner is the interface implemented by types that can be used as
// Options for ListDatabase operations.
type ListDatabasesOptioner interface {
	Optioner
	listDatabasesOption()
}

// CursorOptioner is the interface implemented by types that can be used as
// Options for Cursor operations.
type CursorOptioner interface {
	Optioner
	cursorOption()
}

//ListIndexesOptioner is the interface implemented by types that can be used as
// Options for list_indexes operations.
type ListIndexesOptioner interface {
	Optioner
	listIndexesOption()
}

//CreateIndexesOptioner is the interface implemented by types that can be used as
// Options for create_indexes operations.
type CreateIndexesOptioner interface {
	Optioner
	createIndexesOption()
}

//DropIndexesOptioner is the interface implemented by types that can be used as
// Options for drop_indexes operations.
type DropIndexesOptioner interface {
	Optioner
	dropIndexesOption()
}

var (
	_ AggregateOptioner         = (*OptAllowDiskUse)(nil)
	_ AggregateOptioner         = (*OptBatchSize)(nil)
	_ AggregateOptioner         = (*OptBypassDocumentValidation)(nil)
	_ AggregateOptioner         = (*OptCollation)(nil)
	_ AggregateOptioner         = (*OptComment)(nil)
	_ AggregateOptioner         = (*OptMaxTime)(nil)
	_ AggregateOptioner         = (*OptReadConcern)(nil)
	_ AggregateOptioner         = (*OptWriteConcern)(nil)
	_ CountOptioner             = (*OptCollation)(nil)
	_ CountOptioner             = (*OptHint)(nil)
	_ CountOptioner             = (*OptLimit)(nil)
	_ CountOptioner             = (*OptMaxTime)(nil)
	_ CountOptioner             = (*OptReadConcern)(nil)
	_ CountOptioner             = (*OptSkip)(nil)
	_ CursorOptioner            = OptBatchSize(0)
	_ DeleteOptioner            = (*OptCollation)(nil)
	_ DeleteOptioner            = (*OptWriteConcern)(nil)
	_ DistinctOptioner          = (*OptCollation)(nil)
	_ DistinctOptioner          = (*OptMaxTime)(nil)
	_ DistinctOptioner          = (*OptReadConcern)(nil)
	_ FindOneAndDeleteOptioner  = (*OptCollation)(nil)
	_ FindOneAndDeleteOptioner  = (*OptMaxTime)(nil)
	_ FindOneAndDeleteOptioner  = (*OptProjection)(nil)
	_ FindOneAndDeleteOptioner  = (*OptSort)(nil)
	_ FindOneAndDeleteOptioner  = (*OptWriteConcern)(nil)
	_ FindOneAndReplaceOptioner = (*OptBypassDocumentValidation)(nil)
	_ FindOneAndReplaceOptioner = (*OptCollation)(nil)
	_ FindOneAndReplaceOptioner = (*OptMaxTime)(nil)
	_ FindOneAndReplaceOptioner = (*OptProjection)(nil)
	_ FindOneAndReplaceOptioner = (*OptReturnDocument)(nil)
	_ FindOneAndReplaceOptioner = (*OptSort)(nil)
	_ FindOneAndReplaceOptioner = (*OptUpsert)(nil)
	_ FindOneAndReplaceOptioner = (*OptWriteConcern)(nil)
	_ FindOneAndUpdateOptioner  = (*OptArrayFilters)(nil)
	_ FindOneAndUpdateOptioner  = (*OptBypassDocumentValidation)(nil)
	_ FindOneAndUpdateOptioner  = (*OptCollation)(nil)
	_ FindOneAndUpdateOptioner  = (*OptMaxTime)(nil)
	_ FindOneAndUpdateOptioner  = (*OptProjection)(nil)
	_ FindOneAndUpdateOptioner  = (*OptReturnDocument)(nil)
	_ FindOneAndUpdateOptioner  = (*OptSort)(nil)
	_ FindOneAndUpdateOptioner  = (*OptUpsert)(nil)
	_ FindOneAndUpdateOptioner  = (*OptWriteConcern)(nil)
	_ FindOptioner              = (*OptAllowPartialResults)(nil)
	_ FindOptioner              = (*OptBatchSize)(nil)
	_ FindOptioner              = (*OptCollation)(nil)
	_ FindOptioner              = OptCursorType(0)
	_ FindOptioner              = (*OptComment)(nil)
	_ FindOptioner              = (*OptHint)(nil)
	_ FindOptioner              = (*OptLimit)(nil)
	_ FindOptioner              = (*OptMaxAwaitTime)(nil)
	_ FindOptioner              = (*OptMaxScan)(nil)
	_ FindOptioner              = (*OptMaxTime)(nil)
	_ FindOptioner              = (*OptMin)(nil)
	_ FindOptioner              = (*OptNoCursorTimeout)(nil)
	_ FindOptioner              = (*OptOplogReplay)(nil)
	_ FindOptioner              = (*OptProjection)(nil)
	_ FindOptioner              = (*OptReadConcern)(nil)
	_ FindOptioner              = (*OptReturnKey)(nil)
	_ FindOptioner              = (*OptShowRecordID)(nil)
	_ FindOptioner              = (*OptSkip)(nil)
	_ FindOptioner              = (*OptSnapshot)(nil)
	_ FindOptioner              = (*OptSort)(nil)
	_ FindOneOptioner           = (*OptAllowPartialResults)(nil)
	_ FindOneOptioner           = (*OptBatchSize)(nil)
	_ FindOneOptioner           = (*OptCollation)(nil)
	_ FindOneOptioner           = OptCursorType(0)
	_ FindOneOptioner           = (*OptComment)(nil)
	_ FindOneOptioner           = (*OptHint)(nil)
	_ FindOneOptioner           = (*OptMaxAwaitTime)(nil)
	_ FindOneOptioner           = (*OptMaxScan)(nil)
	_ FindOneOptioner           = (*OptMaxTime)(nil)
	_ FindOneOptioner           = (*OptMin)(nil)
	_ FindOneOptioner           = (*OptNoCursorTimeout)(nil)
	_ FindOneOptioner           = (*OptOplogReplay)(nil)
	_ FindOneOptioner           = (*OptProjection)(nil)
	_ FindOneOptioner           = (*OptReadConcern)(nil)
	_ FindOneOptioner           = (*OptReturnKey)(nil)
	_ FindOneOptioner           = (*OptShowRecordID)(nil)
	_ FindOneOptioner           = (*OptSkip)(nil)
	_ FindOneOptioner           = (*OptSnapshot)(nil)
	_ FindOneOptioner           = (*OptSort)(nil)
	_ InsertManyOptioner        = (*OptBypassDocumentValidation)(nil)
	_ InsertManyOptioner        = (*OptOrdered)(nil)
	_ InsertManyOptioner        = (*OptWriteConcern)(nil)
	_ InsertOneOptioner         = (*OptBypassDocumentValidation)(nil)
	_ InsertOneOptioner         = (*OptWriteConcern)(nil)
	_ InsertOptioner            = (*OptBypassDocumentValidation)(nil)
	_ InsertOptioner            = (*OptOrdered)(nil)
	_ InsertOptioner            = (*OptWriteConcern)(nil)
	_ ListDatabasesOptioner     = OptNameOnly(false)
	_ ListIndexesOptioner       = OptBatchSize(0)
	_ ReplaceOptioner           = (*OptBypassDocumentValidation)(nil)
	_ ReplaceOptioner           = (*OptCollation)(nil)
	_ ReplaceOptioner           = (*OptUpsert)(nil)
	_ ReplaceOptioner           = (*OptWriteConcern)(nil)
	_ UpdateOptioner            = (*OptUpsert)(nil)
	_ UpdateOptioner            = (*OptArrayFilters)(nil)
	_ UpdateOptioner            = (*OptBypassDocumentValidation)(nil)
	_ UpdateOptioner            = (*OptCollation)(nil)
	_ UpdateOptioner            = (*OptWriteConcern)(nil)
	_ ChangeStreamOptioner      = (*OptBatchSize)(nil)
	_ ChangeStreamOptioner      = (*OptCollation)(nil)
	_ ChangeStreamOptioner      = (*OptFullDocument)(nil)
	_ ChangeStreamOptioner      = (*OptMaxAwaitTime)(nil)
	_ ChangeStreamOptioner      = (*OptReadConcern)(nil)
	_ ChangeStreamOptioner      = (*OptResumeAfter)(nil)
)

// OptAllowDiskUse is for internal use.
type OptAllowDiskUse bool

// Option implements the Optioner interface.
func (opt OptAllowDiskUse) Option(d *bson.Document) error {
	d.Append(bson.EC.Boolean("allowDiskUse", bool(opt)))
	return nil
}

func (opt OptAllowDiskUse) aggregateOption() {}

// OptAllowPartialResults is for internal use.
type OptAllowPartialResults bool

// Option implements the Optioner interface.
func (opt OptAllowPartialResults) Option(d *bson.Document) error {
	d.Append(bson.EC.Boolean("allowPartialResults", bool(opt)))
	return nil
}

func (opt OptAllowPartialResults) findOption()    {}
func (opt OptAllowPartialResults) findOneOption() {}

// OptArrayFilters is for internal use.
type OptArrayFilters []*bson.Document

// Option implements the Optioner interface.
func (opt OptArrayFilters) Option(d *bson.Document) error {
	arr := bson.NewArray()
	for _, af := range opt {
		arr.Append(bson.VC.Document(af))
	}
	d.Append(bson.EC.Array("arrayFilters", arr))
	return nil
}

func (OptArrayFilters) findOneAndUpdateOption() {}
func (OptArrayFilters) updateOption()           {}

// OptBatchSize is for internal use.
type OptBatchSize int32

// Option implements the Optioner interface.
func (opt OptBatchSize) Option(d *bson.Document) error {
	d.Append(bson.EC.Int32("batchSize", int32(opt)))
	return nil
}

func (OptBatchSize) aggregateOption()    {}
func (OptBatchSize) changeStreamOption() {}
func (OptBatchSize) findOption()         {}
func (OptBatchSize) findOneOption()      {}
func (OptBatchSize) listIndexesOption()  {}
func (OptBatchSize) cursorOption()       {}

// OptBypassDocumentValidation is for internal use.
type OptBypassDocumentValidation bool

// Option implements the Optioner interface.
func (opt OptBypassDocumentValidation) Option(d *bson.Document) error {
	d.Append(bson.EC.Boolean("bypassDocumentValidation", bool(opt)))
	return nil
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
type OptCollation struct{ Collation *Collation }

// Option implements the Optioner interface.
func (opt OptCollation) Option(d *bson.Document) error {
	d.Append(bson.EC.SubDocument("collation", opt.Collation.toDocument()))
	return nil
}

func (OptCollation) aggregateOption()         {}
func (OptCollation) changeStreamOption()      {}
func (OptCollation) countOption()             {}
func (OptCollation) deleteOption()            {}
func (OptCollation) distinctOption()          {}
func (OptCollation) findOption()              {}
func (OptCollation) findOneOption()           {}
func (OptCollation) findOneAndDeleteOption()  {}
func (OptCollation) findOneAndReplaceOption() {}
func (OptCollation) findOneAndUpdateOption()  {}
func (OptCollation) replaceOption()           {}
func (OptCollation) updateOption()            {}

// OptComment is for internal use.
type OptComment string

// Option implements the Optioner interface.
func (opt OptComment) Option(d *bson.Document) error {
	d.Append(bson.EC.String("comment", string(opt)))
	return nil
}

func (OptComment) aggregateOption() {}
func (OptComment) findOption()      {}
func (OptComment) findOneOption()   {}

// OptCursorType is for internal use.
type OptCursorType CursorType

// Option implements the Optioner interface.
func (opt OptCursorType) Option(d *bson.Document) error {
	switch CursorType(opt) {
	case Tailable:
		d.Append(bson.EC.Boolean("tailable", true))
	case TailableAwait:
		d.Append(bson.EC.Boolean("tailable", true), bson.EC.Boolean("awaitData", true))
	}
	return nil
}

func (OptCursorType) findOption()    {}
func (OptCursorType) findOneOption() {}

// OptFullDocument is for internal use.
type OptFullDocument string

// Option implements the Optioner interface.
func (opt OptFullDocument) Option(d *bson.Document) error {
	d.Append(bson.EC.String("fullDocument", string(opt)))
	return nil
}

func (OptFullDocument) changeStreamOption() {}

// OptHint is for internal use.
type OptHint struct{ Hint interface{} }

// Option implements the Optioner interface.
func (opt OptHint) Option(d *bson.Document) error {
	switch t := (opt).Hint.(type) {
	case string:
		d.Append(bson.EC.String("hint", t))
	case *bson.Document:
		d.Append(bson.EC.SubDocument("hint", t))
	}
	return nil
}

func (OptHint) countOption()   {}
func (OptHint) findOption()    {}
func (OptHint) findOneOption() {}

// OptLimit is for internal use.
type OptLimit int64

// Option implements the Optioner interface.
func (opt OptLimit) Option(d *bson.Document) error {
	d.Append(bson.EC.Int64("limit", int64(opt)))
	return nil
}

func (OptLimit) countOption() {}
func (OptLimit) findOption()  {}

// OptMax is for internal use.
type OptMax struct{ Max *bson.Document }

// Option implements the Optioner interface.
func (opt OptMax) Option(d *bson.Document) error {
	d.Append(bson.EC.SubDocument("max", opt.Max))
	return nil
}

func (OptMax) findOption()    {}
func (OptMax) findOneOption() {}

// OptMaxAwaitTime is for internal use.
type OptMaxAwaitTime time.Duration

// Option implements the Optioner interface.
func (opt OptMaxAwaitTime) Option(d *bson.Document) error {
	d.Append(bson.EC.Int64("maxAwaitTimeMS", int64(time.Duration(opt)/time.Millisecond)))
	return nil
}

func (OptMaxAwaitTime) changeStreamOption() {}
func (OptMaxAwaitTime) findOption()         {}
func (OptMaxAwaitTime) findOneOption()      {}

// OptMaxScan is for internal use.
type OptMaxScan int64

// Option implements the Optioner interface.
func (opt OptMaxScan) Option(d *bson.Document) error {
	d.Append(bson.EC.Int64("maxScan", int64(opt)))
	return nil
}

func (OptMaxScan) findOption()    {}
func (OptMaxScan) findOneOption() {}

// OptMaxTime is for internal use.
type OptMaxTime time.Duration

// Option implements the Optioner interface.
func (opt OptMaxTime) Option(d *bson.Document) error {
	d.Append(bson.EC.Int64("maxTimeMS", int64(time.Duration(opt)/time.Millisecond)))
	return nil
}

func (OptMaxTime) aggregateOption()         {}
func (OptMaxTime) countOption()             {}
func (OptMaxTime) distinctOption()          {}
func (OptMaxTime) findOption()              {}
func (OptMaxTime) findOneOption()           {}
func (OptMaxTime) findOneAndDeleteOption()  {}
func (OptMaxTime) findOneAndReplaceOption() {}
func (OptMaxTime) findOneAndUpdateOption()  {}
func (OptMaxTime) listIndexesOption()       {}
func (OptMaxTime) dropIndexesOption()       {}
func (OptMaxTime) createIndexesOption()     {}

// OptMin is for internal use.
type OptMin struct{ Min *bson.Document }

// Option implements the Optioner interface.
func (opt OptMin) Option(d *bson.Document) error {
	d.Append(bson.EC.SubDocument("min", opt.Min))
	return nil
}

func (OptMin) findOption()    {}
func (OptMin) findOneOption() {}

// OptNoCursorTimeout is for internal use.
type OptNoCursorTimeout bool

// Option implements the Optioner interface.
func (opt OptNoCursorTimeout) Option(d *bson.Document) error {
	d.Append(bson.EC.Boolean("noCursorTimeout", bool(opt)))
	return nil
}

func (OptNoCursorTimeout) findOption()    {}
func (OptNoCursorTimeout) findOneOption() {}

// OptOplogReplay is for internal use.
type OptOplogReplay bool

// Option implements the Optioner interface.
func (opt OptOplogReplay) Option(d *bson.Document) error {
	d.Append(bson.EC.Boolean("oplogReplay", bool(opt)))
	return nil
}

func (OptOplogReplay) findOption()    {}
func (OptOplogReplay) findOneOption() {}

// OptOrdered is for internal use.
type OptOrdered bool

// Option implements the Optioner interface.
func (opt OptOrdered) Option(d *bson.Document) error {
	d.Append(bson.EC.Boolean("ordered", bool(opt)))
	return nil
}

func (OptOrdered) insertManyOption() {}
func (OptOrdered) insertOption()     {}

// OptProjection is for internal use.
type OptProjection struct {
	Projection *bson.Document
	find       bool
}

// Option implements the Optioner interface.
func (opt OptProjection) Option(d *bson.Document) error {
	var key = "fields"
	if opt.find {
		key = "projection"
	}
	d.Append(bson.EC.SubDocument(key, opt.Projection))
	return nil
}

// IsFind is for internal use.
func (opt OptProjection) IsFind() OptProjection {
	opt.find = true

	return opt
}

func (OptProjection) findOption()              {}
func (OptProjection) findOneOption()           {}
func (OptProjection) findOneAndDeleteOption()  {}
func (OptProjection) findOneAndReplaceOption() {}
func (OptProjection) findOneAndUpdateOption()  {}

// OptReadConcern is for internal use.
type OptReadConcern struct{ ReadConcern *bson.Element }

// Option implements the Optioner interface.
func (opt OptReadConcern) Option(d *bson.Document) error {
	if _, err := d.Lookup(opt.ReadConcern.Key()); err == bson.ErrElementNotFound {
		d.Append(opt.ReadConcern)
	}
	return nil
}

func (OptReadConcern) aggregateOption()    {}
func (OptReadConcern) changeStreamOption() {}
func (OptReadConcern) countOption()        {}
func (OptReadConcern) distinctOption()     {}
func (OptReadConcern) findOption()         {}
func (OptReadConcern) findOneOption()      {}

// OptResumeAfter is for internal use.
type OptResumeAfter struct{ ResumeAfter *bson.Document }

// Option implements the Optioner interface.
func (opt OptResumeAfter) Option(d *bson.Document) error {
	if opt.ResumeAfter != nil {
		d.Append(bson.EC.SubDocument("resumeAfter", opt.ResumeAfter))
	}
	return nil
}

func (OptResumeAfter) changeStreamOption() {}

// OptReturnDocument is for internal use.
type OptReturnDocument ReturnDocument

// Option implements the Optioner interface.
func (opt OptReturnDocument) Option(d *bson.Document) error {
	d.Append(bson.EC.Boolean("new", ReturnDocument(opt) == After))
	return nil
}

func (OptReturnDocument) findOneAndReplaceOption() {}
func (OptReturnDocument) findOneAndUpdateOption()  {}

// OptReturnKey is for internal use.
type OptReturnKey bool

// Option implements the Optioner interface.
func (opt OptReturnKey) Option(d *bson.Document) error {
	d.Append(bson.EC.Boolean("returnKey", bool(opt)))
	return nil
}

func (OptReturnKey) findOption()    {}
func (OptReturnKey) findOneOption() {}

// OptShowRecordID is for internal use.
type OptShowRecordID bool

// Option implements the Optioner interface.
func (opt OptShowRecordID) Option(d *bson.Document) error {
	d.Append(bson.EC.Boolean("showRecordId", bool(opt)))
	return nil
}

func (OptShowRecordID) findOption()    {}
func (OptShowRecordID) findOneOption() {}

// OptSkip is for internal use.
type OptSkip int64

// Option implements the Optioner interface.
func (opt OptSkip) Option(d *bson.Document) error {
	d.Append(bson.EC.Int64("skip", int64(opt)))
	return nil
}

func (OptSkip) countOption()   {}
func (OptSkip) findOption()    {}
func (OptSkip) findOneOption() {}

// OptSnapshot is for internal use.
type OptSnapshot bool

// Option implements the Optioner interface.
func (opt OptSnapshot) Option(d *bson.Document) error {
	d.Append(bson.EC.Boolean("snapshot", bool(opt)))
	return nil
}

func (OptSnapshot) findOption()    {}
func (OptSnapshot) findOneOption() {}

// OptSort is for internal use.
type OptSort struct{ Sort *bson.Document }

// Option implements the Optioner interface.
func (opt OptSort) Option(d *bson.Document) error {
	d.Append(bson.EC.SubDocument("sort", opt.Sort))
	return nil
}

func (OptSort) findOption()              {}
func (OptSort) findOneOption()           {}
func (OptSort) findOneAndDeleteOption()  {}
func (OptSort) findOneAndReplaceOption() {}
func (OptSort) findOneAndUpdateOption()  {}

// OptUpsert is for internal use.
type OptUpsert bool

// Option implements the Optioner interface.
func (opt OptUpsert) Option(d *bson.Document) error {
	d.Append(bson.EC.Boolean("upsert", bool(opt)))
	return nil
}

func (OptUpsert) findOneAndReplaceOption() {}
func (OptUpsert) findOneAndUpdateOption()  {}
func (OptUpsert) replaceOption()           {}
func (OptUpsert) updateOption()            {}

// OptWriteConcern is for internal use.
type OptWriteConcern struct {
	WriteConcern *bson.Element
	Acknowledged bool
}

// Option implements the Optioner interface.
func (opt OptWriteConcern) Option(d *bson.Document) error {
	_, err := d.Lookup(opt.WriteConcern.Key())
	if err == bson.ErrElementNotFound {
		d.Append(opt.WriteConcern)
		return nil
	}
	return err
}

func (OptWriteConcern) aggregateOption()         {}
func (OptWriteConcern) deleteOption()            {}
func (OptWriteConcern) findOneAndUpdateOption()  {}
func (OptWriteConcern) findOneAndReplaceOption() {}
func (OptWriteConcern) findOneAndDeleteOption()  {}
func (OptWriteConcern) insertOption()            {}
func (OptWriteConcern) insertManyOption()        {}
func (OptWriteConcern) insertOneOption()         {}
func (OptWriteConcern) replaceOption()           {}
func (OptWriteConcern) updateOption()            {}
func (OptWriteConcern) createIndexesOption()     {}
func (OptWriteConcern) dropIndexesOption()       {}

// OptNameOnly is for internal use.
type OptNameOnly bool

// Option implements the Optioner interface.
func (opt OptNameOnly) Option(d *bson.Document) error {
	d.Append(bson.EC.Boolean("nameOnly", bool(opt)))
	return nil
}

func (OptNameOnly) listDatabasesOption() {}
