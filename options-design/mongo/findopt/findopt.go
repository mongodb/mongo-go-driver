package findopt

import (
	"time"

	"github.com/mongodb/mongo-go-driver/options-design/mongo/mongoopt"
	"github.com/mongodb/mongo-go-driver/options-design/mongo/readconcern"
	"github.com/mongodb/mongo-go-driver/options-design/mongo/writeconcern"
	"github.com/mongodb/mongo-go-driver/options-design/option"
)

type Find interface{ find() }
type DeleteOne interface{ oneAndDelete() }
type ReplaceOne interface{ oneAndReplace() }
type UpdateOne interface{ oneAndUpdate() }
type One interface{ one() }

type FindBundle struct{}

func BundleFind(...Find) *FindBundle { return nil }

func (fb *FindBundle) AllowPartialResults(b bool) *FindBundle              { return nil }
func (fb *FindBundle) BatchSize(i int32) *FindBundle                       { return nil }
func (fb *FindBundle) Collation(collation *mongoopt.Collation) *FindBundle { return nil }
func (fb *FindBundle) Comment(s string) *FindBundle                        { return nil }
func (fb *FindBundle) CursorType(ct mongoopt.CursorType) *FindBundle       { return nil }
func (fb *FindBundle) Hint(hing interface{}) *FindBundle                   { return nil }
func (fb *FindBundle) Limit(i int64) *FindBundle                           { return nil }
func (fb *FindBundle) Max(max interface{}) *FindBundle                     { return nil }
func (fb *FindBundle) MaxAwaitTime(d time.Duration) *FindBundle            { return nil }
func (fb *FindBundle) MaxScan(i int64) *FindBundle                         { return nil }
func (fb *FindBundle) MaxTime(d time.Duration) *FindBundle                 { return nil }
func (fb *FindBundle) Min(min interface{}) *FindBundle                     { return nil }
func (fb *FindBundle) NoCursorTimeout(b bool) *FindBundle                  { return nil }
func (fb *FindBundle) OplogReplay(b bool) *FindBundle                      { return nil }
func (fb *FindBundle) Projection(projection interface{}) *FindBundle       { return nil }
func (fb *FindBundle) ReadConcern(rc *readconcern.ReadConcern) *FindBundle { return nil }
func (fb *FindBundle) ReturnKey(b bool) *FindBundle                        { return nil }
func (fb *FindBundle) ShowRecordID(b bool) *FindBundle                     { return nil }
func (fb *FindBundle) Skip(i int64) *FindBundle                            { return nil }
func (fb *FindBundle) Snapshot(b bool) *FindBundle                         { return nil }
func (fb *FindBundle) Sort(sort interface{}) *FindBundle                   { return nil }

// Unbundle unwinds and deduplicates the options used to create it and those
// added after creation into a single slice of options. This method returns
// a slice of options.Optioner because it may contain both FindOptioners and
// CusrorOptioners.
//
// The deduplicate parameter is used to determine if the bundle is just flattened or
// if we actually deduplicate options.
//
// Since a FindBundle can be recursive, this method will unwind all recursive FindBundles.
func (fb *FindBundle) Unbundle(deduplicate bool) []option.Optioner { return nil }

func (fb *FindBundle) find() {}

type OneBundle struct{}

func BundleOne(...One) *OneBundle { return nil }

func (ob *OneBundle) AllowPartialResults(b bool) *OneBundle              { return nil }
func (ob *OneBundle) BatchSize(i int32) *OneBundle                       { return nil }
func (ob *OneBundle) Collation(collation *mongoopt.Collation) *OneBundle { return nil }
func (ob *OneBundle) Comment(s string) *OneBundle                        { return nil }
func (ob *OneBundle) CursorType(ct mongoopt.CursorType) *OneBundle       { return nil }
func (ob *OneBundle) Hint(hing interface{}) *OneBundle                   { return nil }
func (ob *OneBundle) Max(max interface{}) *OneBundle                     { return nil }
func (ob *OneBundle) MaxAwaitTime(d time.Duration) *OneBundle            { return nil }
func (ob *OneBundle) MaxScan(i int64) *OneBundle                         { return nil }
func (ob *OneBundle) MaxTime(d time.Duration) *OneBundle                 { return nil }
func (ob *OneBundle) Min(min interface{}) *OneBundle                     { return nil }
func (ob *OneBundle) NoCursorTimeout(b bool) *OneBundle                  { return nil }
func (ob *OneBundle) OplogReplay(b bool) *OneBundle                      { return nil }
func (ob *OneBundle) Projection(projection interface{}) *OneBundle       { return nil }
func (ob *OneBundle) ReadConcern(rc *readconcern.ReadConcern) *OneBundle { return nil }
func (ob *OneBundle) ReturnKey(b bool) *OneBundle                        { return nil }
func (ob *OneBundle) ShowRecordID(b bool) *OneBundle                     { return nil }
func (ob *OneBundle) Skip(i int64) *OneBundle                            { return nil }
func (ob *OneBundle) Snapshot(b bool) *OneBundle                         { return nil }
func (ob *OneBundle) Sort(sort interface{}) *OneBundle                   { return nil }

