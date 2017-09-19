package options

// CountOption is for internal use.
type CountOption interface {
	CountName() string
	CountValue() interface{}
}

// collation

// CountName is for internal use.
func (opt *OptCollation) CountName() string {
	return "collation"
}

// CountValue is for internal use.
func (opt *OptCollation) CountValue() interface{} {
	return opt.Collation
}

// hint

// CountName is for internal use.
func (opt *OptHint) CountName() string {
	return "hint"
}

// CountValue is for internal use.
func (opt *OptHint) CountValue() interface{} {
	return opt.Hint
}

// limit

// CountName is for internal use.
func (opt *OptLimit) CountName() string {
	return "limit"
}

// CountValue is for internal use.
func (opt *OptLimit) CountValue() interface{} {
	return *opt
}

// maxTimeMS

// CountName is for internal use.
func (opt *OptMaxTime) CountName() string {
	return "maxTimeMS"
}

// CountValue is for internal use.
func (opt *OptMaxTime) CountValue() interface{} {
	return *opt
}

// skip

// CountName is for internal use.
func (opt *OptSkip) CountName() string {
	return "skip"
}

// CountValue is for internal use.
func (opt *OptSkip) CountValue() interface{} {
	return *opt
}
