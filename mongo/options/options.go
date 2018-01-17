package options

import (
	"time"

	"github.com/skriptble/wilson/bson"
)

// Optioner is the interface implemented by types that can be used as options
// to a command.
type Optioner interface {
	Option(*bson.Document)
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
	_ UpdateOptioner            = (*OptArrayFilters)(nil)
	_ UpdateOptioner            = (*OptBypassDocumentValidation)(nil)
	_ UpdateOptioner            = (*OptCollation)(nil)
	_ UpdateOptioner            = (*OptUpsert)(nil)
)

// OptAllowDiskUse is for internal use.
type OptAllowDiskUse bool

// AggregateName is for internal use.
func (opt OptAllowDiskUse) AggregateName() string {
	return "allowDiskUse"
}

// AggregateValue is for internal use.
func (opt OptAllowDiskUse) AggregateValue() interface{} {
	return opt
}

// Option implements the Optioner interface.
func (opt OptAllowDiskUse) Option(d *bson.Document) {
	d.Append(bson.C.Boolean("allowDiskUse", bool(opt)))
}

func (opt OptAllowDiskUse) aggregateOption() {}

// OptAllowPartialResults is for internal use.
type OptAllowPartialResults bool

// FindName is for internal use.
func (opt OptAllowPartialResults) FindName() string {
	return "allowPartialResults"
}

// FindValue is for internal use.
func (opt OptAllowPartialResults) FindValue() interface{} {
	return opt
}

// Option implements the Optioner interface.
func (opt OptAllowPartialResults) Option(d *bson.Document) {
	d.Append(bson.C.Boolean("allowPartialResults", bool(opt)))
}

func (opt OptAllowPartialResults) findOption() {}

// OptArrayFilters is for internal use.
//
// TODO(skriptble): This type should be []*bson.Document.
type OptArrayFilters []interface{}

// FindOneAndUpdateName is for internal use.
func (opt OptArrayFilters) FindOneAndUpdateName() string {
	return "arrayFilters"
}

// FindOneAndUpdateValue is for internal use.
func (opt OptArrayFilters) FindOneAndUpdateValue() interface{} {
	return opt
}

// UpdateName is for internal use.
func (opt OptArrayFilters) UpdateName() string {
	return "arrayFilters"
}

// UpdateValue is for internal use.
func (opt OptArrayFilters) UpdateValue() interface{} {
	return opt
}

// Option implements the Optioner interface.
func (opt OptArrayFilters) Option(d *bson.Document) {
	arr := bson.NewArray()
	for _, af := range opt {
		doc, ok := af.(*bson.Document)
		if !ok {
			continue
		}
		arr.Append(bson.AC.Document(doc))
	}
	d.Append(bson.C.Array("arrayFilters", arr))
}

func (OptArrayFilters) findOneAndUpdateOption() {}
func (OptArrayFilters) updateOption()           {}

// OptBatchSize is for internal use.
type OptBatchSize int32

// AggregateName is for internal use.
func (opt OptBatchSize) AggregateName() string {
	return "batchSize"
}

// AggregateValue is for internal use.
func (opt OptBatchSize) AggregateValue() interface{} {
	return opt
}

// FindName is for internal use.
func (opt OptBatchSize) FindName() string {
	return "batchSize"
}

// FindValue is for internal use.
func (opt OptBatchSize) FindValue() interface{} {
	return opt
}

// Option implements the Optioner interface.
func (opt OptBatchSize) Option(d *bson.Document) {
	d.Append(bson.C.Int32("batchSize", int32(opt)))
}

func (OptBatchSize) aggregateOption() {}
func (OptBatchSize) findOption()      {}

// OptBypassDocumentValidation is for internal use.
type OptBypassDocumentValidation bool

// AggregateName is for internal use.
func (opt OptBypassDocumentValidation) AggregateName() string {
	return "bypassDocumentValidation"
}

