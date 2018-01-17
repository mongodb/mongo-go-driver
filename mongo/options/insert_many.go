package options

// InsertManyOption is for internal use.
type InsertManyOption interface {
	InsertManyOptioner

	InsertManyName() string
	InsertManyValue() interface{}
}

type InsertManyOptioner interface {
	Optioner
	insertManyOption()
}
