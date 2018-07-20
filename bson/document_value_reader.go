package bson

import (
	"errors"
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

type dvrState struct {
	mode vrMode
	v    *Value
	d    *Document
	a    *Array
	idx  uint
}

// documentValueReader is for reading *Document.
type documentValueReader struct {
	stack []dvrState
	frame int64
}

func newDocumentValueReader(d *Document) (*documentValueReader, error) {
	if d == nil {
		return nil, ErrNilDocument
	}
	stack := make([]dvrState, 1, 5)
	stack[0] = dvrState{
		mode: vrTopLevel,
		d:    d,
	}
	return &documentValueReader{
		stack: stack,
	}, nil
}

func (vr *documentValueReader) invalidTransitionErr(destination vrMode) error {
	te := transitionError{
		current:     vr.stack[vr.frame].mode,
		destination: destination,
	}
	if vr.frame != 0 {
		te.parent = vr.stack[vr.frame-1].mode
	}
	return te
}

func (vr *documentValueReader) typeError(t Type) error {
	val := vr.stack[vr.frame].v
	if val == nil {
		return fmt.Errorf("positioned on %s, but attempted to read %s", Type(0), t)
	}
	return fmt.Errorf("positioned on %s, but attempted to read %s", val.Type(), t)
}

func (dvr *documentValueReader) growStack() {
	length := len(dvr.stack)
	if dvr.frame+1 >= int64(cap(dvr.stack)) {
		// double it
		buf := make([]dvrState, 2*cap(dvr.stack)+1)
		copy(buf, dvr.stack)
		dvr.stack = buf
	}
	dvr.stack = dvr.stack[:length+1]
}

func (dvr *documentValueReader) push(mode vrMode) {
	dvr.growStack()
	dvr.frame++
	// TODO: We might want to clean the stack frame before we move forward to
	// make debugging easier.

	dvr.stack[dvr.frame].mode = mode
}

func (dvr *documentValueReader) pop() {
	switch dvr.stack[dvr.frame].mode {
	case vrElement, vrValue:
		dvr.frame--
	case vrDocument, vrArray, vrCodeWithScope:
		dvr.frame -= 2 // we pop twice to jump over the vrElement: vrDocument -> vrElement -> vrDocument/TopLevel/etc...
	}
}

func (dvr *documentValueReader) ensureElementValue(t Type, destination vrMode) error {
	switch dvr.stack[dvr.frame].mode {
	case vrElement, vrValue:
		if dvr.stack[dvr.frame].v == nil || dvr.stack[dvr.frame].v.Type() != t {
			return dvr.typeError(t)
		}
	default:
		return dvr.invalidTransitionErr(destination)
	}

	return nil
}

func (dvr *documentValueReader) Type() Type {
	switch dvr.stack[dvr.frame].mode {
	case vrElement, vrValue:
	default:
		return Type(0)
	}

	if dvr.stack[dvr.frame].v == nil {
		return Type(0)
	}

	return dvr.stack[dvr.frame].v.Type()
}

func (dvr *documentValueReader) Skip() error {
	if dvr.stack[dvr.frame].mode != vrElement && dvr.stack[dvr.frame].mode != vrValue {
		return dvr.invalidTransitionErr(0)
	}

	defer dvr.pop()

	return nil
}

func (dvr *documentValueReader) ReadArray() (ArrayReader, error) {
	if err := dvr.ensureElementValue(TypeArray, vrArray); err != nil {
		return nil, err
	}
	val := dvr.stack[dvr.frame].v
	dvr.push(vrArray)
	dvr.stack[dvr.frame].a = val.MutableArray()

	return dvr, nil
}

func (dvr *documentValueReader) ReadBinary() (b []byte, btype byte, err error) {
	if err := dvr.ensureElementValue(TypeBinary, 0); err != nil {
		return nil, 0, err
	}
	defer dvr.pop()

	btype, b = dvr.stack[dvr.frame].v.Binary()
	return b, btype, nil
}

func (dvr *documentValueReader) ReadBoolean() (bool, error) {
	if err := dvr.ensureElementValue(TypeBoolean, 0); err != nil {
		return false, err
	}
	defer dvr.pop()

	return dvr.stack[dvr.frame].v.Boolean(), nil
}

func (dvr *documentValueReader) ReadDocument() (DocumentReader, error) {
	switch dvr.stack[dvr.frame].mode {
	case vrTopLevel:
		return dvr, nil
	case vrElement, vrValue:
		if dvr.stack[dvr.frame].v == nil || dvr.stack[dvr.frame].v.Type() != TypeEmbeddedDocument {
			return nil, dvr.typeError(TypeEmbeddedDocument)
		}
	default:
		return nil, dvr.invalidTransitionErr(vrDocument)
	}

	val := dvr.stack[dvr.frame].v
	dvr.push(vrDocument)
	dvr.stack[dvr.frame].d = val.MutableDocument()

	return dvr, nil
}

func (dvr *documentValueReader) ReadCodeWithScope() (code string, dr DocumentReader, err error) {
	if err := dvr.ensureElementValue(TypeCodeWithScope, 0); err != nil {
		return "", nil, err
	}

	code, doc := dvr.stack[dvr.frame].v.MutableJavaScriptWithScope()
	dvr.push(vrCodeWithScope)
	dvr.stack[dvr.frame].d = doc
	return code, dvr, nil
}

func (dvr *documentValueReader) ReadDBPointer() (ns string, oid objectid.ObjectID, err error) {
	if err := dvr.ensureElementValue(TypeDBPointer, 0); err != nil {
		return "", objectid.ObjectID{}, err
	}
	defer dvr.pop()

	ns, oid = dvr.stack[dvr.frame].v.DBPointer()
	return ns, oid, nil
}

func (dvr *documentValueReader) ReadDateTime() (int64, error) {
	if err := dvr.ensureElementValue(TypeDateTime, 0); err != nil {
		return 0, err
	}
	defer dvr.pop()

	return dvr.stack[dvr.frame].v.DateTime(), nil
}

func (dvr *documentValueReader) ReadDecimal128() (decimal.Decimal128, error) {
	if err := dvr.ensureElementValue(TypeDecimal128, 0); err != nil {
		return decimal.Decimal128{}, err
	}
	defer dvr.pop()

	return dvr.stack[dvr.frame].v.Decimal128(), nil
}

func (dvr *documentValueReader) ReadDouble() (float64, error) {
	if err := dvr.ensureElementValue(TypeDouble, 0); err != nil {
		return 0, err
	}
	defer dvr.pop()

	return dvr.stack[dvr.frame].v.Double(), nil
}

func (dvr *documentValueReader) ReadInt32() (int32, error) {
	if err := dvr.ensureElementValue(TypeInt32, 0); err != nil {
		return 0, err
	}
	defer dvr.pop()

	return dvr.stack[dvr.frame].v.Int32(), nil
}

func (dvr *documentValueReader) ReadInt64() (int64, error) {
	if err := dvr.ensureElementValue(TypeInt64, 0); err != nil {
		return 0, err
	}
	defer dvr.pop()

	return dvr.stack[dvr.frame].v.Int64(), nil
}

func (dvr *documentValueReader) ReadJavascript() (code string, err error) {
	if err := dvr.ensureElementValue(TypeJavaScript, 0); err != nil {
		return "", err
	}
	defer dvr.pop()

	return dvr.stack[dvr.frame].v.JavaScript(), nil
}

func (dvr *documentValueReader) ReadMaxKey() error {
	if err := dvr.ensureElementValue(TypeMaxKey, 0); err != nil {
		return err
	}
	defer dvr.pop()

	return nil
}

func (dvr *documentValueReader) ReadMinKey() error {
	if err := dvr.ensureElementValue(TypeMinKey, 0); err != nil {
		return err
	}
	defer dvr.pop()

	return nil
}

func (dvr *documentValueReader) ReadNull() error {
	if err := dvr.ensureElementValue(TypeNull, 0); err != nil {
		return err
	}
	defer dvr.pop()

	return nil
}

func (dvr *documentValueReader) ReadObjectID() (objectid.ObjectID, error) {
	if err := dvr.ensureElementValue(TypeObjectID, 0); err != nil {
		return objectid.ObjectID{}, err
	}
	defer dvr.pop()

	return dvr.stack[dvr.frame].v.ObjectID(), nil
}

func (dvr *documentValueReader) ReadRegex() (pattern string, options string, err error) {
	if err := dvr.ensureElementValue(TypeRegex, 0); err != nil {
		return "", "", err
	}
	defer dvr.pop()

	pattern, options = dvr.stack[dvr.frame].v.Regex()
	return pattern, options, nil
}

func (dvr *documentValueReader) ReadString() (string, error) {
	if err := dvr.ensureElementValue(TypeString, 0); err != nil {
		return "", err
	}
	defer dvr.pop()

	return dvr.stack[dvr.frame].v.StringValue(), nil
}

func (dvr *documentValueReader) ReadSymbol() (symbol string, err error) {
	if err := dvr.ensureElementValue(TypeSymbol, 0); err != nil {
		return "", err
	}
	defer dvr.pop()

	return dvr.stack[dvr.frame].v.Symbol(), nil
}

func (dvr *documentValueReader) ReadTimestamp() (t uint32, i uint32, err error) {
	if err := dvr.ensureElementValue(TypeTimestamp, 0); err != nil {
		return 0, 0, err
	}
	defer dvr.pop()

	t, i = dvr.stack[dvr.frame].v.Timestamp()
	return t, i, nil
}

func (dvr *documentValueReader) ReadUndefined() error {
	if err := dvr.ensureElementValue(TypeUndefined, 0); err != nil {
		return err
	}
	defer dvr.pop()

	return nil
}

func (dvr *documentValueReader) ReadElement() (string, ValueReader, error) {
	switch dvr.stack[dvr.frame].mode {
	case vrTopLevel, vrDocument:
		if dvr.stack[dvr.frame].d == nil {
			return "", nil, errors.New("Invalid ValueReader, cannot have a nil *Document")
		}
	default:
		return "", nil, dvr.invalidTransitionErr(vrElement)
	}

	elem, exists := dvr.stack[dvr.frame].d.ElementAtOK(dvr.stack[dvr.frame].idx)
	if !exists {
		dvr.pop()
		return "", nil, EOD
	}

	dvr.stack[dvr.frame].idx++
	dvr.push(vrElement)
	dvr.stack[dvr.frame].v = elem.Value()

	return elem.Key(), dvr, nil
}

func (dvr *documentValueReader) ReadValue() (ValueReader, error) {
	switch dvr.stack[dvr.frame].mode {
	case vrArray:
		if dvr.stack[dvr.frame].a == nil {
			return nil, errors.New("Invalid ValueReader, cannot have a nil *Array")
		}
	default:
		return nil, dvr.invalidTransitionErr(vrValue)
	}

	val, err := dvr.stack[dvr.frame].a.Lookup(dvr.stack[dvr.frame].idx)
	if err != nil { // The only error we'll get here is an ErrOutOfBounds
		dvr.pop()
		return nil, EOA
	}

	dvr.stack[dvr.frame].idx++
	dvr.push(vrValue)
	dvr.stack[dvr.frame].v = val

	return dvr, nil
}
