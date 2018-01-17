package options

// FindOneAndDeleteOption is for internal use.
type FindOneAndDeleteOption interface {
	FindOneAndDeleteOptioner

	FindOneAndDeleteName() string
	FindOneAndDeleteValue() interface{}
}

type FindOneAndDeleteOptioner interface {
	Optioner
	findOneAndDeleteOption()
}
