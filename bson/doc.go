// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package bson is a library for reading, writing, and manipulating BSON. The
// library has three types for representing BSON.
//
// The Reader type is used to validate and retrieve elements from a byte slice.
// If you are not manipulating the underlying document and just want to validate
// or ensure the document has certain keys, this is the type you should use.
//
// The Document type is a more generic type that can do all the read actions the
// Reader can do and allows for manipulation of documents. The main usecase for
// this type is reading some BSON and then adding, changing, or removing
// elements or to build a document that will need to be manipulated later. If
// the document can be created in a single pass without the need to do any
// lookups, the Builder type is might be more appropriate.
//
// The Builder type (in the "bson/builder" package) is used to create a BSON
// document. The type only allows the iterative building of a document, so there
// is no way to verify the contents outside of writing the document. If you have
// a Builder and need to conditionally add a field, you can write the document
// to a byte slice and use the Reader type to lookup the desired document, but
// in this case you should probably use a Document instead.
//
// The Element type represents a BSON element and the Value type represents an
// individual value for a BSON element.
//
// The Encoder and Decoder types can be used to marshal a type to an io.Writer
// or to unmarshal into a type from an io.Reader. These types will use
// reflection and evaluate struct tags unless the provided types implements the
// Marshaler or Unmarshaler interfaces. The Builder and Reader types can be used
// to implement these interfaces for types.
//
// The DocumentEncoder type can be used to encode a type to a Document instead
// of an io.Writer. This is useful if some additional manipulation is required
// after encoding a document. This type supports the same encoding behavior as
// the Encoder type.
package bson
