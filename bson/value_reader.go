package bson

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

var EOA = errors.New("end of array")
var EOD = errors.New("end of document")

type vrMode int

const (
	_ vrMode = iota
	vrTopLevel
	vrDocument
	vrArray
	vrValue
	vrElement
	vrCodeWithScope
)

func (vm vrMode) String() string {
	var str string

	switch vm {
	case vrTopLevel:
		str = "TopLevel"
	case vrDocument:
		str = "DocumentMode"
	case vrArray:
		str = "ArrayMode"
	case vrValue:
		str = "ValueMode"
	case vrElement:
		str = "ElementMode"
	case vrCodeWithScope:
		str = "CodeWithScopeMode"
	default:
		str = "UnknownMode"
	}

	return str
}

// valueReader is for reading BSON values.
type valueReader struct {
	size   int64
	offset int64
	d      []byte

	stack []vrState
	frame int64
}

func (vr *valueReader) growStack() {
	if vr.frame+1 >= int64(cap(vr.stack)) {
		// double it
		buf := make([]vrState, 2*cap(vr.stack)+1)
		copy(buf, vr.stack)
		vr.stack = buf
	}
	vr.stack = vr.stack[:len(vr.stack)+1]
}

func (vr *valueReader) pushDocument() error {
	vr.growStack()
	vr.frame++

	vr.stack[vr.frame].mode = vrDocument
	vr.stack[vr.frame].offset = 0

	size, err := vr.readLength()
	if err != nil {
		return err
	}
	vr.stack[vr.frame].size = int64(size)

	return nil
}

func (vr *valueReader) pushArray() error {
	vr.growStack()
	vr.frame++

	vr.stack[vr.frame].mode = vrArray
	vr.stack[vr.frame].offset = 0

	size, err := vr.readLength()
	if err != nil {
		return err
	}
	vr.stack[vr.frame].size = int64(size)

	return nil
}

func (vr *valueReader) pushElement(t Type) {
	vr.growStack()
	vr.frame++

	vr.stack[vr.frame].mode = vrElement
	vr.stack[vr.frame].vType = t
	vr.stack[vr.frame].size = vr.stack[vr.frame-1].size
	vr.stack[vr.frame].offset = vr.stack[vr.frame-1].offset
}

func (vr *valueReader) pushValue(t Type) {
	vr.growStack()
	vr.frame++

	vr.stack[vr.frame].mode = vrValue
	vr.stack[vr.frame].vType = t
	vr.stack[vr.frame].size = vr.stack[vr.frame-1].size
	vr.stack[vr.frame].offset = vr.stack[vr.frame-1].offset
}

func (vr *valueReader) pushCodeWithScope(size int64) {
}

func (vr *valueReader) pop() {
	switch vr.stack[vr.frame].mode {
	case vrElement, vrValue:
		vr.stack[vr.frame-1].offset = vr.stack[vr.frame].offset // carry the offset backward
		vr.frame--
	case vrDocument, vrArray:
		vr.stack[vr.frame-2].offset += vr.stack[vr.frame].offset // advance the offset of the previous vrDocument/TopLevel/etc...

		vr.frame -= 2 // we pop twice to jump over the vrElement: vrDocument -> vrElement -> vrDocument/TopLevel/etc...
	}
}

type vrState struct {
	mode   vrMode
	vType  Type
	size   int64
	offset int64
}

func newValueReader(b []byte) *valueReader {
	stack := make([]vrState, 1, 5)
	stack[0] = vrState{
		mode: vrTopLevel,
	}
	return &valueReader{
		d:     b,
		stack: stack,
	}
}

func (vr *valueReader) invalidTransitionErr() error {
	if vr.frame == 0 {
		return fmt.Errorf("invalid state transition <nil> -> %s", vr.stack[vr.frame].mode)
	}
	return fmt.Errorf("invalid state transition %s -> %s", vr.stack[vr.frame-1].mode, vr.stack[vr.frame].mode)
}

func (vr *valueReader) typeError(t Type) error {
	return fmt.Errorf("positioned on %s, but attempted to read %s", vr.stack[vr.frame].vType, t)
}

func (vr *valueReader) invalidDocumentLengthError() error {
	return fmt.Errorf("document length is invalid, given size is %d, but null byte found at %d", vr.stack[vr.frame].size, vr.stack[vr.frame].offset)
}

func (vr *valueReader) Type() Type {
	return vr.stack[vr.frame].vType
}

func (vr *valueReader) Skip() error {
	panic("not implemented")
}

func (vr *valueReader) ReadArray() (ArrayReader, error) {
	switch vr.stack[vr.frame].mode {
	case vrElement, vrValue:
		if vr.stack[vr.frame].vType != TypeArray {
			return nil, vr.typeError(TypeArray)
		}
	default:
		return nil, vr.invalidTransitionErr()
	}

	err := vr.pushArray()
	if err != nil {
		return nil, err
	}

	return vr, nil
}

func (vr *valueReader) ReadBinary() (b []byte, btype byte, err error) {
	switch vr.stack[vr.frame].mode {
	case vrElement, vrValue:
		if vr.stack[vr.frame].vType != TypeBinary {
			return nil, 0, vr.typeError(TypeBinary)
		}
	default:
		return nil, 0, vr.invalidTransitionErr()
	}

	length, err := vr.readLength()
	if err != nil {
		return nil, 0, err
	}

	btype, err = vr.readByte()
	if err != nil {
		return nil, 0, err
	}
	b, err = vr.readBytes(length)
	if err != nil {
		return nil, 0, err
	}

	return b, btype, nil
}

