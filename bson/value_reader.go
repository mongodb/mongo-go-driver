package bson

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

// ErrEOA is the error returned when the end of a BSON array has been reached.
var ErrEOA = errors.New("end of array")

// ErrEOD is the error returned when the end of a BSON document has been reached.
var ErrEOD = errors.New("end of document")

type vrState struct {
	mode  mode
	vType Type
	end   int64
}

// valueReader is for reading BSON values.
type valueReader struct {
	offset int64
	d      []byte

	stack []vrState
	frame int64
}

// NewValueReader returns a ValueReader using b for the underlying BSON
// representation.
func NewValueReader(b []byte) ValueReader {
	return newValueReader(b)
}

func newValueReader(b []byte) *valueReader {
	stack := make([]vrState, 1, 5)
	stack[0] = vrState{
		mode: mTopLevel,
	}
	return &valueReader{
		d:     b,
		stack: stack,
	}
}

func (vr *valueReader) advanceFrame() {
	if vr.frame+1 >= int64(len(vr.stack)) { // We need to grow the stack
		length := len(vr.stack)
		if length+1 >= cap(vr.stack) {
			// double it
			buf := make([]vrState, 2*cap(vr.stack)+1)
			copy(buf, vr.stack)
			vr.stack = buf
		}
		vr.stack = vr.stack[:length+1]
	}
	vr.frame++

	// Clean the stack
	vr.stack[vr.frame].mode = 0
	vr.stack[vr.frame].vType = 0
	vr.stack[vr.frame].end = 0
}

func (vr *valueReader) pushDocument() error {
	vr.advanceFrame()

	vr.stack[vr.frame].mode = mDocument

	size, err := vr.readLength()
	if err != nil {
		return err
	}
	vr.stack[vr.frame].end = int64(size) + vr.offset - 4

	return nil
}

func (vr *valueReader) pushArray() error {
	vr.advanceFrame()

	vr.stack[vr.frame].mode = mArray

	size, err := vr.readLength()
	if err != nil {
		return err
	}
	vr.stack[vr.frame].end = int64(size) + vr.offset - 4

	return nil
}

func (vr *valueReader) pushElement(t Type) {
	vr.advanceFrame()

	vr.stack[vr.frame].mode = mElement
	vr.stack[vr.frame].vType = t
}

func (vr *valueReader) pushValue(t Type) {
	vr.advanceFrame()

	vr.stack[vr.frame].mode = mValue
	vr.stack[vr.frame].vType = t
}

func (vr *valueReader) pushCodeWithScope() (int64, error) {
	vr.advanceFrame()

	vr.stack[vr.frame].mode = mCodeWithScope

	size, err := vr.readLength()
	if err != nil {
		return 0, err
	}
	vr.stack[vr.frame].end = int64(size) + vr.offset - 4

	return int64(size), nil
}

func (vr *valueReader) pop() {
	switch vr.stack[vr.frame].mode {
	case mElement, mValue:
		vr.frame--
	case mDocument, mArray, mCodeWithScope:
		vr.frame -= 2 // we pop twice to jump over the vrElement: vrDocument -> vrElement -> vrDocument/TopLevel/etc...
	}
}

func (vr *valueReader) invalidTransitionErr(destination mode) error {
	te := transitionError{
		current:     vr.stack[vr.frame].mode,
		destination: destination,
	}
	if vr.frame != 0 {
		te.parent = vr.stack[vr.frame-1].mode
	}
	return te
}

func (vr *valueReader) typeError(t Type) error {
	return fmt.Errorf("positioned on %s, but attempted to read %s", vr.stack[vr.frame].vType, t)
}

func (vr *valueReader) invalidDocumentLengthError() error {
	return fmt.Errorf("document is invalid, end byte is at %d, but null byte found at %d", vr.stack[vr.frame].end, vr.offset)
}

func (vr *valueReader) ensureElementValue(t Type, destination mode) error {
	switch vr.stack[vr.frame].mode {
	case mElement, mValue:
		if vr.stack[vr.frame].vType != t {
			return vr.typeError(t)
		}
	default:
		return vr.invalidTransitionErr(destination)
	}

	return nil
}

func (vr *valueReader) Type() Type {
	return vr.stack[vr.frame].vType
}