// AggregateValue is for internal use.
func (opt OptBypassDocumentValidation) AggregateValue() interface{} {
	return opt
}

// FindOneAndUpdateName is for internal use.
func (opt OptBypassDocumentValidation) FindOneAndUpdateName() string {
	return "bypassDocumentValidation"
}

// FindOneAndUpdateValue is for internal use.
func (opt OptBypassDocumentValidation) FindOneAndUpdateValue() interface{} {
	return opt
}

// FindOneAndReplaceName is for internal use.
func (opt OptBypassDocumentValidation) FindOneAndReplaceName() string {
	return "bypassDocumentValidation"
}

// FindOneAndReplaceValue is for internal use.
func (opt OptBypassDocumentValidation) FindOneAndReplaceValue() interface{} {
	return opt
}

// InsertManyName is for internal use.
func (opt OptBypassDocumentValidation) InsertManyName() string {
	return "bypassDocumentValidation"
}

// InsertManyValue is for internal use.
func (opt OptBypassDocumentValidation) InsertManyValue() interface{} {
	return opt
}

// InsertName is for internal use.
func (opt OptBypassDocumentValidation) InsertName() string {
	return "bypassDocumentValidation"
}

// InsertValue is for internal use.
func (opt OptBypassDocumentValidation) InsertValue() interface{} {
	return opt
}

// InsertOneName is for internal use.
func (opt OptBypassDocumentValidation) InsertOneName() string {
	return "bypassDocumentValidation"
}

// InsertOneValue is for internal use.
func (opt OptBypassDocumentValidation) InsertOneValue() interface{} {
	return opt
}

// UpdateName is for internal use.
func (opt OptBypassDocumentValidation) UpdateName() string {
	return "bypassDocumentValidation"
}

// UpdateValue is for internal use.
func (opt OptBypassDocumentValidation) UpdateValue() interface{} {
	return opt
}

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
func (OptBypassDocumentValidation) updateOption()            {}

// OptCollation is for internal use.
type OptCollation struct{ Collation *CollationOptions }

// AggregateName is for internal use.
func (opt OptCollation) AggregateName() string {
	return "collation"
}

// AggregateValue is for internal use.
func (opt OptCollation) AggregateValue() interface{} {
	return opt.Collation
}

// CountName is for internal use.
func (opt OptCollation) CountName() string {
	return "collation"
}

// CountValue is for internal use.
func (opt OptCollation) CountValue() interface{} {
	return opt.Collation
}

// DeleteName is for internal use.
func (opt OptCollation) DeleteName() string {
	return "collation"
}

// DeleteValue is for internal use.
func (opt OptCollation) DeleteValue() interface{} {
	return opt.Collation
}

// DistinctName is for internal use.
func (opt OptCollation) DistinctName() string {
	return "collation"
}

// DistinctValue is for internal use.
func (opt OptCollation) DistinctValue() interface{} {
	return opt.Collation
}

// FindName is for internal use.
func (opt OptCollation) FindName() string {
	return "collation"
}

// FindValue is for internal use.
func (opt OptCollation) FindValue() interface{} {
	return opt.Collation
}

// FindOneAndReplaceName is for internal use.
func (opt OptCollation) FindOneAndReplaceName() string {
	return "collation"
}

// FindOneAndReplaceValue is for internal use.
func (opt OptCollation) FindOneAndReplaceValue() interface{} {
	return opt.Collation
}

// FindOneAndDeleteName is for internal use.
func (opt OptCollation) FindOneAndDeleteName() string {
	return "collation"
}

// FindOneAndDeleteValue is for internal use.
func (opt OptCollation) FindOneAndDeleteValue() interface{} {
	return opt.Collation
}

// FindOneAndUpdateName is for internal use.
func (opt OptCollation) FindOneAndUpdateName() string {
	return "collation"
}

// FindOneAndUpdateValue is for internal use.
func (opt OptCollation) FindOneAndUpdateValue() interface{} {
	return opt.Collation
}

