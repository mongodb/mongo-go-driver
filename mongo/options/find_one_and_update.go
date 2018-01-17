package options

// FindOneAndUpdateOption is for internal use.
type FindOneAndUpdateOption interface {
	FindOneAndUpdateOptioner

	FindOneAndUpdateName() string
	FindOneAndUpdateValue() interface{}
}

// FindOneAndUpdateOptioner is the interface implemented by types that can be
// used as Options for FindOneAndUpdate commands.
type FindOneAndUpdateOptioner interface {
	Optioner
	findOneAndUpdateOption()
}
