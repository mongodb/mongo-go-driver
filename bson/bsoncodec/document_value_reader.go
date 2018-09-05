package bsoncodec

import (
	"errors"
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

type dvrState struct {
	mode mode
	v    *bson.Value
	d    *bson.Document
	a    *bson.Array
	idx  uint
}

// documentValueReader is for reading *Document.
type documentValueReader struct {
	stack []dvrState
	frame int64
}

// NewDocumentValueReader creates a new ValueReader from the given *Document.
func NewDocumentValueReader(d *bson.Document) (ValueReader, error) {
	if d == nil {
		return nil, bson.ErrNilDocument
	}
	return newDocumentValueReader(d), nil
}

func newDocumentValueReader(d *bson.Document) *documentValueReader {
	stack := make([]dvrState, 1, 5)
	stack[0] = dvrState{
		mode: mTopLevel,
		d:    d,
	}
	return &documentValueReader{
		stack: stack,
	}
}

func (dvr *documentValueReader) invalidTransitionErr(destination mode) error {
	te := TransitionError{
		current:     dvr.stack[dvr.frame].mode,
		destination: destination,
	}
	if dvr.frame != 0 {
		te.parent = dvr.stack[dvr.frame-1].mode
	}
	return te
}

func (dvr *documentValueReader) typeError(t bson.Type) error {
	val := dvr.stack[dvr.frame].v
	if val == nil {
		return fmt.Errorf("positioned on %s, but attempted to read %s", bson.Type(0), t)
	}
	return fmt.Errorf("positioned on %s, but attempted to read %s", val.Type(), t)
}

func (dvr *documentValueReader) growStack() {
	if dvr.frame+1 >= int64(len(dvr.stack)) { // We need to grow the stack
		length := len(dvr.stack)
		if length+1 >= cap(dvr.stack) {
			// double it
			buf := make([]dvrState, 2*cap(dvr.stack)+1)
			copy(buf, dvr.stack)
			dvr.stack = buf
		}
		dvr.stack = dvr.stack[:length+1]
	}
}

func (dvr *documentValueReader) push(m mode) {
	dvr.growStack()
	dvr.frame++

	// Clean the stack
	dvr.stack[dvr.frame].mode = m
	dvr.stack[dvr.frame].v = nil
	dvr.stack[dvr.frame].d = nil
	dvr.stack[dvr.frame].a = nil
	dvr.stack[dvr.frame].idx = 0

}

func (dvr *documentValueReader) pop() {
	switch dvr.stack[dvr.frame].mode {
	case mElement, mValue:
		dvr.frame--
	case mDocument, mArray, mCodeWithScope:
		dvr.frame -= 2 // we pop twice to jump over the vrElement: vrDocument -> vrElement -> vrDocument/TopLevel/etc...
	}
}

func (dvr *documentValueReader) ensureElementValue(t bson.Type, destination mode) error {
	switch dvr.stack[dvr.frame].mode {
	case mElement, mValue:
		if dvr.stack[dvr.frame].v == nil || dvr.stack[dvr.frame].v.Type() != t {
			return dvr.typeError(t)
		}
	default:
		return dvr.invalidTransitionErr(destination)
	}

	return nil
}

func (dvr *documentValueReader) Type() bson.Type {
	switch dvr.stack[dvr.frame].mode {
	case mElement, mValue:
	default:
		return bson.Type(0)
	}

	if dvr.stack[dvr.frame].v == nil {
		return bson.Type(0)
	}

	return dvr.stack[dvr.frame].v.Type()
}

func (dvr *documentValueReader) Skip() error {
	if dvr.stack[dvr.frame].mode != mElement && dvr.stack[dvr.frame].mode != mValue {
		return dvr.invalidTransitionErr(0)
	}

	defer dvr.pop()

	return nil
}

func (dvr *documentValueReader) ReadArray() (ArrayReader, error) {
	if err := dvr.ensureElementValue(bson.TypeArray, mArray); err != nil {
		return nil, err
	}
	val := dvr.stack[dvr.frame].v
	dvr.push(mArray)
	dvr.stack[dvr.frame].a = val.MutableArray()

	return dvr, nil
}

func (dvr *documentValueReader) ReadBinary() (b []byte, btype byte, err error) {
	if err := dvr.ensureElementValue(bson.TypeBinary, 0); err != nil {
		return nil, 0, err
	}
	defer dvr.pop()

	btype, b = dvr.stack[dvr.frame].v.Binary()
	return b, btype, nil
}

func (dvr *documentValueReader) ReadBoolean() (bool, error) {
	if err := dvr.ensureElementValue(bson.TypeBoolean, 0); err != nil {
		return false, err
	}
	defer dvr.pop()

	return dvr.stack[dvr.frame].v.Boolean(), nil
}

func (dvr *documentValueReader) ReadDocument() (DocumentReader, error) {
	switch dvr.stack[dvr.frame].mode {
	case mTopLevel:
		return dvr, nil
	case mElement, mValue:
		if dvr.stack[dvr.frame].v == nil || dvr.stack[dvr.frame].v.Type() != bson.TypeEmbeddedDocument {
			return nil, dvr.typeError(bson.TypeEmbeddedDocument)
		}
	default:
		return nil, dvr.invalidTransitionErr(mDocument)
	}

	val := dvr.stack[dvr.frame].v
	dvr.push(mDocument)
	dvr.stack[dvr.frame].d = val.MutableDocument()

	return dvr, nil
}

func (dvr *documentValueReader) ReadCodeWithScope() (code string, dr DocumentReader, err error) {
	if err := dvr.ensureElementValue(bson.TypeCodeWithScope, 0); err != nil {
		return "", nil, err
	}

	code, doc := dvr.stack[dvr.frame].v.MutableJavaScriptWithScope()
	dvr.push(mCodeWithScope)
	dvr.stack[dvr.frame].d = doc
	return code, dvr, nil
}

func (dvr *documentValueReader) ReadDBPointer() (ns string, oid objectid.ObjectID, err error) {
	if err := dvr.ensureElementValue(bson.TypeDBPointer, 0); err != nil {
		return "", objectid.ObjectID{}, err
	}
	defer dvr.pop()

	ns, oid = dvr.stack[dvr.frame].v.DBPointer()
	return ns, oid, nil
}

func (dvr *documentValueReader) ReadDateTime() (int64, error) {
	if err := dvr.ensureElementValue(bson.TypeDateTime, 0); err != nil {
		return 0, err
	}
	defer dvr.pop()

	return dvr.stack[dvr.frame].v.DateTime(), nil
}

func (dvr *documentValueReader) ReadDecimal128() (decimal.Decimal128, error) {
	if err := dvr.ensureElementValue(bson.TypeDecimal128, 0); err != nil {
		return decimal.Decimal128{}, err
	}
	defer dvr.pop()

	return dvr.stack[dvr.frame].v.Decimal128(), nil
}

func (dvr *documentValueReader) ReadDouble() (float64, error) {
	if err := dvr.ensureElementValue(bson.TypeDouble, 0); err != nil {
		return 0, err
	}
	defer dvr.pop()

	return dvr.stack[dvr.frame].v.Double(), nil
}

func (dvr *documentValueReader) ReadInt32() (int32, error) {
	if err := dvr.ensureElementValue(bson.TypeInt32, 0); err != nil {
		return 0, err
	}
	defer dvr.pop()

	return dvr.stack[dvr.frame].v.Int32(), nil
}

func (dvr *documentValueReader) ReadInt64() (int64, error) {
	if err := dvr.ensureElementValue(bson.TypeInt64, 0); err != nil {
		return 0, err
	}
	defer dvr.pop()

	return dvr.stack[dvr.frame].v.Int64(), nil
}

func (dvr *documentValueReader) ReadJavascript() (code string, err error) {
	if err := dvr.ensureElementValue(bson.TypeJavaScript, 0); err != nil {
		return "", err
	}
	defer dvr.pop()

	return dvr.stack[dvr.frame].v.JavaScript(), nil
}

func (dvr *documentValueReader) ReadMaxKey() error {
	if err := dvr.ensureElementValue(bson.TypeMaxKey, 0); err != nil {
		return err
	}
	defer dvr.pop()

	return nil
}

func (dvr *documentValueReader) ReadMinKey() error {
	if err := dvr.ensureElementValue(bson.TypeMinKey, 0); err != nil {
		return err
	}
	defer dvr.pop()

	return nil
}

func (dvr *documentValueReader) ReadNull() error {
	if err := dvr.ensureElementValue(bson.TypeNull, 0); err != nil {
		return err
	}
	defer dvr.pop()

	return nil
}

func (dvr *documentValueReader) ReadObjectID() (objectid.ObjectID, error) {
	if err := dvr.ensureElementValue(bson.TypeObjectID, 0); err != nil {
		return objectid.ObjectID{}, err
	}
	defer dvr.pop()

	return dvr.stack[dvr.frame].v.ObjectID(), nil
}

func (dvr *documentValueReader) ReadRegex() (pattern string, options string, err error) {
	if err := dvr.ensureElementValue(bson.TypeRegex, 0); err != nil {
		return "", "", err
	}
	defer dvr.pop()

	pattern, options = dvr.stack[dvr.frame].v.Regex()
	return pattern, options, nil
}

func (dvr *documentValueReader) ReadString() (string, error) {
	if err := dvr.ensureElementValue(bson.TypeString, 0); err != nil {
		return "", err
	}
	defer dvr.pop()

	return dvr.stack[dvr.frame].v.StringValue(), nil
}

func (dvr *documentValueReader) ReadSymbol() (symbol string, err error) {
	if err := dvr.ensureElementValue(bson.TypeSymbol, 0); err != nil {
		return "", err
	}
	defer dvr.pop()

	return dvr.stack[dvr.frame].v.Symbol(), nil
}

func (dvr *documentValueReader) ReadTimestamp() (t uint32, i uint32, err error) {
	if err := dvr.ensureElementValue(bson.TypeTimestamp, 0); err != nil {
		return 0, 0, err
	}
	defer dvr.pop()

	t, i = dvr.stack[dvr.frame].v.Timestamp()
	return t, i, nil
}

func (dvr *documentValueReader) ReadUndefined() error {
	if err := dvr.ensureElementValue(bson.TypeUndefined, 0); err != nil {
		return err
	}
	defer dvr.pop()

	return nil
}

func (dvr *documentValueReader) ReadElement() (string, ValueReader, error) {
	switch dvr.stack[dvr.frame].mode {
	case mTopLevel, mDocument, mCodeWithScope:
		if dvr.stack[dvr.frame].d == nil {
			return "", nil, errors.New("Invalid ValueReader, cannot have a nil *Document")
		}
	default:
		return "", nil, dvr.invalidTransitionErr(mElement)
	}

	elem, exists := dvr.stack[dvr.frame].d.ElementAtOK(dvr.stack[dvr.frame].idx)
	if !exists {
		dvr.pop()
		return "", nil, ErrEOD
	}

	dvr.stack[dvr.frame].idx++
	dvr.push(mElement)
	dvr.stack[dvr.frame].v = elem.Value()

	return elem.Key(), dvr, nil
}

func (dvr *documentValueReader) ReadValue() (ValueReader, error) {
	switch dvr.stack[dvr.frame].mode {
	case mArray:
		if dvr.stack[dvr.frame].a == nil {
			return nil, errors.New("Invalid ValueReader, cannot have a nil *Array")
		}
	default:
		return nil, dvr.invalidTransitionErr(mValue)
	}

	val, err := dvr.stack[dvr.frame].a.Lookup(dvr.stack[dvr.frame].idx)
	if err != nil { // The only error we'll get here is an ErrOutOfBounds
		dvr.pop()
		return nil, ErrEOA
	}

	dvr.stack[dvr.frame].idx++
	dvr.push(mValue)
	dvr.stack[dvr.frame].v = val

	return dvr, nil
}