// UpdateName is for internal use.
func (opt OptCollation) UpdateName() string {
	return "collation"
}

// UpdateValue is for internal use.
func (opt OptCollation) UpdateValue() interface{} {
	return opt.Collation
}

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
func (OptCollation) updateOption()            {}

// OptComment is for internal use.
type OptComment string

// AggregateName is for internal use.
func (opt OptComment) AggregateName() string {
	return "comment"
}

// AggregateValue is for internal use.
func (opt OptComment) AggregateValue() interface{} {
	return opt
}

// FindName is for internal use.
func (opt OptComment) FindName() string {
	return "comment"
}

// FindValue is for internal use.
func (opt OptComment) FindValue() interface{} {
	return opt
}

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

// CountName is for internal use.
func (opt OptHint) CountName() string {
	return "hint"
}

// CountValue is for internal use.
func (opt OptHint) CountValue() interface{} {
	return opt.Hint
}

// FindName is for internal use.
func (opt OptHint) FindName() string {
	return "hint"
}

// FindValue is for internal use.
func (opt OptHint) FindValue() interface{} {
	return opt.Hint
}

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

// CountName is for internal use.
func (opt OptLimit) CountName() string {
	return "limit"
}

// CountValue is for internal use.
func (opt OptLimit) CountValue() interface{} {
	return opt
}

// FindName is for internal use.
func (opt OptLimit) FindName() string {
	return "limit"
}

// FindValue is for internal use.
func (opt OptLimit) FindValue() interface{} {
	return opt
}

// Option implements the Optioner interface.
func (opt OptLimit) Option(d *bson.Document) {
	d.Append(bson.C.Int64("limit", int64(opt)))
}

func (OptLimit) countOption() {}
func (OptLimit) findOption()  {}

// OptMax is for internal use.
//
// TODO(skriptble): Max should be *bson.Document.
type OptMax struct{ Max interface{} }

// FindName is for internal use.
func (opt OptMax) FindName() string {
	return "max"
}

// FindValue is for internal use.
func (opt OptMax) FindValue() interface{} {
	return opt.Max
}

// Option implements the Optioner interface.
func (opt OptMax) Option(d *bson.Document) {
	doc, ok := (opt).Max.(*bson.Document)
	if !ok {
		return
	}
	d.Append(bson.C.SubDocument("max", doc))
}

func (OptMax) findOption() {}

// OptMaxAwaitTime is for internal use.
type OptMaxAwaitTime time.Duration

// FindName is for internal use.
func (opt OptMaxAwaitTime) FindName() string {
	return "maxAwaitTimeMS"
}

// FindValue is for internal use.
func (opt OptMaxAwaitTime) FindValue() interface{} {
	return opt
}

// Option implements the Optioner interface.
func (opt OptMaxAwaitTime) Option(d *bson.Document) {
	d.Append(bson.C.Int64("maxAwaitTimeMS", int64(time.Duration(opt)/time.Millisecond)))
}

func (OptMaxAwaitTime) findOption() {}

// OptMaxScan is for internal use.
type OptMaxScan int64

// FindName is for internal use.
func (opt OptMaxScan) FindName() string {
	return "maxScan"
}

// FindValue is for internal use.
func (opt OptMaxScan) FindValue() interface{} {
	return opt
}

// Option implements the Optioner interface.
func (opt OptMaxScan) Option(d *bson.Document) {
	d.Append(bson.C.Int64("maxScan", int64(opt)))
}

func (OptMaxScan) findOption() {}

// OptMaxTime is for internal use.
type OptMaxTime time.Duration

// AggregateName is for internal use.
func (opt OptMaxTime) AggregateName() string {
	return "maxTimeMS"
}

// AggregateValue is for internal use.
func (opt OptMaxTime) AggregateValue() interface{} {
	return opt
}

// CountName is for internal use.
func (opt OptMaxTime) CountName() string {
	return "maxTimeMS"
}

