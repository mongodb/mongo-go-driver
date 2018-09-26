package bsoncodec

import (
	"fmt"
	"io"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

type ejvrState struct {
	mode  mode
	vType bson.Type
	depth int
}

// extJSONValueReader is for reading extended JSON.
type extJSONValueReader struct {
	p *extJSONParser

	stack []ejvrState
	frame int
}

// NewExtJSONValueReader creates a new ValueReader from a given io.Reader
// It will interpret the JSON of r as canonical or relaxed according to the
// given canonical flag
func NewExtJSONValueReader(r io.Reader, canonical bool) ValueReader {
	return newExtJSONValueReader(r, canonical)
}

func newExtJSONValueReader(r io.Reader, canonical bool) *extJSONValueReader {
	p := newExtJSONParser(r, canonical)
	typ, err := p.peekType()

	if err != nil {
		// TODO: invalid JSON--return error message?
		return nil
	}

	var m mode
	switch typ {
	case bson.TypeEmbeddedDocument:
		m = mTopLevel
	case bson.TypeArray:
		m = mArray
	default:
		m = mValue
	}

	stack := make([]ejvrState, 1, 5)
	stack[0] = ejvrState{
		mode:  m,
		vType: typ,
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

func (ejvr *extJSONValueReader) pushCodeWithScope() {
	ejvr.advanceFrame()

	ejvr.stack[ejvr.frame].mode = mCodeWithScope
}

func (ejvr *extJSONValueReader) pushArray() {
	ejvr.advanceFrame()

	ejvr.stack[ejvr.frame].mode = mArray
}

func (ejvr *extJSONValueReader) push(m mode, t bson.Type) {
	ejvr.advanceFrame()

	ejvr.stack[ejvr.frame].mode = m
	ejvr.stack[ejvr.frame].vType = t
}

func (ejvr *extJSONValueReader) pop() {
	switch ejvr.stack[ejvr.frame].mode {
	case mElement, mValue:
		ejvr.frame--
	case mDocument, mArray, mCodeWithScope:
		ejvr.frame -= 2 // we pop twice to jump over the vrElement: vrDocument -> vrElement -> vrDocument/TopLevel/etc...
	}
}

func (ejvr *extJSONValueReader) skipDocument() error {
	// read entire document until ErrEOD (using readKey and readValue)
	_, typ, err := ejvr.p.readKey()
	for err == nil {
		_, err = ejvr.p.readValue(typ)
		if err != nil {
			break
		}

		_, typ, err = ejvr.p.readKey()
	}

	return err
}

func (ejvr *extJSONValueReader) skipArray() error {
	// read entire array until ErrEOA (using peekType)
	_, err := ejvr.p.peekType()
	for err == nil {
		_, err = ejvr.p.peekType()
	}

	return err
}

func (ejvr *extJSONValueReader) invalidTransitionErr(destination mode) error {
	te := TransitionError{
		current:     ejvr.stack[ejvr.frame].mode,
		destination: destination,
	}
	if ejvr.frame != 0 {
		te.parent = ejvr.stack[ejvr.frame-1].mode
	}
	return te
}

func (ejvr *extJSONValueReader) typeError(t bson.Type) error {
	return fmt.Errorf("positioned on %s, but attempted to read %s", ejvr.stack[ejvr.frame].vType, t)
}

func (ejvr *extJSONValueReader) ensureElementValue(t bson.Type, destination mode) error {
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

func (ejvr *extJSONValueReader) Type() bson.Type {
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
	case bson.TypeArray:
		// read entire array until ErrEOA
		err := ejvr.skipArray()
		if err != ErrEOA {
			return err
		}
	case bson.TypeEmbeddedDocument:
		// read entire doc until ErrEOD
		err := ejvr.skipDocument()
		if err != ErrEOD {
			return err
		}
	case bson.TypeCodeWithScope:
		// read the code portion and set up parser in document mode
		_, err := ejvr.p.readValue(t)
		if err != nil {
			return err
		}

		// read until ErrEOD
		err = ejvr.skipDocument()
		if err != ErrEOD {
			return err
		}
	default:
		_, err := ejvr.p.readValue(t)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ejvr *extJSONValueReader) ReadArray() (ArrayReader, error) {
	switch ejvr.stack[ejvr.frame].mode {
	case mTopLevel: // allow reading array from top level
	case mArray:
		return ejvr, nil
	default:
		if err := ejvr.ensureElementValue(bson.TypeArray, mArray); err != nil {
			return nil, err
		}
	}

	ejvr.pushArray()

	return ejvr, nil
}

func (ejvr *extJSONValueReader) ReadBinary() (b []byte, btype byte, err error) {
	if err := ejvr.ensureElementValue(bson.TypeBinary, 0); err != nil {
		return nil, 0, err
	}

	v, err := ejvr.p.readValue(bson.TypeBinary)
	if err != nil {
		return nil, 0, err
	}

	b, btype, err = v.parseBinary()

	ejvr.pop()
	return b, btype, err
}

func (ejvr *extJSONValueReader) ReadBoolean() (bool, error) {
	if err := ejvr.ensureElementValue(bson.TypeBoolean, 0); err != nil {
		return false, err
	}

	v, err := ejvr.p.readValue(bson.TypeBoolean)
	if err != nil {
		return false, err
	}

	if v.t != bson.TypeBoolean {
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
		if ejvr.stack[ejvr.frame].vType != bson.TypeEmbeddedDocument {
			return nil, ejvr.typeError(bson.TypeEmbeddedDocument)
		}

		ejvr.pushDocument()
		return ejvr, nil
	default:
		return nil, ejvr.invalidTransitionErr(mDocument)
	}
}

func (ejvr *extJSONValueReader) ReadCodeWithScope() (code string, dr DocumentReader, err error) {
	if err = ejvr.ensureElementValue(bson.TypeCodeWithScope, 0); err != nil {
		return "", nil, err
	}

	v, err := ejvr.p.readValue(bson.TypeCodeWithScope)
	if err != nil {
		return "", nil, err
	}

	code, err = v.parseJavascript()

	ejvr.pushCodeWithScope()
	return code, ejvr, err
}

func (ejvr *extJSONValueReader) ReadDBPointer() (ns string, oid objectid.ObjectID, err error) {
	if err = ejvr.ensureElementValue(bson.TypeDBPointer, 0); err != nil {
		return "", objectid.NilObjectID, err
	}

	v, err := ejvr.p.readValue(bson.TypeDBPointer)
	if err != nil {
		return "", objectid.NilObjectID, err
	}

	ns, oid, err = v.parseDBPointer()

	ejvr.pop()
	return ns, oid, err
}

func (ejvr *extJSONValueReader) ReadDateTime() (int64, error) {
	if err := ejvr.ensureElementValue(bson.TypeDateTime, 0); err != nil {
		return 0, err
	}

	v, err := ejvr.p.readValue(bson.TypeDateTime)
	if err != nil {
		return 0, err
	}

	d, err := v.parseDateTime()

	ejvr.pop()
	return d, err
}

func (ejvr *extJSONValueReader) ReadDecimal128() (decimal.Decimal128, error) {
	if err := ejvr.ensureElementValue(bson.TypeDecimal128, 0); err != nil {
		return decimal.Decimal128{}, err
	}

	v, err := ejvr.p.readValue(bson.TypeDecimal128)
	if err != nil {
		return decimal.Decimal128{}, err
	}

	d, err := v.parseDecimal128()

	ejvr.pop()
	return d, err
}

func (ejvr *extJSONValueReader) ReadDouble() (float64, error) {
	if err := ejvr.ensureElementValue(bson.TypeDouble, 0); err != nil {
		return 0, err
	}

	v, err := ejvr.p.readValue(bson.TypeDouble)
	if err != nil {
		return 0, err
	}

	d, err := v.parseDouble()

	ejvr.pop()
	return d, err
}

func (ejvr *extJSONValueReader) ReadInt32() (int32, error) {
	if err := ejvr.ensureElementValue(bson.TypeInt32, 0); err != nil {
		return 0, err
	}

	v, err := ejvr.p.readValue(bson.TypeInt32)
	if err != nil {
		return 0, err
	}

	i, err := v.parseInt32()

	ejvr.pop()
	return i, err
}

func (ejvr *extJSONValueReader) ReadInt64() (int64, error) {
	if err := ejvr.ensureElementValue(bson.TypeInt64, 0); err != nil {
		return 0, err
	}

	v, err := ejvr.p.readValue(bson.TypeInt64)
	if err != nil {
		return 0, err
	}

	i, err := v.parseInt64()

	ejvr.pop()
	return i, err
}

func (ejvr *extJSONValueReader) ReadJavascript() (code string, err error) {
	if err = ejvr.ensureElementValue(bson.TypeJavaScript, 0); err != nil {
		return "", err
	}

	v, err := ejvr.p.readValue(bson.TypeJavaScript)
	if err != nil {
		return "", err
	}

	code, err = v.parseJavascript()

	ejvr.pop()
	return code, err
}

func (ejvr *extJSONValueReader) ReadMaxKey() error {
	if err := ejvr.ensureElementValue(bson.TypeMaxKey, 0); err != nil {
		return err
	}

	v, err := ejvr.p.readValue(bson.TypeMaxKey)
	if err != nil {
		return err
	}

	err = v.parseMinMaxKey("max")

	ejvr.pop()
	return err
}

func (ejvr *extJSONValueReader) ReadMinKey() error {
	if err := ejvr.ensureElementValue(bson.TypeMinKey, 0); err != nil {
		return err
	}

	v, err := ejvr.p.readValue(bson.TypeMinKey)
	if err != nil {
		return err
	}

	err = v.parseMinMaxKey("min")

	ejvr.pop()
	return err
}

func (ejvr *extJSONValueReader) ReadNull() error {
	if err := ejvr.ensureElementValue(bson.TypeNull, 0); err != nil {
		return err
	}

	v, err := ejvr.p.readValue(bson.TypeNull)
	if err != nil {
		return err
	}

	if v.t != bson.TypeNull {
		return fmt.Errorf("expected type null but got type %s", v.t)
	}

	ejvr.pop()
	return nil
}

func (ejvr *extJSONValueReader) ReadObjectID() (objectid.ObjectID, error) {
	if err := ejvr.ensureElementValue(bson.TypeObjectID, 0); err != nil {
		return objectid.ObjectID{}, err
	}

	v, err := ejvr.p.readValue(bson.TypeObjectID)
	if err != nil {
		return objectid.ObjectID{}, err
	}

	oid, err := v.parseObjectID()

	ejvr.pop()
	return oid, err
}

func (ejvr *extJSONValueReader) ReadRegex() (pattern string, options string, err error) {
	if err = ejvr.ensureElementValue(bson.TypeRegex, 0); err != nil {
		return "", "", err
	}

	v, err := ejvr.p.readValue(bson.TypeRegex)
	if err != nil {
		return "", "", err
	}

	pattern, options, err = v.parseRegex()

	ejvr.pop()
	return pattern, options, err
}

func (ejvr *extJSONValueReader) ReadString() (string, error) {
	if err := ejvr.ensureElementValue(bson.TypeString, 0); err != nil {
		return "", err
	}

	v, err := ejvr.p.readValue(bson.TypeString)
	if err != nil {
		return "", err
	}

	if v.t != bson.TypeString {
		return "", fmt.Errorf("expected type string but got type %s", v.t)
	}

	ejvr.pop()
	return v.v.(string), nil
}

func (ejvr *extJSONValueReader) ReadSymbol() (symbol string, err error) {
	if err = ejvr.ensureElementValue(bson.TypeSymbol, 0); err != nil {
		return "", err
	}

	v, err := ejvr.p.readValue(bson.TypeSymbol)
	if err != nil {
		return "", err
	}

	symbol, err = v.parseSymbol()

	ejvr.pop()
	return symbol, err
}

func (ejvr *extJSONValueReader) ReadTimestamp() (t uint32, i uint32, err error) {
	if err = ejvr.ensureElementValue(bson.TypeTimestamp, 0); err != nil {
		return 0, 0, err
	}

	v, err := ejvr.p.readValue(bson.TypeTimestamp)
	if err != nil {
		return 0, 0, err
	}

	t, i, err = v.parseTimestamp()

	ejvr.pop()
	return t, i, err
}

func (ejvr *extJSONValueReader) ReadUndefined() error {
	if err := ejvr.ensureElementValue(bson.TypeUndefined, 0); err != nil {
		return err
	}

	v, err := ejvr.p.readValue(bson.TypeUndefined)
	if err != nil {
		return err
	}

	err = v.parseUndefined()

	ejvr.pop()
	return err
}

func (ejvr *extJSONValueReader) ReadElement() (string, ValueReader, error) {
	switch ejvr.stack[ejvr.frame].mode {
	case mTopLevel, mDocument, mCodeWithScope:
	default:
		return "", nil, ejvr.invalidTransitionErr(mElement)
	}

	name, t, err := ejvr.p.readKey()

	if err != nil {
		if err == ErrEOD {
			if ejvr.stack[ejvr.frame].mode == mCodeWithScope {
				_, err := ejvr.p.peekType()
				if err != nil {
					return "", nil, err
				}
			}

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
