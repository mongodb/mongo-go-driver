package options

// InsertOption is for internal use.
type InsertOption interface {
	InsertName() string
	InsertValue() interface{}
}

// bypassDocumentValidation

// InsertName is for internal use.
func (opt *OptBypassDocumentValidation) InsertName() string {
	return "bypassDocumentValidation"
}

// InsertValue is for internal use.
func (opt *OptBypassDocumentValidation) InsertValue() interface{} {
	return *opt
}

// ordered

// InsertName is for internal use.
func (opt *OptOrdered) InsertName() string {
	return "ordered"
}

// InsertValue is for internal use.
func (opt *OptOrdered) InsertValue() interface{} {
	return *opt
}
