package bson

// Iterator facilitates iterating over a bson.Document.
type Iterator struct {
	d     *Document
	index int
	elem  *Element
	err   error
}

func newIterator(d *Document) *Iterator {
	return &Iterator{d: d}
}

// Next fetches the next element of the document, returning whether or not the next element was able
// to be fetched. If true is returned, then call Element to get the element. If false is returned,
// call Err to check if an error occurred.
func (itr *Iterator) Next() bool {
	if itr.index >= len(itr.d.elems) {
		return false
	}

	e := itr.d.elems[itr.index]

	_, err := e.Validate()
	if err != nil {
		itr.err = err
		return false
	}

	itr.elem = e
	itr.index++

	return true
}

// Element returns the current element of the Iterator. The pointer that it returns will
// _always_ be the same for a given Iterator.
func (itr *Iterator) Element() *Element {
	return itr.elem
}

// Err returns the error that occurred when iterating, or nil if none occurred.
func (itr *Iterator) Err() error {
	return itr.err
}
