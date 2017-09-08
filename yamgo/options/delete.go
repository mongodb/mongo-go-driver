package options

// DeleteOption is for internal use.
type DeleteOption interface {
	DeleteName() string
	DeleteValue() interface{}
}

// collation

// DeleteName is for internal use.
func (opt *OptCollation) DeleteName() string {
	return "collation"
}

// DeleteValue is for internal use.
func (opt *OptCollation) DeleteValue() interface{} {
	return opt.Collation
}
