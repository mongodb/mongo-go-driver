package options

// InsertOneOption is for internal use.
type InsertOneOption interface {
	InsertOneOptioner

	InsertOneName() string
	InsertOneValue() interface{}
}

// InsertOneOptioner is the interface implemented by types that can be used as
// Options for InsertOne commands.
type InsertOneOptioner interface {
	Optioner
	insertOneOption()
}
