package options

// FindOption is for internal use.
type FindOption interface {
	FindName() string
	FindValue() interface{}
}

// allowPartialResults

// FindName is for internal use.
func (opt *OptAllowPartialResults) FindName() string {
	return "allowPartialResults"
}

// FindValue is for internal use.
func (opt *OptAllowPartialResults) FindValue() interface{} {
	return *opt
}

// batchSize

// FindName is for internal use.
func (opt *OptBatchSize) FindName() string {
	return "batchSize"
}

// FindValue is for internal use.
func (opt *OptBatchSize) FindValue() interface{} {
	return *opt
}

// collation

// FindName is for internal use.
func (opt *OptCollation) FindName() string {
	return "collation"
}

// FindValue is for internal use.
func (opt *OptCollation) FindValue() interface{} {
	return opt.Collation
}

// comment

// FindName is for internal use.
func (opt *OptComment) FindName() string {
	return "comment"
}

// FindValue is for internal use.
func (opt *OptComment) FindValue() interface{} {
	return *opt
}

// hint

// FindName is for internal use.
func (opt *OptHint) FindName() string {
	return "hint"
}

// FindValue is for internal use.
func (opt *OptHint) FindValue() interface{} {
	return opt.Hint
}

// limit

// FindName is for internal use.
func (opt *OptLimit) FindName() string {
	return "limit"
}

// FindValue is for internal use.
func (opt *OptLimit) FindValue() interface{} {
	return *opt
}

// max

// FindName is for internal use.
func (opt *OptMax) FindName() string {
	return "max"
}

// FindValue is for internal use.
func (opt *OptMax) FindValue() interface{} {
	return opt.Max
}

// maxAwaitTimeMS

// FindName is for internal use.
func (opt *OptMaxAwaitTime) FindName() string {
	return "maxAwaitTimeMS"
}

// FindValue is for internal use.
func (opt *OptMaxAwaitTime) FindValue() interface{} {
	return *opt
}

// maxScan

// FindName is for internal use.
func (opt *OptMaxScan) FindName() string {
	return "maxScan"
}

// FindValue is for internal use.
func (opt *OptMaxScan) FindValue() interface{} {
	return *opt
}

// maxTimeMS

// FindName is for internal use.
func (opt *OptMaxTime) FindName() string {
	return "maxTimeMS"
}

// FindValue is for internal use.
func (opt *OptMaxTime) FindValue() interface{} {
	return *opt
}

// min

// FindName is for internal use.
func (opt *OptMin) FindName() string {
	return "min"
}

// FindValue is for internal use.
func (opt *OptMin) FindValue() interface{} {
	return opt.Min
}

// noCursorTimeout

// FindName is for internal use.
func (opt *OptNoCursorTimeout) FindName() string {
	return "noCursorTimeout"
}

// FindValue is for internal use.
func (opt *OptNoCursorTimeout) FindValue() interface{} {
	return *opt
}

// oplogReplay

// FindName is for internal use.
func (opt *OptOplogReplay) FindName() string {
	return "oplogReplay"
}

// FindValue is for internal use.
func (opt *OptOplogReplay) FindValue() interface{} {
	return *opt
}

// projection

// FindName is for internal use.
func (opt *OptProjection) FindName() string {
	return "projection"
}

// FindValue is for internal use.
func (opt *OptProjection) FindValue() interface{} {
	return opt.Projection
}

// returnKey

// FindName is for internal use.
func (opt *OptReturnKey) FindName() string {
	return "returnKey"
}

// FindValue is for internal use.
func (opt *OptReturnKey) FindValue() interface{} {
	return *opt
}

// showRecordId

// FindName is for internal use.
func (opt *OptShowRecordID) FindName() string {
	return "showRecordId"
}

// FindValue is for internal use.
func (opt *OptShowRecordID) FindValue() interface{} {
	return *opt
}

// skip

// FindName is for internal use.
func (opt *OptSkip) FindName() string {
	return "skip"
}

// FindValue is for internal use.
func (opt *OptSkip) FindValue() interface{} {
	return *opt
}

// snapshot

// FindName is for internal use.
func (opt *OptSnapshot) FindName() string {
	return "snapshot"
}

// FindValue is for internal use.
func (opt *OptSnapshot) FindValue() interface{} {
	return *opt
}

// sort

// FindName is for internal use.
func (opt *OptSort) FindName() string {
	return "sort"
}

// FindValue is for internal use.
func (opt *OptSort) FindValue() interface{} {
	return opt.Sort
}
