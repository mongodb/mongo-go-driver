package options

// InsertManyOption is for internal use.
type InsertManyOption interface {
	InsertManyOptioner

	InsertManyName() string
	InsertManyValue() interface{}
}

// InsertManyOptioner is the interface implemented by types that can be used as
// Options for InsertMany commands.
type InsertManyOptioner interface {
	Optioner
	insertManyOption()
}