func (ob *OneBundle) Unbundle() []option.Optioner { return nil }

type DeleteOneBundle struct{}

func BundleDeleteOne(...DeleteOne) *DeleteOneBundle { return nil }

func (dob *DeleteOneBundle) Collation(collation *mongoopt.Collation) *DeleteOneBundle    { return nil }
func (dob *DeleteOneBundle) MaxTime(d time.Duration) *DeleteOneBundle                    { return nil }
func (dob *DeleteOneBundle) Projection(projection interface{}) *DeleteOneBundle          { return nil }
func (dob *DeleteOneBundle) Sort(sort interface{}) *DeleteOneBundle                      { return nil }
func (dob *DeleteOneBundle) WriteConcern(wc *writeconcern.WriteConcern) *DeleteOneBundle { return nil }

func (dob *DeleteOneBundle) Unbundle() []option.FindOneAndDeleteOptioner { return nil }

type ReplaceOneBundle struct{}

func BundleReplaceOne(...ReplaceOne) *ReplaceOneBundle { return nil }

func (rob *ReplaceOneBundle) BypassDocumentValidation(b bool) *ReplaceOneBundle         { return nil }
func (rob *ReplaceOneBundle) Collation(collation *mongoopt.Collation) *ReplaceOneBundle { return nil }
func (rob *ReplaceOneBundle) MaxTime(d time.Duration) *ReplaceOneBundle                 { return nil }
func (rob *ReplaceOneBundle) Projection(projection interface{}) *ReplaceOneBundle       { return nil }
func (rob *ReplaceOneBundle) ReturnDocument(rd mongoopt.ReturnDocument) *ReplaceOneBundle {
	return nil
}
func (rob *ReplaceOneBundle) Sort(sort interface{}) *ReplaceOneBundle                      { return nil }
func (rob *ReplaceOneBundle) Upsert(b bool) *ReplaceOneBundle                              { return nil }
func (rob *ReplaceOneBundle) WriteConcern(wc *writeconcern.WriteConcern) *ReplaceOneBundle { return nil }

func (rob *ReplaceOneBundle) Unbundle() []option.FindOneAndReplaceOptioner { return nil }

type UpdateOneBundle struct{}

func BundleUpdateOne(...UpdateOne) *UpdateOneBundle { return nil }

func (uob *UpdateOneBundle) ArrayFilters(filters ...interface{}) *ReplaceOneBundle        { return nil }
func (uob *UpdateOneBundle) BypassDocumentValidation(b bool) *ReplaceOneBundle            { return nil }
func (uob *UpdateOneBundle) Collation(collation *mongoopt.Collation) *ReplaceOneBundle    { return nil }
func (uob *UpdateOneBundle) MaxTime(d time.Duration) *ReplaceOneBundle                    { return nil }
func (uob *UpdateOneBundle) Projection(projection interface{}) *ReplaceOneBundle          { return nil }
func (uob *UpdateOneBundle) ReturnDocument(rd mongoopt.ReturnDocument) *ReplaceOneBundle  { return nil }
func (uob *UpdateOneBundle) Sort(sort interface{}) *ReplaceOneBundle                      { return nil }
func (uob *UpdateOneBundle) Upsert(b bool) *ReplaceOneBundle                              { return nil }
func (uob *UpdateOneBundle) WriteConcern(wc *writeconcern.WriteConcern) *ReplaceOneBundle { return nil }

func (uob *UpdateOneBundle) Unbundle() []option.FindOneAndUpdateOptioner { return nil }

func AllowPartialResults(b bool) OptAllowPartialResults           { return false }             // Find, One
func ArrayFilters(filters ...interface{}) OptArrayFilters         { return OptArrayFilters{} } // UpdateOne
func BatchSize(i int32) OptBatchSize                              { return 0 }                 // Find, One
func BypassDocumentValidation(b bool) OptBypassDocumentValidation { return false }             // ReplaceOne, UpdateOne
func Collation(collation *mongoopt.Collation) OptCollation        { return OptCollation{} }    // Find, One, DeleteOne, ReplaceOne, UpdateOne
func CursorType(ct mongoopt.CursorType) OptCursorType             { return 0 }                 // Find, One
func Comment(s string) OptComment                                 { return "" }                // Find, One
func Hint(hing interface{}) OptHint                               { return OptHint{} }         // Find, One
func Limit(i int64) OptLimit                                      { return 0 }                 // Find
func Max(max interface{}) OptMax                                  { return OptMax{} }          // Find, One
func MaxAwaitTime(d time.Duration) OptMaxAwaitTime                { return 0 }                 // Find, One
func MaxScan(i int64) OptMaxScan                                  { return 0 }                 // Find, One
func MaxTime(d time.Duration) OptMaxTime                          { return 0 }                 // Find, One, DeleteOne, ReplaceOne, UpdateOne
func Min(min interface{}) OptMin                                  { return OptMin{} }          // Find, One
func NoCursorTimeout(b bool) OptNoCursorTimeout                   { return false }             // Find, One
func OplogReplay(b bool) OptOplogReplay                           { return false }             // Find, One
func Projection(projection interface{}) OptProjection             { return OptProjection{} }   // Find, One, DeleteOne, ReplaceOne, UpdateOne
func ReadConcern(rc *readconcern.ReadConcern) OptReadConcern      { return OptReadConcern{} }  // Find, One
func ReturnDocument(rd mongoopt.ReturnDocument) OptReturnDocument { return 0 }                 // ReplaceOne, UpdateOne
func ReturnKey(b bool) OptReturnKey                               { return false }             // Find, One
func ShowRecordID(b bool) OptShowRecordID                         { return false }             // Find, One
func Skip(i int64) OptSkip                                        { return 0 }                 // Find, One
func Snapshot(b bool) OptSnapshot                                 { return false }             // Find, One
func Sort(sort interface{}) OptSort                               { return OptSort{} }         // Find, One, DeleteOne, ReplaceOne, UpdateOne
func Upsert(b bool) OptUpsert                                     { return false }             // ReplaceOne, UpdateOne
func WriteConcern(wc *writeconcern.WriteConcern) OptWriteConcern  { return OptWriteConcern{} } // DeleteOne, ReplaceOne, UpdateOne

