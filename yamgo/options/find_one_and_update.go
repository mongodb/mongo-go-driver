package options

// FindOneAndUpdateOption is for internal use.
type FindOneAndUpdateOption interface {
	FindOneAndUpdateName() string
	FindOneAndUpdateValue() interface{}
}

// arrayFilters

// FindOneAndUpdateName is for internal use.
func (opt *OptArrayFilters) FindOneAndUpdateName() string {
	return "arrayFilters"
}

// FindOneAndUpdateValue is for internal use.
func (opt *OptArrayFilters) FindOneAndUpdateValue() interface{} {
	return *opt
}

// bypassDocumentValidation

// FindOneAndUpdateName is for internal use.
func (opt *OptBypassDocumentValidation) FindOneAndUpdateName() string {
	return "bypassDocumentValidation"
}

// FindOneAndUpdateValue is for internal use.
func (opt *OptBypassDocumentValidation) FindOneAndUpdateValue() interface{} {
	return *opt
}

// collation

// FindOneAndUpdateName is for internal use.
func (opt *OptCollation) FindOneAndUpdateName() string {
	return "collation"
}

// FindOneAndUpdateValue is for internal use.
func (opt *OptCollation) FindOneAndUpdateValue() interface{} {
	return opt.Collation
}

// maxTimeMS

// FindOneAndUpdateName is for internal use.
func (opt *OptMaxTime) FindOneAndUpdateName() string {
	return "maxTimeMS"
}

// FindOneAndUpdateValue is for internal use.
func (opt *OptMaxTime) FindOneAndUpdateValue() interface{} {
	return *opt
}

// projection

// FindOneAndUpdateName is for internal use.
func (opt *OptProjection) FindOneAndUpdateName() string {
	return "fields"
}

// FindOneAndUpdateValue is for internal use.
func (opt *OptProjection) FindOneAndUpdateValue() interface{} {
	return opt.Projection
}

// returnDocument

// FindOneAndUpdateName is for internal use.
func (opt *OptReturnDocument) FindOneAndUpdateName() string {
	return "returnDocument"
}

// FindOneAndUpdateValue is for internal use.
func (opt *OptReturnDocument) FindOneAndUpdateValue() interface{} {
	return *opt
}

// sort

// FindOneAndUpdateName is for internal use.
func (opt *OptSort) FindOneAndUpdateName() string {
	return "sort"
}

// FindOneAndUpdateValue is for internal use.
func (opt *OptSort) FindOneAndUpdateValue() interface{} {
	return opt.Sort
}

// upsert

// FindOneAndUpdateName is for internal use.
func (opt *OptUpsert) FindOneAndUpdateName() string {
	return "upsert"
}

// FindOneAndUpdateValue is for internal use.
func (opt *OptUpsert) FindOneAndUpdateValue() interface{} {
	return *opt
}
