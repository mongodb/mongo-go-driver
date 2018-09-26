package bson

import (
	"fmt"
	"io"

	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

type ejvrState struct {
	mode  mode
	vType Type
	depth int
}

// extJSONValueReader is for reading extended JSON.
type extJSONValueReader struct {
	p *extJSONParser

	stack []ejvrState
	frame int
}

func NewExtJSONValueReader(r io.Reader) *extJSONValueReader {
	return newExtJSONValueReader(r)
}

func newExtJSONValueReader(r io.Reader) *extJSONValueReader {
	p := newExtJSONParser(r)
	typ, err := p.peekType()

	if err != nil {
		// TODO: invalid JSON--return error message?
		return nil
	}

	var m mode
	switch typ {
	case TypeEmbeddedDocument:
		m = mTopLevel
	case TypeArray:
		m = mArray
	default:
		m = mValue
	}

	stack := make([]ejvrState, 1, 5)
	stack[0] = ejvrState{
		mode:  m,
	}
	return &extJSONValueReader{
		p:     p,
		stack: stack,
	}
}

func (ejvr *extJSONValueReader) advanceFrame() {
	if ejvr.frame+1 >= len(ejvr.stack) { // We need to grow the stack
		length := len(ejvr.stack)
		if length+1 >= cap(ejvr.stack) {
			// double it
			buf := make([]ejvrState, 2*cap(ejvr.stack)+1)
			copy(buf, ejvr.stack)
			ejvr.stack = buf
		}
		ejvr.stack = ejvr.stack[:length+1]
	}
	ejvr.frame++

	// Clean the stack
	ejvr.stack[ejvr.frame].mode = 0
	ejvr.stack[ejvr.frame].vType = 0
	ejvr.stack[ejvr.frame].depth = 0
}

func (ejvr *extJSONValueReader) pushDocument() {
	ejvr.advanceFrame()

	ejvr.stack[ejvr.frame].mode = mDocument
	ejvr.stack[ejvr.frame].depth = ejvr.p.depth
}

func (ejvr *extJSONValueReader) pushArray() {
	ejvr.advanceFrame()

	ejvr.stack[ejvr.frame].mode = mArray
}

func (ejvr *extJSONValueReader) push(m mode, t Type) {
	ejvr.advanceFrame()

	ejvr.stack[ejvr.frame].mode = m
	ejvr.stack[ejvr.frame].vType = t
}

func (ejvr *extJSONValueReader) pop() {
	switch ejvr.stack[ejvr.frame].mode {
	case mElement, mValue:
		ejvr.frame--
	case mDocument, mArray:
		ejvr.frame -= 2 // we pop twice to jump over the vrElement: vrDocument -> vrElement -> vrDocument/TopLevel/etc...
	}
}
func (ejvr *extJSONValueReader) invalidTransitionErr(destination mode) error {
	te := transitionError{
		current:     ejvr.stack[ejvr.frame].mode,
		destination: destination,
	}
	if ejvr.frame != 0 {
		te.parent = ejvr.stack[ejvr.frame-1].mode
	}
	return te
}

func (ejvr *extJSONValueReader) typeError(t Type) error {
	return fmt.Errorf("positioned on %s, but attempted to read %s", ejvr.stack[ejvr.frame].vType, t)
}

func (ejvr *extJSONValueReader) ensureElementValue(t Type, destination mode) error {
	switch ejvr.stack[ejvr.frame].mode {
	case mElement, mValue:
		if ejvr.stack[ejvr.frame].vType != t {
			return ejvr.typeError(t)
		}
	default:
		return ejvr.invalidTransitionErr(destination)
	}

	return nil
}

func (ejvr *extJSONValueReader) Type() Type {
	return ejvr.stack[ejvr.frame].vType
}

func (ejvr *extJSONValueReader) Skip() error {
	switch ejvr.stack[ejvr.frame].mode {
	case mElement, mValue:
	default:
		return ejvr.invalidTransitionErr(0)
	}

	defer ejvr.pop()

	t := ejvr.stack[ejvr.frame].vType
	switch t {
	case TypeArray:
		// TODO
		// read entire array until ErrEOA (using peekType or ReadValue)
	case TypeEmbeddedDocument:
		// TODO
		// read entire doc until ErrEOD (using readKey, readValue or ReadElement)
	case TypeCodeWithScope:
		// TODO
		// read javascript (check)
		// read entire document until ErrEOD (using readKey, readValue or ReadElement (or recursion))
	default:
		_, err := ejvr.p.readValue(t)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ejvr *extJSONValueReader) ReadArray() (ArrayReader, error) {
	if err := ejvr.ensureElementValue(TypeArray, mArray); err != nil {
		return nil, err
	}

	ejvr.pushArray()

	return ejvr, nil
}

func (ejvr *extJSONValueReader) ReadBinary() (b []byte, btype byte, err error) {
	if err := ejvr.ensureElementValue(TypeBinary, 0); err != nil {
		return nil, 0, err
	}

	v, err := ejvr.p.readValue(TypeBinary)
	if err != nil {
		return nil, 0, err
	}

	b, btype, err = parseBinary(v)

	ejvr.pop()
	return b, btype, err
}

func (ejvr *extJSONValueReader) ReadBoolean() (bool, error) {
	if err := ejvr.ensureElementValue(TypeBoolean, 0); err != nil {
		return false, err
	}

	v, err := ejvr.p.readValue(TypeBoolean)
	if err != nil {
		return false, err
	}

	if v.t != TypeBoolean {
		return false, fmt.Errorf("expected type bool, but got type %s", v.t)
	}

	ejvr.pop()
	return v.v.(bool), nil
}

func (ejvr *extJSONValueReader) ReadDocument() (DocumentReader, error) {
	switch ejvr.stack[ejvr.frame].mode {
	case mTopLevel:
		return ejvr, nil
	case mElement, mValue:
		if ejvr.stack[ejvr.frame].vType != TypeEmbeddedDocument {
			return nil, ejvr.typeError(TypeEmbeddedDocument)
		}

		ejvr.pushDocument()
		return ejvr, nil
	default:
		return nil, ejvr.invalidTransitionErr(mDocument)
	}
}

func (ejvr *extJSONValueReader) ReadCodeWithScope() (code string, dr DocumentReader, err error) {
	if err = ejvr.ensureElementValue(TypeCodeWithScope, 0); err != nil {
		return
	}

	v, err := ejvr.p.readValue(TypeCodeWithScope)
	if err != nil {
		return
	}

	code, err = parseJavascript(v)

	ejvr.pushDocument()
	return code, ejvr, nil
}

func (ejvr *extJSONValueReader) ReadDBPointer() (ns string, oid objectid.ObjectID, err error) {
	if err = ejvr.ensureElementValue(TypeDBPointer, 0); err != nil {
		return
	}

	v, err := ejvr.p.readValue(TypeDBPointer)
	if err != nil {
		return
	}

	ns, oid, err = parseDBPointer(v)

	ejvr.pop()
	return
}

func (ejvr *extJSONValueReader) ReadDateTime() (int64, error) {
	if err := ejvr.ensureElementValue(TypeDateTime, 0); err != nil {
		return 0, err
	}

	v, err := ejvr.p.readValue(TypeDateTime)
	if err != nil {
		return 0, err
	}

	d, err := parseDateTime(v)

	ejvr.pop()
	return d, err
}

func (ejvr *extJSONValueReader) ReadDecimal128() (decimal.Decimal128, error) {
	if err := ejvr.ensureElementValue(TypeDecimal128, 0); err != nil {
		return decimal.Decimal128{}, err
	}

	v, err := ejvr.p.readValue(TypeDecimal128)
	if err != nil {
		return decimal.Decimal128{}, err
	}

	d, err := parseDecimal128(v)

	ejvr.pop()
	return d, err
}

func (ejvr *extJSONValueReader) ReadDouble() (float64, error) {
	if err := ejvr.ensureElementValue(TypeDouble, 0); err != nil {
		return 0, err
	}

	v, err := ejvr.p.readValue(TypeDouble)
	if err != nil {
		return 0, err
	}

	d, err := parseDouble(v)

	ejvr.pop()
	return d, err
}

func (ejvr *extJSONValueReader) ReadInt32() (int32, error) {
	if err := ejvr.ensureElementValue(TypeInt32, 0); err != nil {
		return 0, err
	}

	v, err := ejvr.p.readValue(TypeInt32)
	if err != nil {
		return 0, err
	}

	i, err := parseInt32(v)

	ejvr.pop()
	return i, err
}

func (ejvr *extJSONValueReader) ReadInt64() (int64, error) {
	if err := ejvr.ensureElementValue(TypeInt64, 0); err != nil {
		return 0, err
	}

	v, err := ejvr.p.readValue(TypeInt64)
	if err != nil {
		return 0, err
	}

	i, err := parseInt64(v)

	ejvr.pop()
	return i, err
}

func (ejvr *extJSONValueReader) ReadJavascript() (code string, err error) {
	if err = ejvr.ensureElementValue(TypeJavaScript, 0); err != nil {
		return
	}

	v, err := ejvr.p.readValue(TypeJavaScript)
	if err != nil {
		return
	}

	code, err = parseJavascript(v)

	ejvr.pop()
	return
}

func (ejvr *extJSONValueReader) ReadMaxKey() error {
	if err := ejvr.ensureElementValue(TypeMaxKey, 0); err != nil {
		return err
	}

	v, err := ejvr.p.readValue(TypeMaxKey)
	if err != nil {
		return err
	}

	err = parseMinMaxKey(v, "max")

	ejvr.pop()
	return err
}

func (ejvr *extJSONValueReader) ReadMinKey() error {
	if err := ejvr.ensureElementValue(TypeMinKey, 0); err != nil {
		return err
	}

	v, err := ejvr.p.readValue(TypeMinKey)
	if err != nil {
		return err
	}

	err = parseMinMaxKey(v, "min")

	ejvr.pop()
	return err
}

func (ejvr *extJSONValueReader) ReadNull() error {
	if err := ejvr.ensureElementValue(TypeNull, 0); err != nil {
		return err
	}

	v, err := ejvr.p.readValue(TypeNull)
	if err != nil {
		return err
	}

	if v.t != TypeNull {
		return fmt.Errorf("expected type null but got type %s", v.t)
	}

	ejvr.pop()
	return nil
}

func (ejvr *extJSONValueReader) ReadObjectID() (objectid.ObjectID, error) {
	if err := ejvr.ensureElementValue(TypeObjectID, 0); err != nil {
		return objectid.ObjectID{}, err
	}

	v, err := ejvr.p.readValue(TypeObjectID)
	if err != nil {
		return objectid.ObjectID{}, err
	}

	oid, err := parseObjectID(v)

	ejvr.pop()
	return oid, err
}

func (ejvr *extJSONValueReader) ReadRegex() (pattern string, options string, err error) {
	if err = ejvr.ensureElementValue(TypeRegex, 0); err != nil {
		return
	}

	v, err := ejvr.p.readValue(TypeRegex)
	if err != nil {
		return
	}

	pattern, options, err = parseRegex(v)

	ejvr.pop()
	return
}

func (ejvr *extJSONValueReader) ReadString() (string, error) {
	if err := ejvr.ensureElementValue(TypeString, 0); err != nil {
		return "", err
	}

	v, err := ejvr.p.readValue(TypeString)
	if err != nil {
		return "", err
	}

	if v.t != TypeString {
		return "", fmt.Errorf("expected type string but got type %s", v.t)
	}

	ejvr.pop()
	return v.v.(string), nil
}

func (ejvr *extJSONValueReader) ReadSymbol() (symbol string, err error) {
	if err = ejvr.ensureElementValue(TypeSymbol, 0); err != nil {
		return
	}

	v, err := ejvr.p.readValue(TypeSymbol)
	if err != nil {
		return
	}

	symbol, err = parseSymbol(v)

	ejvr.pop()
	return
}

func (ejvr *extJSONValueReader) ReadTimestamp() (t uint32, i uint32, err error) {
	if err = ejvr.ensureElementValue(TypeTimestamp, 0); err != nil {
		return
	}

	v, err := ejvr.p.readValue(TypeTimestamp)
	if err != nil {
		return
	}

	t, i, err = parseTimestamp(v)

	ejvr.pop()
	return
}

func (ejvr *extJSONValueReader) ReadUndefined() error {
	if err := ejvr.ensureElementValue(TypeUndefined, 0); err != nil {
		return err
	}

	v, err := ejvr.p.readValue(TypeUndefined)
	if err != nil {
		return err
	}

	err = parseUndefined(v)

	ejvr.pop()
	return err
}

func (ejvr *extJSONValueReader) ReadElement() (string, ValueReader, error) {
	switch ejvr.stack[ejvr.frame].mode {
	case mTopLevel, mDocument:
	default:
		return "", nil, ejvr.invalidTransitionErr(mElement)
	}

	name, t, err := ejvr.p.readKey()

	if err != nil {
		if err == ErrEOD {
			ejvr.pop()
		}

		return "", nil, err
	}

	ejvr.push(mElement, t)
	return name, ejvr, nil
}

func (ejvr *extJSONValueReader) ReadValue() (ValueReader, error) {
	switch ejvr.stack[ejvr.frame].mode {
	case mArray:
	default:
		return nil, ejvr.invalidTransitionErr(mValue)
	}

	t, err := ejvr.p.peekType()
	if err != nil {
		if err == ErrEOA {
			ejvr.pop()
		}

		return nil, err
	}

	ejvr.push(mValue, t)
	return ejvr, nil
}
