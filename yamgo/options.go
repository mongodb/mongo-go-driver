package yamgo

import (
	"time"

	"github.com/10gen/mongo-go-driver/yamgo/options"
	"github.com/10gen/mongo-go-driver/yamgo/readpref"
)

type optReadPreference struct{ ReadPreference *readpref.ReadPref }

// DatabaseOption is an option to provide for Database creation.
type DatabaseOption interface {
	setDatabaseOption(*Database)
}

func (opt *optReadPreference) setDatabaseOption(db *Database) {
	db.readPreference = opt.ReadPreference
}

// CollectionOption is an option to provide for Collection creation.
type CollectionOption interface {
	setCollectionOption(*Collection)
}

func (opt *optReadPreference) setCollectionOption(db *Collection) {
	db.readPreference = opt.ReadPreference
}

// In order to facilitate users not having to supply a default value (e.g. nil or an uninitialized
// struct) when not using any options for an operation, options are defined as functions that take
// the necessary state and return a private type which implements an interface for the given
// operation. This API will allow users to use the following syntax for operations:
//
// coll.UpdateOne(filter, update)
// coll.UpdateOne(filter, update, yamgo.Upsert(true))
// coll.UpdateOne(filter, update, yamgo.Upsert(true), yamgo.bypassDocumentValidation(false))
//
// To see which options are available on each operation, see the following files:
//
// - delete_options.go
// - insert_options.go
// - update_options.go

// AllowDiskUse enables writing to temporary files.
func AllowDiskUse(b bool) *options.OptAllowDiskUse {
	opt := options.OptAllowDiskUse(b)
	return &opt
}

// AllowPartialResults gets partial results from a mongos if some shards are down (instead of
// throwing an error).
func AllowPartialResults(b bool) *options.OptAllowPartialResults {
	opt := options.OptAllowPartialResults(b)
	return &opt
}

// ArrayFilters specifies to which array elements an update should apply.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func ArrayFilters(filters ...interface{}) *options.OptArrayFilters {
	opt := options.OptArrayFilters(filters)
	return &opt
}

// BatchSize specifies the number of documents to return per batch.
func BatchSize(i int32) *options.OptBatchSize {
	opt := options.OptBatchSize(i)
	return &opt
}

// BypassDocumentValidation is used to opt out of document-level validation for a given write.
func BypassDocumentValidation(b bool) *options.OptBypassDocumentValidation {
	opt := options.OptBypassDocumentValidation(b)
	return &opt
}

// Collation allows users to specify language-specific rules for string comparison, such as rules
// for lettercase and accent marks.
//
// See https://docs.mongodb.com/manual/reference/collation/.
func Collation(collation *options.CollationOptions) *options.OptCollation {
	opt := options.OptCollation{Collation: collation}
	return &opt
}

// Comment specifies a comment to attach to the query to help attach and trace profile data.
//
// See https://docs.mongodb.com/manual/reference/command/profile.
func Comment(s string) *options.OptComment {
	opt := options.OptComment(s)
	return &opt
}

// CursorType indicates the type of cursor to use.
func CursorType(cursorType options.CursorType) *options.OptCursorType {
	opt := options.OptCursorType(cursorType)
	return &opt
}

// Hint specifies the index to use.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func Hint(hint interface{}) *options.OptHint {
	opt := options.OptHint{Hint: hint}
	return &opt
}

// Limit specifies the maximum number of documents to return.
func Limit(i int64) *options.OptLimit {
	opt := options.OptLimit(i)
	return &opt
}

// Max specifies the exclusive upper bound for a specific index.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func Max(max interface{}) *options.OptMax {
	opt := options.OptMax{Max: max}
	return &opt
}

// MaxAwaitTime specifies the maximum amount of time for the server to wait on new documents to
// satisfy a tailable-await cursor query.
func MaxAwaitTime(duration time.Duration) *options.OptMaxAwaitTime {
	opt := options.OptMaxAwaitTime(duration)
	return &opt
}

// MaxScan specifies the maximum number of documents or index keys to scan when executing the query.
func MaxScan(i int64) *options.OptMaxScan {
	opt := options.OptMaxScan(i)
	return &opt
}

// MaxTime specifies the maximum amount of time to allow the query to run.
func MaxTime(duration time.Duration) *options.OptMaxTime {
	opt := options.OptMaxTime(duration)
	return &opt
}

// Min specifies the inclusive lower bound for a specific index.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func Min(min interface{}) *options.OptMin {
	opt := options.OptMin{Min: min}
	return &opt
}

// NoCursorTimeout specifies whether to prevent the server from timing out idle cursors after an
// inactivity period.
func NoCursorTimeout(b bool) *options.OptNoCursorTimeout {
	opt := options.OptNoCursorTimeout(b)
	return &opt
}

func OplogReplay(b bool) *options.OptOplogReplay {
	opt := options.OptOplogReplay(b)
	return &opt
}

// Ordered specifies whether the remaining writes should be aborted if one of the earlier ones fails.
func Ordered(b bool) *options.OptOrdered {
	opt := options.OptOrdered(b)
	return &opt
}

// Projection limits the fields to return for all matching documents.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func Projection(projection interface{}) *options.OptProjection {
	opt := options.OptProjection{Projection: projection}
	return &opt
}

// ReadPreference describes which servers in a deployment can receive read operations.
func ReadPreference(readPreference *readpref.ReadPref) *optReadPreference {
	opt := optReadPreference{ReadPreference: readPreference}
	return &opt
}

// ReturnKey specifies whether to only return the index keys in the resulting documents.
func ReturnKey(b bool) *options.OptReturnKey {
	opt := options.OptReturnKey(b)
	return &opt
}

// ShowRecordID determines whether to return the record identifier for each document.
func ShowRecordID(b bool) *options.OptShowRecordID {
	opt := options.OptShowRecordID(b)
	return &opt
}

// Skip specifies the number of documents to skip before returning.
func Skip(i int64) *options.OptSkip {
	opt := options.OptSkip(i)
	return &opt
}

// Snapshot specifies whether to prevent the cursor from returning a document more than once
// because of an intervening write operation.
func Snapshot(b bool) *options.OptSnapshot {
	opt := options.OptSnapshot(b)
	return &opt
}

// Sort specifies order in which to return matching documents.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func Sort(sort interface{}) *options.OptSort {
	opt := options.OptSort{Sort: sort}
	return &opt
}

// Upsert specifies that a new document should be inserted if no document matches the update
// filter.
func Upsert(b bool) *options.OptUpsert {
	opt := options.OptUpsert(b)
	return &opt
}
