package options

// FindOneAndUpdateOption is for internal use.
type FindOneAndUpdateOption interface {
	FindOneAndUpdateOptioner

	FindOneAndUpdateName() string
	FindOneAndUpdateValue() interface{}
}

type FindOneAndUpdateOptioner interface {
	Optioner
	findOneAndUpdateOption()
}
