package options

// FindOneAndReplaceOption is for internal use.
type FindOneAndReplaceOption interface {
	FindOneAndReplaceOptioner

	FindOneAndReplaceName() string
	FindOneAndReplaceValue() interface{}
}

type FindOneAndReplaceOptioner interface {
	Optioner
	findOneAndReplaceOption()
}
