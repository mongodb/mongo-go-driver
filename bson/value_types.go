package bson

import "github.com/10gen/mongo-go-driver/bson/objectid"

// Binary represents a BSON binary value.
type Binary struct {
	Subtype byte
	Data    []byte
}

// Undefined represents the BSON undefined value.
var Undefined struct{}

// Null represents the BSON null value.
var Null struct{}

// Regex represents a BSON regex value.
type Regex struct {
	Pattern string
	Options string
}

// DBPointer represents a BSON dbpointer value.
type DBPointer struct {
	DB      string
	Pointer objectid.ObjectID
}

// JavaScriptCode represents a BSON JavaScript code value.
type JavaScriptCode string

// Symbol represents a BSON symbol value.
type Symbol string

// CodeWithScope represents a BSON JavaScript code with scope value.
type CodeWithScope struct {
	Code  string
	Scope *Document
}

// Timestamp represents a BSON timestamp value.
type Timestamp struct {
	T uint32
	I uint32
}

// MinKey represents the BSON maxkey value.
var MinKey struct{}

// MaxKey represents the BSON minkey value.
var MaxKey struct{}
