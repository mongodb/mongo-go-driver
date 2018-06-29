package bson

// ArrayIterator facilitates iterating over a bson.Array.
type ArrayIterator struct {
	array *Array
	pos   uint
	val   *Value
	err   error
}

// NewArrayIterator constructs a new ArrayIterator over a given Array
func NewArrayIterator(array *Array) (*ArrayIterator, error) {
	iter := &ArrayIterator{}
	iter.array = array

	return iter, nil
}

// Next fetches the next value in the Array, returning whether or not it could be fetched successfully. If true is
// returned, call Value to get the value. If false is returned, call Err to check if an error occurred.
func (iter *ArrayIterator) Next() bool {
	v, err := iter.array.Lookup(iter.pos)

	if err != nil {
		// error if out of bounds
		// don't assign iter.err
		return false
	}

	_, err = v.validate(false)
	if err != nil {
		iter.err = err
		return false
	}

	iter.val = v
	iter.pos++

	return true
}

// Value returns the current value of the ArrayIterator. The pointer returned will _always_ be the same for a given
// ArrayIterator. The returned value will be nil if this function is called before the first successful call to Next().
func (iter *ArrayIterator) Value() *Value {
	return iter.val
}

// Err returns the error that occurred while iterating, or nil if none occurred.
func (iter *ArrayIterator) Err() error {
	return iter.err
}
