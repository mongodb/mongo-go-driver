package options

// FindOneAndReplaceOption is for internal use.
type FindOneAndReplaceOption interface {
	FindOneAndReplaceName() string
	FindOneAndReplaceValue() interface{}
}

// bypassDocumentValidation

// FindOneAndReplaceName is for internal use.
func (opt *OptBypassDocumentValidation) FindOneAndReplaceName() string {
	return "bypassDocumentValidation"
}

// FindOneAndReplaceValue is for internal use.
func (opt *OptBypassDocumentValidation) FindOneAndReplaceValue() interface{} {
	return *opt
}

// collation

// FindOneAndReplaceName is for internal use.
func (opt *OptCollation) FindOneAndReplaceName() string {
	return "collation"
}

// FindOneAndReplaceValue is for internal use.
func (opt *OptCollation) FindOneAndReplaceValue() interface{} {
	return opt.Collation
}

// maxTimeMS

// FindOneAndReplaceName is for internal use.
func (opt *OptMaxTime) FindOneAndReplaceName() string {
	return "maxTimeMS"
}

// FindOneAndReplaceValue is for internal use.
func (opt *OptMaxTime) FindOneAndReplaceValue() interface{} {
	return *opt
}

// projection

// FindOneAndReplaceName is for internal use.
func (opt *OptProjection) FindOneAndReplaceName() string {
	return "fields"
}

// FindOneAndReplaceValue is for internal use.
func (opt *OptProjection) FindOneAndReplaceValue() interface{} {
	return opt.Projection
}

// returnDocument

// FindOneAndReplaceName is for internal use.
func (opt *OptReturnDocument) FindOneAndReplaceName() string {
	return "returnDocument"
}

// FindOneAndReplaceValue is for internal use.
func (opt *OptReturnDocument) FindOneAndReplaceValue() interface{} {
	return *opt
}

// sort

// FindOneAndReplaceName is for internal use.
func (opt *OptSort) FindOneAndReplaceName() string {
	return "sort"
}

// FindOneAndReplaceValue is for internal use.
func (opt *OptSort) FindOneAndReplaceValue() interface{} {
	return opt.Sort
}

// upsert

// FindOneAndReplaceName is for internal use.
func (opt *OptUpsert) FindOneAndReplaceName() string {
	return "upsert"
}

// FindOneAndReplaceValue is for internal use.
func (opt *OptUpsert) FindOneAndReplaceValue() interface{} {
	return *opt
}
