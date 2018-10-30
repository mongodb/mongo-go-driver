// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"errors"
)

// ErrInvalidReadOnlyDocument indicates that the underlying bytes of a bson.Reader are invalid.
var ErrInvalidReadOnlyDocument = errors.New("invalid read-only document")

// ErrInvalidKey indicates that the BSON representation of a key is missing a null terminator.
var ErrInvalidKey = errors.New("invalid document key")

// ErrInvalidArrayKey indicates that a key that isn't a positive integer was used to lookup an
// element in an array.
var ErrInvalidArrayKey = errors.New("invalid array key")

// ErrInvalidLength indicates that a length in a binary representation of a BSON document is invalid.
var ErrInvalidLength = errors.New("document length is invalid")

// ErrEmptyKey indicates that no key was provided to a Lookup method.
var ErrEmptyKey = errors.New("empty key provided")

// ErrNilElement indicates that a nil element was provided when none was expected.
var ErrNilElement = errors.New("element is nil")

// ErrNilDocument indicates that an operation was attempted on a nil *bson.Document.
var ErrNilDocument = errors.New("document is nil")

// ErrInvalidDocumentType indicates that a type which doesn't represent a BSON document was
// was provided when a document was expected.
var ErrInvalidDocumentType = errors.New("invalid document type")

// ErrInvalidDepthTraversal indicates that a provided path of keys to a nested value in a document
// does not exist.
//
// TODO(skriptble): This error message is pretty awful.
// Please fix.
var ErrInvalidDepthTraversal = errors.New("invalid depth traversal")

// ErrElementNotFound indicates that an Element matching a certain condition does not exist.
var ErrElementNotFound = errors.New("element not found")

// ErrOutOfBounds indicates that an index provided to access something was invalid.
var ErrOutOfBounds = errors.New("out of bounds")