// CountValue is for internal use.
func (opt OptMaxTime) CountValue() interface{} {
	return opt
}

// DistinctName is for internal use.
func (opt OptMaxTime) DistinctName() string {
	return "maxTimeMS"
}

// DistinctValue is for internal use.
func (opt OptMaxTime) DistinctValue() interface{} {
	return opt
}

// FindName is for internal use.
func (opt OptMaxTime) FindName() string {
	return "maxTimeMS"
}

// FindValue is for internal use.
func (opt OptMaxTime) FindValue() interface{} {
	return opt
}

// FindOneAndDeleteName is for internal use.
func (opt OptMaxTime) FindOneAndDeleteName() string {
	return "maxTimeMS"
}

// FindOneAndDeleteValue is for internal use.
func (opt OptMaxTime) FindOneAndDeleteValue() interface{} {
	return opt
}

// FindOneAndReplaceName is for internal use.
func (opt OptMaxTime) FindOneAndReplaceName() string {
	return "maxTimeMS"
}

// FindOneAndReplaceValue is for internal use.
func (opt OptMaxTime) FindOneAndReplaceValue() interface{} {
	return opt
}

// FindOneAndUpdateName is for internal use.
func (opt OptMaxTime) FindOneAndUpdateName() string {
	return "maxTimeMS"
}

// FindOneAndUpdateValue is for internal use.
func (opt OptMaxTime) FindOneAndUpdateValue() interface{} {
	return opt
}

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
//
// TODO(skriptble): Min should be *bson.Document.
type OptMin struct{ Min interface{} }

// FindName is for internal use.
func (opt OptMin) FindName() string {
	return "min"
}

// FindValue is for internal use.
func (opt OptMin) FindValue() interface{} {
	return opt.Min
}

// Option implements the Optioner interface.
func (opt OptMin) Option(d *bson.Document) {
	doc, ok := (opt).Min.(*bson.Document)
	if !ok {
		return
	}
	d.Append(bson.C.SubDocument("min", doc))
}

func (OptMin) findOption() {}

// OptNoCursorTimeout is for internal use.
type OptNoCursorTimeout bool

// FindName is for internal use.
func (opt OptNoCursorTimeout) FindName() string {
	return "noCursorTimeout"
}

// FindValue is for internal use.
func (opt OptNoCursorTimeout) FindValue() interface{} {
	return opt
}

// Option implements the Optioner interface.
func (opt OptNoCursorTimeout) Option(d *bson.Document) {
	d.Append(bson.C.Boolean("noCursorTimeout", bool(opt)))
}

func (OptNoCursorTimeout) findOption() {}

// OptOplogReplay is for internal use.
type OptOplogReplay bool

// FindName is for internal use.
func (opt OptOplogReplay) FindName() string {
	return "oplogReplay"
}

// FindValue is for internal use.
func (opt OptOplogReplay) FindValue() interface{} {
	return opt
}

// Option implements the Optioner interface.
func (opt OptOplogReplay) Option(d *bson.Document) {
	d.Append(bson.C.Boolean("oplogReplay", bool(opt)))
}

func (OptOplogReplay) findOption() {}

// OptOrdered is for internal use.
type OptOrdered bool

// InsertManyName is for internal use.
func (opt OptOrdered) InsertManyName() string {
	return "ordered"
}

// InsertManyValue is for internal use.
func (opt OptOrdered) InsertManyValue() interface{} {
	return opt
}

// InsertName is for internal use.
func (opt OptOrdered) InsertName() string {
	return "ordered"
}

// InsertValue is for internal use.
func (opt OptOrdered) InsertValue() interface{} {
	return opt
}

// Option implements the Optioner interface.
func (opt OptOrdered) Option(d *bson.Document) {
	d.Append(bson.C.Boolean("ordered", bool(opt)))
}

func (OptOrdered) insertManyOption() {}
func (OptOrdered) insertOption()     {}