func (vr *valueReader) Skip() error {
	switch vr.stack[vr.frame].mode {
	case mElement, mValue:
	default:
		return vr.invalidTransitionErr(0)
	}

	defer vr.pop()

	switch vr.stack[vr.frame].vType {
	case TypeArray, TypeEmbeddedDocument, TypeCodeWithScope:
		length, err := vr.readLength()
		if err != nil {
			return err
		}
		return vr.skipBytes(length - 4) // skipBytes skips that exact number of bytes, so we need to take off the lenght bytes
	case TypeBinary:
		length, err := vr.readLength()
		if err != nil {
			return err
		}
		// We need to skip length + 1 (for the subtype byte)
		return vr.skipBytes(length + 1)
	case TypeBoolean:
		return vr.skipBytes(1)
	case TypeDBPointer:
		length, err := vr.readLength()
		if err != nil {
			return err
		}
		// skip length + 12 for the ObjectID
		return vr.skipBytes(length + 12)
	case TypeDateTime, TypeDouble, TypeInt64, TypeTimestamp:
		return vr.skipBytes(8)
	case TypeDecimal128:
		return vr.skipBytes(16)
	case TypeInt32:
		return vr.skipBytes(4)
	case TypeJavaScript, TypeString, TypeSymbol:
		length, err := vr.readLength()
		if err != nil {
			return err
		}
		return vr.skipBytes(length)
	case TypeMaxKey, TypeMinKey, TypeNull, TypeUndefined:
		return nil
	case TypeObjectID:
		return vr.skipBytes(12)
	case TypeRegex:
		err := vr.skipCString()
		if err != nil {
			return err
		}
		return vr.skipCString()
	default:
		return fmt.Errorf("attempted to skip unknown BSON type %v", vr.stack[vr.frame].vType)
	}
}

func (vr *valueReader) ReadArray() (ArrayReader, error) {
	if err := vr.ensureElementValue(TypeArray, mArray); err != nil {
		return nil, err
	}

	err := vr.pushArray()
	if err != nil {
		return nil, err
	}

	return vr, nil
}

