package options

// DistinctOption is for internal use.
type DistinctOption interface {
	DistinctName() string
	DistinctValue() interface{}
}

// collation

// DistinctName is for internal use.
func (opt *OptCollation) DistinctName() string {
	return "collation"
}

// DistinctValue is for internal use.
func (opt *OptCollation) DistinctValue() interface{} {
	return opt.Collation
}

// maxTimeMS

// DistinctName is for internal use.
func (opt *OptMaxTime) DistinctName() string {
	return "maxTimeMS"
}

// DistinctValue is for internal use.
func (opt *OptMaxTime) DistinctValue() interface{} {
	return *opt
}
