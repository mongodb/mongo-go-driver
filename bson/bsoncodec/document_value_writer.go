package bsoncodec

import (
	"errors"
	"fmt"
	"sync"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

// This pool is used to keep the allocations of documentValueWriters down.
// documentValueWriters retrieved from this pool should have reset called
// on them before use.
var dvwPool = sync.Pool{
	New: func() interface{} {
		return newDocumentValueWriter(nil)
	},
}

type documentValueWriter struct {
	stack []dvwState
	frame int64
}

type dvwState struct {
	mode mode
	d    *bson.Document
	key  string
}

// NewDocumentValueWriter creates a ValueWriter that adds elements to d.
func NewDocumentValueWriter(d *bson.Document) (ValueWriter, error) {
	if d == nil {
		return nil, errors.New("cannot create ValueWriter with nil *Document")
	}
	return newDocumentValueWriter(d), nil
}

func newDocumentValueWriter(d *bson.Document) *documentValueWriter {
	// We start the document value writer in the mode of writing a document
	dvw := new(documentValueWriter)
	stack := make([]dvwState, 1, 5)
	stack[0] = dvwState{mode: mTopLevel, d: d}
	dvw.stack = stack

	return dvw
}

func (dvw *documentValueWriter) reset(d *bson.Document) {
	dvw.stack = dvw.stack[:1]
	dvw.frame = 0
	dvw.stack[0] = dvwState{mode: mTopLevel, d: d}
}

func (dvw *documentValueWriter) advanceFrame() {
	if dvw.frame+1 >= int64(len(dvw.stack)) { // We need to grow the stack
		length := len(dvw.stack)
		if length+1 >= cap(dvw.stack) {
			// double it
			buf := make([]dvwState, 2*cap(dvw.stack)+1)
			copy(buf, dvw.stack)
			dvw.stack = buf
		}
		dvw.stack = dvw.stack[:length+1]
	}
	dvw.frame++
}

func (dvw *documentValueWriter) push(m mode) {
	dvw.advanceFrame()

	dvw.stack[dvw.frame].mode = m
	switch m {
	case mElement, mValue:
		dvw.stack[dvw.frame].d = dvw.stack[dvw.frame-1].d
	}
}

func (dvw *documentValueWriter) pop() {
	switch dvw.stack[dvw.frame].mode {
	case mElement, mValue:
		dvw.frame--
	case mDocument, mArray, mCodeWithScope:
		dvw.frame -= 2 // we pop twice to jump over the mElement: mDocument -> mElement -> mDocument/mTopLevel/etc...
	}
}

func (dvw *documentValueWriter) invalidTransitionError(destination mode) error {
	te := TransitionError{
		current:     dvw.stack[dvw.frame].mode,
		destination: destination,
	}
	if dvw.frame != 0 {
		te.parent = dvw.stack[dvw.frame-1].mode
	}
	return te
}

func (dvw *documentValueWriter) ensureValidTransition() error {
	switch dvw.stack[dvw.frame].mode {
	case mElement, mValue:
		return nil
	default:
		return dvw.invalidTransitionError(0)
	}
}

func (dvw *documentValueWriter) WriteArray() (ArrayWriter, error) {
	switch dvw.stack[dvw.frame].mode {
	case mElement, mValue:
	default:
		return nil, dvw.invalidTransitionError(mArray)
	}

	d := bson.NewDocument()
	arr := bson.ArrayFromDocument(d)
	dvw.stack[dvw.frame].d.Append(bson.EC.Array(dvw.stack[dvw.frame].key, arr))
	dvw.push(mArray)
	dvw.stack[dvw.frame].d = d
	return dvw, nil
}

func (dvw *documentValueWriter) WriteBinary(b []byte) error {
	if err := dvw.ensureValidTransition(); err != nil {
		return err
	}
	defer dvw.pop()

	dvw.stack[dvw.frame].d.Append(bson.EC.Binary(dvw.stack[dvw.frame].key, b))
	return nil
}

func (dvw *documentValueWriter) WriteBinaryWithSubtype(b []byte, btype byte) error {
	if err := dvw.ensureValidTransition(); err != nil {
		return err
	}
	defer dvw.pop()

	dvw.stack[dvw.frame].d.Append(bson.EC.BinaryWithSubtype(dvw.stack[dvw.frame].key, b, btype))
	return nil
}

func (dvw *documentValueWriter) WriteBoolean(b bool) error {
	if err := dvw.ensureValidTransition(); err != nil {
		return err
	}
	defer dvw.pop()

	dvw.stack[dvw.frame].d.Append(bson.EC.Boolean(dvw.stack[dvw.frame].key, b))
	return nil
}

func (dvw *documentValueWriter) WriteCodeWithScope(code string) (DocumentWriter, error) {
	switch dvw.stack[dvw.frame].mode {
	case mElement, mValue:
	default:
		return nil, dvw.invalidTransitionError(mCodeWithScope)
	}

	scope := bson.NewDocument()
	dvw.stack[dvw.frame].d.Append(bson.EC.CodeWithScope(dvw.stack[dvw.frame].key, code, scope))
	dvw.push(mDocument)
	dvw.stack[dvw.frame].d = scope
	return dvw, nil
}

func (dvw *documentValueWriter) WriteDBPointer(ns string, oid objectid.ObjectID) error {
	if err := dvw.ensureValidTransition(); err != nil {
		return err
	}
	defer dvw.pop()

	dvw.stack[dvw.frame].d.Append(bson.EC.DBPointer(dvw.stack[dvw.frame].key, ns, oid))
	return nil
}

func (dvw *documentValueWriter) WriteDateTime(dt int64) error {
	if err := dvw.ensureValidTransition(); err != nil {
		return err
	}
	defer dvw.pop()

	dvw.stack[dvw.frame].d.Append(bson.EC.DateTime(dvw.stack[dvw.frame].key, dt))
	return nil
}

func (dvw *documentValueWriter) WriteDecimal128(d128 decimal.Decimal128) error {
	if err := dvw.ensureValidTransition(); err != nil {
		return err
	}
	defer dvw.pop()

	dvw.stack[dvw.frame].d.Append(bson.EC.Decimal128(dvw.stack[dvw.frame].key, d128))
	return nil
}

func (dvw *documentValueWriter) WriteDouble(f64 float64) error {
	if err := dvw.ensureValidTransition(); err != nil {
		return err
	}
	defer dvw.pop()

	dvw.stack[dvw.frame].d.Append(bson.EC.Double(dvw.stack[dvw.frame].key, f64))
	return nil
}

func (dvw *documentValueWriter) WriteInt32(i32 int32) error {
	if err := dvw.ensureValidTransition(); err != nil {
		return err
	}
	defer dvw.pop()

	dvw.stack[dvw.frame].d.Append(bson.EC.Int32(dvw.stack[dvw.frame].key, i32))
	return nil
}

func (dvw *documentValueWriter) WriteInt64(i64 int64) error {
	if err := dvw.ensureValidTransition(); err != nil {
		return err
	}
	defer dvw.pop()

	dvw.stack[dvw.frame].d.Append(bson.EC.Int64(dvw.stack[dvw.frame].key, i64))
	return nil
}

func (dvw *documentValueWriter) WriteJavascript(code string) error {
	if err := dvw.ensureValidTransition(); err != nil {
		return err
	}
	defer dvw.pop()

	dvw.stack[dvw.frame].d.Append(bson.EC.JavaScript(dvw.stack[dvw.frame].key, code))
	return nil
}

func (dvw *documentValueWriter) WriteMaxKey() error {
	if err := dvw.ensureValidTransition(); err != nil {
		return err
	}
	defer dvw.pop()

	dvw.stack[dvw.frame].d.Append(bson.EC.MaxKey(dvw.stack[dvw.frame].key))
	return nil
}

func (dvw *documentValueWriter) WriteMinKey() error {
	if err := dvw.ensureValidTransition(); err != nil {
		return err
	}
	defer dvw.pop()

	dvw.stack[dvw.frame].d.Append(bson.EC.MinKey(dvw.stack[dvw.frame].key))
	return nil
}

func (dvw *documentValueWriter) WriteNull() error {
	if err := dvw.ensureValidTransition(); err != nil {
		return err
	}
	defer dvw.pop()

	dvw.stack[dvw.frame].d.Append(bson.EC.Null(dvw.stack[dvw.frame].key))
	return nil
}

func (dvw *documentValueWriter) WriteObjectID(oid objectid.ObjectID) error {
	if err := dvw.ensureValidTransition(); err != nil {
		return err
	}
	defer dvw.pop()

	dvw.stack[dvw.frame].d.Append(bson.EC.ObjectID(dvw.stack[dvw.frame].key, oid))
	return nil
}

func (dvw *documentValueWriter) WriteRegex(pattern string, options string) error {
	if err := dvw.ensureValidTransition(); err != nil {
		return err
	}
	defer dvw.pop()

	dvw.stack[dvw.frame].d.Append(bson.EC.Regex(dvw.stack[dvw.frame].key, pattern, options))
	return nil
}

func (dvw *documentValueWriter) WriteString(str string) error {
	if err := dvw.ensureValidTransition(); err != nil {
		return err
	}
	defer dvw.pop()

	dvw.stack[dvw.frame].d.Append(bson.EC.String(dvw.stack[dvw.frame].key, str))
	return nil
}

func (dvw *documentValueWriter) WriteDocument() (DocumentWriter, error) {
	if dvw.stack[dvw.frame].mode == mTopLevel {
		return dvw, nil
	}

	switch dvw.stack[dvw.frame].mode {
	case mElement, mValue:
	default:
		return nil, dvw.invalidTransitionError(mDocument)
	}

	d := bson.NewDocument()
	dvw.stack[dvw.frame].d.Append(bson.EC.SubDocument(dvw.stack[dvw.frame].key, d))
	dvw.push(mDocument)
	dvw.stack[dvw.frame].d = d
	return dvw, nil
}

func (dvw *documentValueWriter) WriteSymbol(symbol string) error {
	if err := dvw.ensureValidTransition(); err != nil {
		return err
	}
	defer dvw.pop()

	dvw.stack[dvw.frame].d.Append(bson.EC.Symbol(dvw.stack[dvw.frame].key, symbol))
	return nil
}

func (dvw *documentValueWriter) WriteTimestamp(t uint32, i uint32) error {
	if err := dvw.ensureValidTransition(); err != nil {
		return err
	}
	defer dvw.pop()

	dvw.stack[dvw.frame].d.Append(bson.EC.Timestamp(dvw.stack[dvw.frame].key, t, i))
	return nil
}

func (dvw *documentValueWriter) WriteUndefined() error {
	if err := dvw.ensureValidTransition(); err != nil {
		return err
	}
	defer dvw.pop()

	dvw.stack[dvw.frame].d.Append(bson.EC.Undefined(dvw.stack[dvw.frame].key))
	return nil
}

func (dvw *documentValueWriter) WriteDocumentElement(key string) (ValueWriter, error) {
	switch dvw.stack[dvw.frame].mode {
	case mTopLevel, mDocument:
	default:
		return nil, dvw.invalidTransitionError(mElement)
	}

	dvw.push(mElement)
	dvw.stack[dvw.frame].key = key

	return dvw, nil
}

func (dvw *documentValueWriter) WriteDocumentEnd() error {
	switch dvw.stack[dvw.frame].mode {
	case mTopLevel, mDocument:
	default:
		return fmt.Errorf("incorrect mode to end document: %s", dvw.stack[dvw.frame].mode)
	}

	dvw.pop()
	return nil
}

func (dvw *documentValueWriter) WriteArrayElement() (ValueWriter, error) {
	if dvw.stack[dvw.frame].mode != mArray {
		return nil, dvw.invalidTransitionError(mValue)
	}

	dvw.push(mValue)
	return dvw, nil
}

func (dvw *documentValueWriter) WriteArrayEnd() error {
	if dvw.stack[dvw.frame].mode != mArray {
		return fmt.Errorf("incorrect mode to end array: %s", dvw.stack[dvw.frame].mode)
	}

	dvw.pop()
	return nil
}
