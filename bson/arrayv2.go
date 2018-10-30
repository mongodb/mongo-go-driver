package bson

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"

	"github.com/mongodb/mongo-go-driver/bson/bsoncore"
	"github.com/mongodb/mongo-go-driver/bson/bsontype"
)

// ErrNilArray indicates that an operation was attempted on a nil *Array.
var ErrNilArray = errors.New("array is nil")

// Array represents an array in BSON.
type Arrayv2 struct {
	values []Valuev2
	doc    *Documentv2
}

// NewArray creates a new array with the specified value.
func NewArrayv2(values ...Valuev2) *Arrayv2 {
	arr := &Arrayv2{values: make([]Valuev2, len(values))}
	copy(arr.values, values)
	return arr
}

// Len returns the number of elements in the array.
func (a *Arrayv2) Len() int {
	if a == nil {
		return 0
	}
	return len(a.values)
}

// Reset clears all elements from the array.
func (a *Arrayv2) Reset() {
	if a == nil {
		return
	}

	for idx := range a.values {
		a.values[idx] = a.values[idx].reset()
	}
	a.values = a.values[:0]
}

// Index functions in a similar way to a Go native array or slice, that is, if the given index is
// out of bounds, this method will panic. Len can be used to retrieve the length of this Array.
func (a *Arrayv2) Index(index uint) Valuev2 { return a.values[index] }

func (a *Arrayv2) lookupTraverse(index uint, keys ...string) (Elementv2, error) {
	if a == nil {
		return Elementv2{}, KeyNotFound{}
	}
	if index > uint(len(a.values)) {
		return Elementv2{}, KeyNotFound{}
	}
	val := a.values[index]

	if len(keys) == 0 {
		return Elementv2{Key: strconv.Itoa(int(index)), Value: val}, nil
	}

	var err error
	var elem Elementv2
	switch val.Type() {
	case bsontype.EmbeddedDocument:
		elem, err = val.Document().LookupElementErr(keys...)
	case bsontype.Array:
		index, err := strconv.ParseUint(keys[0], 10, 0)
		if err != nil {
			err = KeyNotFound{}
			break
		}

		elem, err = val.Array().lookupTraverse(uint(index), keys[1:]...)
	default:
		err = KeyNotFound{Type: elem.Value.Type()}
	}

	switch tt := err.(type) {
	case KeyNotFound:
		tt.Depth++
		return Elementv2{}, tt
	case nil:
		return elem, nil
	default:
		return Elementv2{}, err
	}
}

// Append adds the given values to the end of the array.
//
// Append is safe to call on a nil Array.
func (a *Arrayv2) Append(values ...Valuev2) *Arrayv2 {
	if a == nil {
		a = &Arrayv2{values: make([]Valuev2, 0, len(values))}
	}
	a.values = append(a.values, values...)
	return a
}

// Prepend adds the given values to the beginning of the array.
//
// Append is safe to call on a nil Array.
func (a *Arrayv2) Prepend(values ...Valuev2) *Arrayv2 {
	if a == nil {
		a = &Arrayv2{values: make([]Valuev2, 0, len(values))}
	}
	a.values = append(a.values, values...)
	copy(a.values[len(values):], a.values)
	copy(a.values[:len(values)], values)

	return a
}

// Set replaces the value at the given index with the parameter value. It panics if the index is
// out of bounds.
func (a *Arrayv2) Set(index uint, value Valuev2) *Arrayv2 {
	a.values[index] = value
	return a
}

// Delete removes the value at the given index from the array.
func (a *Arrayv2) Delete(index uint) Valuev2 {
	if index >= uint(len(a.values)) {
		return Valuev2{}
	}

	value := a.values[index]
	a.values = append(a.values[:index], a.values[index+1:]...)

	return value
}

// String implements the fmt.Stringer interface.
func (a *Arrayv2) String() string {
	var buf bytes.Buffer
	buf.Write([]byte("bson.Array["))
	for idx, val := range a.values {
		if idx > 0 {
			buf.Write([]byte(", "))
		}
		fmt.Fprintf(&buf, "%s", val)
	}
	buf.WriteByte(']')

	return buf.String()
}

// MarshalBSONValue implements the bsoncodec.ValueMarshaler interface.
func (a *Arrayv2) MarshalBSONValue() (bsontype.Type, []byte, error) {
	if a == nil {
		return bsontype.Null, nil, nil
	}

	idx, dst := bsoncore.AppendArrayStart(nil)
	for idx, value := range a.values {
		t, data, err := value.MarshalBSONValue()
		if err != nil {
			return bsontype.Type(0), nil, err
		}
		dst = append(dst, byte(t))
		dst = append(dst, strconv.Itoa(idx)...)
		dst = append(dst, 0x00)
		dst = append(dst, data...)
	}
	dst, err := bsoncore.AppendArrayEnd(dst, idx)
	if err != nil {
		return bsontype.Type(0), nil, err
	}
	return bsontype.Array, dst, nil
}

// UnmarshalBSONValue implements the bsoncodec.ValueUnmarshaler interface.
func (a *Arrayv2) UnmarshalBSONValue(t bsontype.Type, data []byte) error {
	if a == nil {
		return ErrNilArray
	}
	a.Reset()

	elements, err := Raw(data).Elements()
	if err != nil {
		return err
	}

	for _, elem := range elements {
		var val Valuev2
		rawval := elem.Value()
		err = val.UnmarshalBSONValue(rawval.Type, rawval.Value)
		if err != nil {
			return err
		}
		a.values = append(a.values, val)
	}
	return nil
}

// Equal compares this document to another, returning true if they are equal.
func (a *Arrayv2) Equal(a2 *Arrayv2) bool {
	if a == nil && a2 == nil {
		return true
	}

	if a == nil || a2 == nil {
		return false
	}

	if len(a.values) != len(a2.values) {
		return false
	}

	for idx := range a.values {
		if !a.values[idx].Equal(a2.values[idx]) {
			return false
		}
	}

	return true
}

// embed implements the Embedabble interface.
func (a *Arrayv2) embed() {}