// OptProjection is for internal use.
//
// TODO(skriptble): Projection should be *bson.Document.
type OptProjection struct {
	Projection interface{}
	find       bool
}

// FindName is for internal use.
func (opt OptProjection) FindName() string {
	return "projection"
}

// FindValue is for internal use.
func (opt OptProjection) FindValue() interface{} {
	return opt.Projection
}

// FindOneAndDeleteName is for internal use.
func (opt OptProjection) FindOneAndDeleteName() string {
	return "fields"
}

// FindOneAndDeleteValue is for internal use.
func (opt OptProjection) FindOneAndDeleteValue() interface{} {
	return opt.Projection
}

// FindOneAndReplaceName is for internal use.
func (opt OptProjection) FindOneAndReplaceName() string {
	return "fields"
}

// FindOneAndReplaceValue is for internal use.
func (opt OptProjection) FindOneAndReplaceValue() interface{} {
	return opt.Projection
}

// FindOneAndUpdateName is for internal use.
func (opt OptProjection) FindOneAndUpdateName() string {
	return "fields"
}

// FindOneAndUpdateValue is for internal use.
func (opt OptProjection) FindOneAndUpdateValue() interface{} {
	return opt.Projection
}

// Option implements the Optioner interface.
func (opt OptProjection) Option(d *bson.Document) {
	var key = "fields"
	if opt.find {
		key = "projection"
	}
	doc, ok := (opt).Projection.(*bson.Document)
	if !ok {
		return
	}
	d.Append(bson.C.SubDocument(key, doc))
}

func (OptProjection) findOption()              {}
func (OptProjection) findOneAndDeleteOption()  {}
func (OptProjection) findOneAndReplaceOption() {}
func (OptProjection) findOneAndUpdateOption()  {}

// OptReturnDocument is for internal use.
type OptReturnDocument ReturnDocument

// FindOneAndReplaceName is for internal use.
func (opt OptReturnDocument) FindOneAndReplaceName() string {
	return "returnDocument"
}

// FindOneAndReplaceValue is for internal use.
func (opt OptReturnDocument) FindOneAndReplaceValue() interface{} {
	return opt
}

// FindOneAndUpdateName is for internal use.
func (opt OptReturnDocument) FindOneAndUpdateName() string {
	return "returnDocument"
}

// FindOneAndUpdateValue is for internal use.
func (opt OptReturnDocument) FindOneAndUpdateValue() interface{} {
	return opt
}

// Option implements the Optioner interface.
func (opt OptReturnDocument) Option(d *bson.Document) {
	d.Append(bson.C.Boolean("new", ReturnDocument(opt) == After))
}

func (OptReturnDocument) findOneAndReplaceOption() {}
func (OptReturnDocument) findOneAndUpdateOption()  {}

// OptReturnKey is for internal use.
type OptReturnKey bool

// FindName is for internal use.
func (opt OptReturnKey) FindName() string {
	return "returnKey"
}

// FindValue is for internal use.
func (opt OptReturnKey) FindValue() interface{} {
	return opt
}

// Option implements the Optioner interface.
func (opt OptReturnKey) Option(d *bson.Document) {
	d.Append(bson.C.Boolean("returnKey", bool(opt)))
}

func (OptReturnKey) findOption() {}

// OptShowRecordID is for internal use.
type OptShowRecordID bool

// FindName is for internal use.
func (opt OptShowRecordID) FindName() string {
	return "showRecordId"
}

// FindValue is for internal use.
func (opt OptShowRecordID) FindValue() interface{} {
	return opt
}

// Option implements the Optioner interface.
func (opt OptShowRecordID) Option(d *bson.Document) {
	d.Append(bson.C.Boolean("showRecordId", bool(opt)))
}

func (OptShowRecordID) findOption() {}

// OptSkip is for internal use.
type OptSkip int64