func (vr *valueReader) ReadBinary() (b []byte, btype byte, err error) {
	if err := vr.ensureElementValue(TypeBinary, 0); err != nil {
		return nil, 0, err
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

	vr.pop()
	return b, btype, nil
}

func (vr *valueReader) ReadBoolean() (bool, error) {
	if err := vr.ensureElementValue(TypeBoolean, 0); err != nil {
		return false, err
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
	case mTopLevel:
		// read size
		size, err := vr.readLength()
		if err != nil {
			return nil, err
		}
		vr.stack[vr.frame].end = int64(size) + vr.offset - 4
		return vr, nil
	case mElement, mValue:
		if vr.stack[vr.frame].vType != TypeEmbeddedDocument {
			return nil, vr.typeError(TypeEmbeddedDocument)
		}
	default:
		return nil, vr.invalidTransitionErr(mDocument)
	}

	err := vr.pushDocument()
	if err != nil {
		return nil, err
	}

	return vr, nil
}

func (vr *valueReader) ReadCodeWithScope() (code string, dr DocumentReader, err error) {
	if err := vr.ensureElementValue(TypeCodeWithScope, 0); err != nil {
		return "", nil, err
	}

	totalLength, err := vr.readLength()
	if err != nil {
		return "", nil, err
	}
	strLength, err := vr.readLength()
	if err != nil {
		return "", nil, err
	}
	strBytes, err := vr.readBytes(strLength)
	if err != nil {
		return "", nil, err
	}
	code = string(strBytes[:len(strBytes)-1])

	size, err := vr.pushCodeWithScope()
	if err != nil {
		return "", nil, err
	}

	// The total length should equal:
	// 4 (total length) + strLength + 4 (the length of str itself) + (document length)
	componentsLength := int64(4+strLength+4) + size
	if int64(totalLength) != componentsLength {
		return "", nil, fmt.Errorf(
			"length of CodeWithScope does not match lengths of components; total: %d; components: %d",
			totalLength, componentsLength,
		)
	}
	return code, vr, nil
}

func (vr *valueReader) ReadDBPointer() (ns string, oid objectid.ObjectID, err error) {
	if err := vr.ensureElementValue(TypeDBPointer, 0); err != nil {
		return "", oid, err
	}

	length, err := vr.readLength()
	if err != nil {
		return "", oid, err
	}

	sbytes, err := vr.readBytes(length)
	if err != nil {
		return "", oid, err
	}

	ns = string(sbytes[:len(sbytes)-1])

	oidbytes, err := vr.readBytes(12)
	if err != nil {
		return "", oid, err
	}

	copy(oid[:], oidbytes)

	vr.pop()
	return ns, oid, nil
}

func (vr *valueReader) ReadDateTime() (int64, error) {
	if err := vr.ensureElementValue(TypeDateTime, 0); err != nil {
		return 0, err
	}

	i, err := vr.readi64()
	if err != nil {
		return 0, err
	}

	vr.pop()
	return i, nil
}

func (vr *valueReader) ReadDecimal128() (decimal.Decimal128, error) {
	if err := vr.ensureElementValue(TypeDecimal128, 0); err != nil {
		return decimal.Decimal128{}, err
	}

	b, err := vr.readBytes(16)
	if err != nil {
		return decimal.Decimal128{}, err
	}

	l := binary.LittleEndian.Uint64(b[0:8])
	h := binary.LittleEndian.Uint64(b[8:16])

	vr.pop()
	return decimal.NewDecimal128(h, l), nil
}

func (vr *valueReader) ReadDouble() (float64, error) {
	if err := vr.ensureElementValue(TypeDouble, 0); err != nil {
		return 0, err
	}

	u, err := vr.readu64()
	if err != nil {
		return 0, err
	}

	vr.pop()
	return math.Float64frombits(u), nil
}

func (vr *valueReader) ReadInt32() (int32, error) {
	if err := vr.ensureElementValue(TypeInt32, 0); err != nil {
		return 0, err
	}

	vr.pop()
	return vr.readi32()
}

func (vr *valueReader) ReadInt64() (int64, error) {
	if err := vr.ensureElementValue(TypeInt64, 0); err != nil {
		return 0, err
	}

	vr.pop()
	return vr.readi64()
}

func (vr *valueReader) ReadJavascript() (code string, err error) {
	if err := vr.ensureElementValue(TypeJavaScript, 0); err != nil {
		return "", err
	}

	vr.pop()
	return vr.readString()
}

func (vr *valueReader) ReadMaxKey() error {
	if err := vr.ensureElementValue(TypeMaxKey, 0); err != nil {
		return err
	}

	vr.pop()
	return nil
}

func (vr *valueReader) ReadMinKey() error {
	if err := vr.ensureElementValue(TypeMinKey, 0); err != nil {
		return err
	}

	vr.pop()
	return nil
}

func (vr *valueReader) ReadNull() error {
	if err := vr.ensureElementValue(TypeNull, 0); err != nil {
		return err
	}

	vr.pop()
	return nil
}

func (vr *valueReader) ReadObjectID() (objectid.ObjectID, error) {
	if err := vr.ensureElementValue(TypeObjectID, 0); err != nil {
		return objectid.ObjectID{}, err
	}

	oidbytes, err := vr.readBytes(12)
	if err != nil {
		return objectid.ObjectID{}, err
	}

	var oid objectid.ObjectID
	copy(oid[:], oidbytes)

	vr.pop()
	return oid, nil
}

func (vr *valueReader) ReadRegex() (string, string, error) {
	if err := vr.ensureElementValue(TypeRegex, 0); err != nil {
		return "", "", err
	}

	pattern, err := vr.readCString()
	if err != nil {
		return "", "", err
	}

	options, err := vr.readCString()
	if err != nil {
		return "", "", err
	}

	vr.pop()
	return pattern, options, nil
}

func (vr *valueReader) ReadString() (string, error) {
	if err := vr.ensureElementValue(TypeString, 0); err != nil {
		return "", err
	}

	vr.pop()
	return vr.readString()
}

func (vr *valueReader) ReadSymbol() (symbol string, err error) {
	if err := vr.ensureElementValue(TypeSymbol, 0); err != nil {
		return "", err
	}

	vr.pop()
	return vr.readString()
}

func (vr *valueReader) ReadTimestamp() (t uint32, i uint32, err error) {
	if err := vr.ensureElementValue(TypeTimestamp, 0); err != nil {
		return 0, 0, err
	}

	i, err = vr.readu32()
	if err != nil {
		return 0, 0, err
	}

	t, err = vr.readu32()
	if err != nil {
		return 0, 0, err
	}

	vr.pop()
	return t, i, nil
}

func (vr *valueReader) ReadUndefined() error {
	if err := vr.ensureElementValue(TypeUndefined, 0); err != nil {
		return err
	}

	vr.pop()
	return nil
}

func (vr *valueReader) ReadElement() (string, ValueReader, error) {
	switch vr.stack[vr.frame].mode {
	case mTopLevel, mDocument:
	default:
		return "", nil, vr.invalidTransitionErr(mElement)
	}

	t, err := vr.readByte()
	if err != nil {
		return "", nil, err
	}

	if t == 0 {
		if vr.offset != vr.stack[vr.frame].end {
			return "", nil, vr.invalidDocumentLengthError()
		}

		vr.pop()
		return "", nil, ErrEOD
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
	case mArray:
	default:
		return nil, vr.invalidTransitionErr(mValue)
	}

	t, err := vr.readByte()
	if err != nil {
		return nil, err
	}

	if t == 0 {
		if vr.offset != vr.stack[vr.frame].end {
			return nil, vr.invalidDocumentLengthError()
		}

		vr.pop()
		return nil, ErrEOA
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
	return vr.d[start : start+int64(length)], nil
}

func (vr *valueReader) skipBytes(length int32) error {
	if vr.offset+int64(length) > int64(len(vr.d)) {
		return io.EOF
	}

	vr.offset += int64(length)
	return nil
}

func (vr *valueReader) readByte() (byte, error) {
	if vr.offset+1 > int64(len(vr.d)) {
		return 0x0, io.EOF
	}

	vr.offset++
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
	return string(vr.d[start : start+int64(idx)]), nil
}

func (vr *valueReader) skipCString() error {
	idx := bytes.IndexByte(vr.d[vr.offset:], 0x00)
	if idx < 0 {
		return io.EOF
	}
	// idx does not include the null byte
	vr.offset += int64(idx) + 1
	return nil
}

func (vr *valueReader) readString() (string, error) {
	length, err := vr.readLength()
	if err != nil {
		return "", err
	}

	if vr.d[vr.offset+int64(length)-1] != 0x00 {
		return "", fmt.Errorf("string does not end with null byte, but with %v", vr.d[vr.offset+int64(length)-1])
	}

	start := vr.offset
	vr.offset += int64(length)
	return string(vr.d[start : start+int64(length)-1]), nil
}

func (vr *valueReader) readLength() (int32, error) { return vr.readi32() }

func (vr *valueReader) readi32() (int32, error) {
	if vr.offset+4 > int64(len(vr.d)) {
		return 0, io.EOF
	}

	idx := vr.offset
	vr.offset += 4
	return (int32(vr.d[idx]) | int32(vr.d[idx+1])<<8 | int32(vr.d[idx+2])<<16 | int32(vr.d[idx+3])<<24), nil
}

func (vr *valueReader) readu32() (uint32, error) {
	if vr.offset+4 > int64(len(vr.d)) {
		return 0, io.EOF
	}

	idx := vr.offset
	vr.offset += 4
	return (uint32(vr.d[idx]) | uint32(vr.d[idx+1])<<8 | uint32(vr.d[idx+2])<<16 | uint32(vr.d[idx+3])<<24), nil
}

func (vr *valueReader) readi64() (int64, error) {
	if vr.offset+8 > int64(len(vr.d)) {
		return 0, io.EOF
	}

	idx := vr.offset
	vr.offset += 8
	return int64(vr.d[idx]) | int64(vr.d[idx+1])<<8 | int64(vr.d[idx+2])<<16 | int64(vr.d[idx+3])<<24 |
		int64(vr.d[idx+4])<<32 | int64(vr.d[idx+5])<<40 | int64(vr.d[idx+6])<<48 | int64(vr.d[idx+7])<<56, nil
}

func (vr *valueReader) readu64() (uint64, error) {
	if vr.offset+8 > int64(len(vr.d)) {
		return 0, io.EOF
	}

	idx := vr.offset
	vr.offset += 8
	return uint64(vr.d[idx]) | uint64(vr.d[idx+1])<<8 | uint64(vr.d[idx+2])<<16 | uint64(vr.d[idx+3])<<24 |
		uint64(vr.d[idx+4])<<32 | uint64(vr.d[idx+5])<<40 | uint64(vr.d[idx+6])<<48 | uint64(vr.d[idx+7])<<56, nil
}