func (vr *valueReader) ReadBoolean() (bool, error) {
	switch vr.stack[vr.frame].mode {
	case vrElement, vrValue:
		if vr.stack[vr.frame].vType != TypeBoolean {
			return false, vr.typeError(TypeBoolean)
		}
	default:
		return false, vr.invalidTransitionErr()
	}

	b, err := vr.readByte()
	if err != nil {
		return false, err
	}

	if b > 1 {
		return false, fmt.Errorf("invalid byte for boolean, %b", b)
	}

	vr.pop()
	return b == 1, nil
}

func (vr *valueReader) ReadDocument() (DocumentReader, error) {
	switch vr.stack[vr.frame].mode {
	case vrTopLevel:
		// read size
		size, err := vr.readLength()
		if err != nil {
			return nil, err
		}
		vr.size, vr.stack[vr.frame].size = int64(size), int64(size)
		return vr, nil
	case vrElement, vrValue:
		if vr.stack[vr.frame].vType != TypeEmbeddedDocument {
			return nil, vr.typeError(TypeEmbeddedDocument)
		}
	default:
		return nil, vr.invalidTransitionErr()
	}

	err := vr.pushDocument()
	if err != nil {
		return nil, err
	}

	return vr, nil
}

func (vr *valueReader) ReadCodeWithScope() (code string, dr DocumentReader, err error) {
	panic("not implemented")
}

func (vr *valueReader) ReadDBPointer() (ns string, oid objectid.ObjectID, err error) {
	panic("not implemented")
}

func (vr *valueReader) ReadDateTime() (int64, error) {
	panic("not implemented")
}

func (vr *valueReader) ReadDecimal128() (decimal.Decimal128, error) {
	panic("not implemented")
}

func (vr *valueReader) ReadDouble() (float64, error) {
	panic("not implemented")
}

func (vr *valueReader) ReadInt32() (int32, error) {
	panic("not implemented")
}

func (vr *valueReader) ReadInt64() (int64, error) {
	panic("not implemented")
}

func (vr *valueReader) ReadJavascript() (code string, err error) {
	panic("not implemented")
}

func (vr *valueReader) ReadMaxKey() error {
	panic("not implemented")
}

func (vr *valueReader) ReadMinKey() error {
	panic("not implemented")
}

func (vr *valueReader) ReadNull() error {
	panic("not implemented")
}

func (vr *valueReader) ReadObjectID() (objectid.ObjectID, error) {
	panic("not implemented")
}

func (vr *valueReader) ReadRegex() (pattern string, options string, err error) {
	panic("not implemented")
}

func (vr *valueReader) ReadString() (string, error) {
	panic("not implemented")
}

func (vr *valueReader) ReadSymbol() (symbol string, err error) {
	panic("not implemented")
}

func (vr *valueReader) ReadTimeStamp(t uint32, i uint32, err error) {
	panic("not implemented")
}

func (vr *valueReader) ReadUndefined() error {
	panic("not implemented")
}

func (vr *valueReader) ReadElement() (string, ValueReader, error) {
	switch vr.stack[vr.frame].mode {
	case vrTopLevel, vrDocument:
	default:
		return "", nil, vr.invalidTransitionErr()
	}

	t, err := vr.readByte()
	if err != nil {
		return "", nil, err
	}

	if t == 0 {
		if vr.stack[vr.frame].offset != vr.stack[vr.frame].size {
			return "", nil, vr.invalidDocumentLengthError()
		}

		vr.pop()
		return "", nil, EOD
	}

	name, err := vr.readCString()
	if err != nil {
		return "", nil, err
	}

	vr.pushElement(Type(t))
	return name, vr, nil
}

func (vr *valueReader) ReadValue() (ValueReader, error) {
	switch vr.stack[vr.frame].mode {
	case vrArray:
	default:
		return nil, vr.invalidTransitionErr()
	}

	t, err := vr.readByte()
	if err != nil {
		return nil, err
	}

	if t == 0 {
		if vr.stack[vr.frame].offset != vr.stack[vr.frame].size {
			return nil, vr.invalidDocumentLengthError()
		}

		vr.pop()
		return nil, EOA
	}

	_, err = vr.readCString()
	if err != nil {
		return nil, err
	}

	vr.pushValue(Type(t))
	return vr, nil
}

func (vr *valueReader) readBytes(length int32) ([]byte, error) {
	if vr.offset+int64(length) > int64(len(vr.d)) {
		return nil, io.EOF
	}

	start := vr.offset
	vr.offset += int64(length)
	vr.stack[vr.frame].offset += int64(length)
	return vr.d[start : start+int64(length)], nil
}

func (vr *valueReader) readByte() (byte, error) {
	if vr.offset+1 > int64(len(vr.d)) {
		return 0x0, io.EOF
	}

	vr.offset++
	vr.stack[vr.frame].offset++
	return vr.d[vr.offset-1], nil
}

func (vr *valueReader) readCString() (string, error) {
	idx := bytes.IndexByte(vr.d[vr.offset:], 0x00)
	if idx < 0 {
		return "", io.EOF
	}
	start := vr.offset
	// idx does not include the null byte
	vr.offset += int64(idx) + 1
	vr.stack[vr.frame].offset += int64(idx) + 1
	return string(vr.d[start : start+int64(idx)]), nil
}

func (vr *valueReader) readLength() (int32, error) {
	if vr.offset+4 > int64(len(vr.d)) {
		return 0, io.EOF
	}

	idx := vr.offset
	vr.offset += 4
	vr.stack[vr.frame].offset += 4
	return (int32(vr.d[idx]) | int32(vr.d[idx+1])<<8 | int32(vr.d[idx+2])<<16 | int32(vr.d[idx+3])<<24), nil
}