// CountName is for internal use.
func (opt OptSkip) CountName() string {
	return "skip"
}

// CountValue is for internal use.
func (opt OptSkip) CountValue() interface{} {
	return opt
}

// FindName is for internal use.
func (opt OptSkip) FindName() string {
	return "skip"
}

// FindValue is for internal use.
func (opt OptSkip) FindValue() interface{} {
	return opt
}

// Option implements the Optioner interface.
func (opt OptSkip) Option(d *bson.Document) {
	d.Append(bson.C.Int64("skip", int64(opt)))
}

func (OptSkip) countOption() {}
func (OptSkip) findOption()  {}

// OptSnapshot is for internal use.
type OptSnapshot bool

// FindName is for internal use.
func (opt OptSnapshot) FindName() string {
	return "snapshot"
}

// FindValue is for internal use.
func (opt OptSnapshot) FindValue() interface{} {
	return opt
}

// Option implements the Optioner interface.
func (opt OptSnapshot) Option(d *bson.Document) {
	d.Append(bson.C.Boolean("snapshot", bool(opt)))
}

func (OptSnapshot) findOption() {}

// OptSort is for internal use.
//
// TODO(skriptble): Sort should be *bson.Document.
type OptSort struct{ Sort interface{} }

// FindName is for internal use.
func (opt OptSort) FindName() string {
	return "sort"
}

// FindValue is for internal use.
func (opt OptSort) FindValue() interface{} {
	return opt.Sort
}

// FindOneAndDeleteName is for internal use.
func (opt OptSort) FindOneAndDeleteName() string {
	return "sort"
}

// FindOneAndDeleteValue is for internal use.
func (opt OptSort) FindOneAndDeleteValue() interface{} {
	return opt.Sort
}

// FindOneAndReplaceName is for internal use.
func (opt OptSort) FindOneAndReplaceName() string {
	return "sort"
}

// FindOneAndReplaceValue is for internal use.
func (opt OptSort) FindOneAndReplaceValue() interface{} {
	return opt.Sort
}

// FindOneAndUpdateName is for internal use.
func (opt OptSort) FindOneAndUpdateName() string {
	return "sort"
}

// FindOneAndUpdateValue is for internal use.
func (opt OptSort) FindOneAndUpdateValue() interface{} {
	return opt.Sort
}

// Option implements the Optioner interface.
func (opt OptSort) Option(d *bson.Document) {
	doc, ok := (opt).Sort.(*bson.Document)
	if !ok {
		return
	}
	d.Append(bson.C.SubDocument("sort", doc))
}

func (OptSort) findOption()              {}
func (OptSort) findOneAndDeleteOption()  {}
func (OptSort) findOneAndReplaceOption() {}
func (OptSort) findOneAndUpdateOption()  {}

// OptUpsert is for internal use.
type OptUpsert bool

// FindOneAndReplaceName is for internal use.
func (opt OptUpsert) FindOneAndReplaceName() string {
	return "upsert"
}

// FindOneAndReplaceValue is for internal use.
func (opt OptUpsert) FindOneAndReplaceValue() interface{} {
	return opt
}

// FindOneAndUpdateName is for internal use.
func (opt OptUpsert) FindOneAndUpdateName() string {
	return "upsert"
}

// FindOneAndUpdateValue is for internal use.
func (opt OptUpsert) FindOneAndUpdateValue() interface{} {
	return opt
}

// UpdateName is for internal use.
func (opt OptUpsert) UpdateName() string {
	return "upsert"
}

// UpdateValue is for internal use.
func (opt OptUpsert) UpdateValue() interface{} {
	return opt
}

// Option implements the Optioner interface.
func (opt OptUpsert) Option(d *bson.Document) {
	d.Append(bson.C.Boolean("upsert", bool(opt)))
}

func (OptUpsert) findOneAndReplaceOption() {}
func (OptUpsert) findOneAndUpdateOption()  {}
func (OptUpsert) updateOption()            {}
