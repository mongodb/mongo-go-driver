package options

// FindOneAndReplaceOption is for internal use.
type FindOneAndReplaceOption interface {
	FindOneAndReplaceOptioner

	FindOneAndReplaceName() string
	FindOneAndReplaceValue() interface{}
}

// FindOneAndReplaceOptioner is the interface implemented by types that can be
// used as Options for FindOneAndReplace commands.
type FindOneAndReplaceOptioner interface {
	Optioner
	findOneAndReplaceOption()
}
