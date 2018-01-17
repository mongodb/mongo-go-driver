package options

// InsertOneOption is for internal use.
type InsertOneOption interface {
	InsertOneOptioner

	InsertOneName() string
	InsertOneValue() interface{}
}

type InsertOneOptioner interface {
	Optioner
	insertOneOption()
}
