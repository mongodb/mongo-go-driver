package options

// FindOneAndDeleteOption is for internal use.
type FindOneAndDeleteOption interface {
	FindOneAndDeleteOptioner

	FindOneAndDeleteName() string
	FindOneAndDeleteValue() interface{}
}

// FindOneAndDeleteOptioner is the interface implemented by types that can be
// used as Options for FindOneAndDelete commands.
type FindOneAndDeleteOptioner interface {
	Optioner
	findOneAndDeleteOption()
}
