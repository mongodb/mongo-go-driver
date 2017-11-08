package options

// InsertOneOption is for internal use.
type InsertOneOption interface {
	InsertOneName() string
	InsertOneValue() interface{}
}

// bypassDocumentValidation

// InsertOneName is for internal use.
func (opt *OptBypassDocumentValidation) InsertOneName() string {
	return "bypassDocumentValidation"
}

// InsertOneValue is for internal use.
func (opt *OptBypassDocumentValidation) InsertOneValue() interface{} {
	return *opt
}
