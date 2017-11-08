package options

// InsertManyOption is for internal use.
type InsertManyOption interface {
	InsertManyName() string
	InsertManyValue() interface{}
}

// bypassDocumentValidation

// InsertManyName is for internal use.
func (opt *OptBypassDocumentValidation) InsertManyName() string {
	return "bypassDocumentValidation"
}

// InsertManyValue is for internal use.
func (opt *OptBypassDocumentValidation) InsertManyValue() interface{} {
	return *opt
}

// ordered

// InsertManyName is for internal use.
func (opt *OptOrdered) InsertManyName() string {
	return "ordered"
}

// InsertManyValue is for internal use.
func (opt *OptOrdered) InsertManyValue() interface{} {
	return *opt
}
