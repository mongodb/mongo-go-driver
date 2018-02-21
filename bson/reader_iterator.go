// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

// ReaderIterator facilitates iterating over a bson.Reader.
type ReaderIterator struct {
	r    Reader
	pos  uint32
	end  uint32
	elem *Element
	err  error
}

// NewReaderIterator constructors a new ReaderIterator over a given Reader.
func NewReaderIterator(r Reader) (*ReaderIterator, error) {
	itr := new(ReaderIterator)
	if len(r) < 5 {
		return nil, NewErrTooSmall()
	}
	givenLength := readi32(r[0:4])
	if len(r) < int(givenLength) {
		return nil, ErrInvalidLength
	}

	itr.r = r
	itr.pos = 4
	itr.end = uint32(givenLength)
	itr.elem = &Element{value: &Value{}}

	return itr, nil
}

// Next fetches the next element of the Reader, returning whether or not the next element was able
// to be fetched. If true is returned, then call Element to get the element. If false is returned,
// call Err to check if an error occurred.
func (itr *ReaderIterator) Next() bool {
	if itr.pos >= itr.end {
		itr.err = ErrInvalidReadOnlyDocument
		return false
	}
	if itr.r[itr.pos] == '\x00' {
		return false
	}
	elemStart := itr.pos
	itr.pos++
	n, err := itr.r.validateKey(itr.pos, itr.end)
	itr.pos += n
	if err != nil {
		itr.err = err
		return false
	}

	itr.elem.value.start = elemStart
	itr.elem.value.offset = itr.pos
	itr.elem.value.data = itr.r

	n, err = itr.elem.value.validate(false)
	itr.pos += n
	if err != nil {
		itr.err = err
		return false
	}
	return true
}

// Element returns the current element of the ReaderIterator. The pointer that it returns will
// _always_ be the same for a given ReaderIterator.
func (itr *ReaderIterator) Element() *Element {
	return itr.elem
}

// Err returns the error that occurred when iterating, or nil if none occurred.
func (itr *ReaderIterator) Err() error {
	return itr.err
}