type OptAllowPartialResults option.OptAllowPartialResults

func (OptAllowPartialResults) find() {}
func (OptAllowPartialResults) one()  {}

type OptArrayFilters option.OptArrayFilters

func (OptArrayFilters) updateOne() {}

type OptBatchSize option.OptBatchSize

func (OptBatchSize) find() {}
func (OptBatchSize) one()  {}

type OptBypassDocumentValidation option.OptBypassDocumentValidation

func (OptBypassDocumentValidation) replaceOne() {}
func (OptBypassDocumentValidation) updateOne()  {}

type OptCollation option.OptCollation

func (OptCollation) find()       {}
func (OptCollation) one()        {}
func (OptCollation) deleteOne()  {}
func (OptCollation) replaceOne() {}
func (OptCollation) updateOne()  {}

type OptCursorType option.OptCursorType

func (OptCursorType) find() {}
func (OptCursorType) one()  {}

type OptComment option.OptComment

func (OptComment) find() {}
func (OptComment) one()  {}

type OptHint option.OptHint

func (OptHint) find() {}
func (OptHint) one()  {}

type OptLimit option.OptLimit

func (OptLimit) find() {}

type OptMax option.OptMax

func (OptMax) find() {}
func (OptMax) one()  {}

type OptMaxAwaitTime option.OptMaxAwaitTime

func (OptMaxAwaitTime) find() {}
func (OptMaxAwaitTime) one()  {}

type OptMaxScan option.OptMaxScan

func (OptMaxScan) find() {}
func (OptMaxScan) one()  {}

type OptMaxTime option.OptMaxTime

func (OptMaxTime) find()       {}
func (OptMaxTime) one()        {}
func (OptMaxTime) deleteOne()  {}
func (OptMaxTime) replaceOne() {}
func (OptMaxTime) updateOne()  {}

type OptMin option.OptMin

func (OptMin) find() {}
func (OptMin) one()  {}

type OptNoCursorTimeout option.OptNoCursorTimeout

func (OptNoCursorTimeout) find() {}
func (OptNoCursorTimeout) one()  {}

type OptOplogReplay option.OptOplogReplay

func (OptOplogReplay) find() {}
func (OptOplogReplay) one()  {}

type OptProjection option.OptProjection

func (OptProjection) find()       {}
func (OptProjection) one()        {}
func (OptProjection) deleteOne()  {}
func (OptProjection) replaceOne() {}
func (OptProjection) updateOne()  {}

type OptReadConcern option.OptReadConcern

func (OptReadConcern) find() {}
func (OptReadConcern) one()  {}

type OptReturnDocument option.OptReturnDocument

func (OptReturnDocument) replaceOne() {}
func (OptReturnDocument) updateOne()  {}

type OptReturnKey option.OptReturnKey

func (OptReturnKey) find() {}
func (OptReturnKey) one()  {}

type OptShowRecordID option.OptShowRecordID

func (OptShowRecordID) find() {}
func (OptShowRecordID) one()  {}

type OptSkip option.OptSkip

func (OptSkip) find() {}
func (OptSkip) one()  {}

type OptSnapshot option.OptSnapshot

func (OptSnapshot) find() {}
func (OptSnapshot) one()  {}

type OptSort option.OptSort

func (OptSort) find()       {}
func (OptSort) one()        {}
func (OptSort) deleteOne()  {}
func (OptSort) replaceOne() {}
func (OptSort) updateOne()  {}

type OptUpsert option.OptUpsert

func (OptUpsert) reaplceOne() {}
func (OptUpsert) updateOne()  {}

type OptWriteConcern option.OptWriteConcern

func (OptWriteConcern) deleteOne()  {}
func (OptWriteConcern) replaceOne() {}
func (OptWriteConcern) updateOne()  {}
