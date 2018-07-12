package bson

import (
	"bufio"
	"errors"
	"fmt"
	"io"

	"github.com/mongodb/mongo-go-driver/bson-codec-design/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson-codec-design/bson/objectid"
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
	r      *bufio.Reader

	state vrState
}

type vrState struct {
	prev *vrState

	mode   vrMode
	vType  Type
	size   int64
	offset int64
}

func (vrs *vrState) push() {
	vrs.prev = &vrState{
		mode:   vrs.mode,
		vType:  vrs.vType,
		size:   vrs.size,
		offset: vrs.offset,
	}
}

func (vrs *vrState) pop() {
	if vrs.mode == vrTopLevel {
		return // vrTopLevel has no previous stack frame
	}

	vrs.mode = vrs.prev.mode
	vrs.vType = vrs.prev.vType
	vrs.size = vrs.prev.size
	vrs.offset = vrs.prev.offset

	vrs.prev = vrs.prev.prev
}

func newValueReader(r io.Reader) *valueReader {
	return &valueReader{
		r: bufio.NewReader(r),
		state: vrState{
			mode: vrTopLevel,
		},
	}
}

func (vr *valueReader) invalidTransitionErr() error {
	if vr.state.prev == nil {
		return fmt.Errorf("invalid state transition <nil> -> %s", vr.state.mode)
	}
	return fmt.Errorf("invalid state transition %s -> %s", vr.state.prev.mode, vr.state.mode)
}

func (vr *valueReader) typeError(t Type) error {
	return fmt.Errorf("positioned on %s, but attempted to read %s", vr.state.vType, t)
}

func (vr *valueReader) invalidDocumentLengthError() error {
	return fmt.Errorf("document length is invalid, given size is %d, but null byte found at %d", vr.state.size, vr.state.offset)
}

func (vr *valueReader) Type() Type {
	return vr.state.vType
}

func (vr *valueReader) Skip() error {
	panic("not implemented")
}

func (vr *valueReader) ReadArray() (ArrayReader, error) {
	panic("not implemented")
}

func (vr *valueReader) ReadBinary() (b []byte, btype byte, err error) {
	panic("not implemented")
}

func (vr *valueReader) ReadBoolean() (bool, error) {
	switch vr.state.mode {
	case vrElement, vrValue:
		if vr.state.vType != TypeBoolean {
			return false, vr.typeError(TypeBoolean)
		}
	default:
		return false, vr.invalidTransitionErr()
	}

	b, err := vr.r.ReadByte()
	vr.offset += 1
	vr.state.offset += 1
	if err != nil {
		return false, err
	}

	if b > 1 {
		return false, fmt.Errorf("invalid byte for boolean, %b", b)
	}

	offset := vr.state.offset
	vr.state.pop()
	vr.state.offset = offset
	return b == 1, nil
}

func (vr *valueReader) ReadDocument() (DocumentReader, error) {
	switch vr.state.mode {
	case vrTopLevel:
		// read size
		size, err := vr.readLength()
		if err != nil {
			return nil, err
		}
		vr.size, vr.state.size = int64(size), int64(size)
		return vr, nil
	case vrElement, vrValue:
		if vr.state.vType != TypeEmbeddedDocument {
			return nil, vr.typeError(TypeEmbeddedDocument)
		}
	default:
		return nil, vr.invalidTransitionErr()
	}

	vr.state.push()
	vr.state.mode = vrDocument
	vr.state.offset = 0

	size, err := vr.readLength()
	if err != nil {
		return nil, err
	}
	vr.state.size = int64(size)

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
	switch vr.state.mode {
	case vrTopLevel, vrDocument:
	default:
		return "", nil, vr.invalidTransitionErr()
	}

	t, err := vr.r.ReadByte()
	vr.offset++
	vr.state.offset++
	if err != nil {
		return "", nil, err
	}

	if t == 0 {
		if vr.state.offset != vr.state.size {
			return "", nil, vr.invalidDocumentLengthError()
		}

		vr.state.pop()
		return "", nil, EOD
	}

	nameBytes, err := vr.r.ReadSlice(0x00)
	vr.offset += int64(len(nameBytes))
	vr.state.offset += int64(len(nameBytes))
	if err != nil {
		return "", nil, err
	}

	vr.state.push()
	vr.state.vType = Type(t)
	vr.state.mode = vrElement
	return string(nameBytes[:len(nameBytes)-1]), vr, nil
}

func (vr *valueReader) Next() bool {
	panic("not implemented")
}

func (vr *valueReader) ReadValue() (ValueReader, error) {
	panic("not implemented")
}

func (vr *valueReader) readLength() (int32, error) {
	var buf [4]byte
	n, err := io.ReadFull(vr.r, buf[:])
	vr.offset += int64(n)
	vr.state.offset += int64(n)
	if err != nil {
		return 0, err
	}
	return (int32(buf[0]) | int32(buf[1])<<8 | int32(buf[2])<<16 | int32(buf[3])<<24), nil
}
