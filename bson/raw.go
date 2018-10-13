package bson

import (
	"bytes"
	"errors"
	"reflect"

	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
	"github.com/mongodb/mongo-go-driver/bson/bsonrw"
	"github.com/mongodb/mongo-go-driver/bson/bsontype"
)

// ErrNilRegistry is returned when the provided registry is nil.
var ErrNilRegistry = errors.New("Registry cannot be nil")

// RawElement represents a BSON element in byte form. This type provides a simple way to
// transform a slice of bytes into a BSON element and extract information from it.
type RawElement []byte

// Key returns the key for this element. It panics if there is an error reading the key.
func (re RawElement) Key() string {
	key, err := re.KeyErr()
	if err != nil {
		panic(err)
	}
	return key
}

// KeyErr returns the key for this element, returning an error if the element is not valid.
func (re RawElement) KeyErr() (string, error) {
	return "", nil
}

// Value returns the value of this element. It panics if there is an error reading the value.
func (re RawElement) Value() Value {
	val, err := re.ValueErr()
	if err != nil {
		panic(err)
	}
	return val
}

// ValueErr returns the value for this element, returning an error if the element is not valid.
func (re RawElement) ValueErr() (Value, error) {
	return Value{}, nil
}

// Validate ensures re is a valid BSON element.
func (re RawElement) Validate() error {
	return nil
}

// String implements the fmt.Stringer interface. The output will be in extended JSON format.
func (re RawElement) String() string {
	return ""
}

// RawValue represents a BSON value in byte form. It can be used to hold unprocessed BSON or to
// defer processing of BSON. Type is the BSON type of the value and Value are the raw bytes that
// represent the element.
type RawValue struct {
	Type  bsontype.Type
	Value []byte

	r *bsoncodec.Registry
}

// Unmarshal deserializes BSON into the provided val. If RawValue cannot be unmarshaled into val, an
// error is returned. This method will use the registry used to create the RawValue, if the RawValue
// was created from partial BSON processing, or it will use the default registry. Users wishing to
// specify the registry to use should use UnmarshalWithRegistry.
func (rv RawValue) Unmarshal(val interface{}) error {
	reg := rv.r
	if reg == nil {
		reg = DefaultRegistry
	}
	return rv.UnmarshalWithRegistry(reg, val)
}

// Equal compares rv and rv2 and returns true if they are equal.
func (rv RawValue) Equal(rv2 RawValue) bool {
	if rv.Type != rv2.Type {
		return false
	}

	if !bytes.Equal(rv.Value, rv2.Value) {
		return false
	}

	return true
}

// UnmarshalWithRegistry performs the same unmarshalling as Unmarshal but uses the provided registry
// instead of the one attached or the default registry.
func (rv RawValue) UnmarshalWithRegistry(r *bsoncodec.Registry, val interface{}) error {
	if r == nil {
		return ErrNilRegistry
	}

	vr := bsonrw.NewBSONValueReaderValueMode(rv.Type, rv.Value)
	dec, err := r.LookupDecoder(reflect.TypeOf(val))
	if err != nil {
		return err
	}
	return dec.DecodeValue(bsoncodec.DecodeContext{Registry: r}, vr, val)
}

// RawArray represnts a BSON array in byte form. It treats the bytes as if they were an array.
type RawArray []byte
