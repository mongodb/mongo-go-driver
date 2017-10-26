package options

// FindOneAndDeleteOption is for internal use.
type FindOneAndDeleteOption interface {
	FindOneAndDeleteName() string
	FindOneAndDeleteValue() interface{}
}

// collation

// FindOneAndDeleteName is for internal use.
func (opt *OptCollation) FindOneAndDeleteName() string {
	return "collation"
}

// FindOneAndDeleteValue is for internal use.
func (opt *OptCollation) FindOneAndDeleteValue() interface{} {
	return opt.Collation
}

// maxTimeMS

// FindOneAndDeleteName is for internal use.
func (opt *OptMaxTime) FindOneAndDeleteName() string {
	return "maxTimeMS"
}

// FindOneAndDeleteValue is for internal use.
func (opt *OptMaxTime) FindOneAndDeleteValue() interface{} {
	return *opt
}

// projection

// FindOneAndDeleteName is for internal use.
func (opt *OptProjection) FindOneAndDeleteName() string {
	return "fields"
}

// FindOneAndDeleteValue is for internal use.
func (opt *OptProjection) FindOneAndDeleteValue() interface{} {
	return opt.Projection
}

// sort

// FindOneAndDeleteName is for internal use.
func (opt *OptSort) FindOneAndDeleteName() string {
	return "sort"
}

// FindOneAndDeleteValue is for internal use.
func (opt *OptSort) FindOneAndDeleteValue() interface{} {
	return opt.Sort
}
