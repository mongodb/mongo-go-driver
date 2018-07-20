package bson

import (
	"errors"
	"fmt"
	"sync"

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
	d    *Document
	key  string
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
	te := transitionError{
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

// NewDocumentValueWriter creates a ValueWriter that adds elements to d.
func NewDocumentValueWriter(d *Document) (ValueWriter, error) {
	if d == nil {
		return nil, errors.New("cannot create ValueWriter with nil *Document")
	}
	return newDocumentValueWriter(d), nil
}

func newDocumentValueWriter(d *Document) *documentValueWriter {
	// We start the document value writer in the mode of writing a document
	dvw := new(documentValueWriter)
	stack := make([]dvwState, 1, 5)
	stack[0] = dvwState{mode: mTopLevel, d: d}
	dvw.stack = stack

	return dvw
}

func (dvw *documentValueWriter) reset(d *Document) {
	dvw.stack = dvw.stack[:1]
	dvw.frame = 0
	dvw.stack[0] = dvwState{mode: mTopLevel, d: d}
}

func (dvw *documentValueWriter) WriteArray() (ArrayWriter, error) {
	if err := dvw.ensureValidTransition(); err != nil {
		return nil, err
	}

	d := NewDocument()
	arr := ArrayFromDocument(d)
	dvw.stack[dvw.frame].d.Append(EC.Array(dvw.stack[dvw.frame].key, arr))
	dvw.push(mArray)
	dvw.stack[dvw.frame].d = d
	return dvw, nil
}

func (dvw *documentValueWriter) WriteBinary(b []byte) error {
	if err := dvw.ensureValidTransition(); err != nil {
		return err
	}
	defer dvw.pop()

	dvw.stack[dvw.frame].d.Append(EC.Binary(dvw.stack[dvw.frame].key, b))
	return nil
}

func (dvw *documentValueWriter) WriteBinaryWithSubtype(b []byte, btype byte) error {
	if err := dvw.ensureValidTransition(); err != nil {
		return err
	}
	defer dvw.pop()

	dvw.stack[dvw.frame].d.Append(EC.BinaryWithSubtype(dvw.stack[dvw.frame].key, b, btype))
	return nil
	// return nil
}

func (dvw *documentValueWriter) WriteBoolean(b bool) error {
	if err := dvw.ensureValidTransition(); err != nil {
		return err
	}
	defer dvw.pop()

	dvw.stack[dvw.frame].d.Append(EC.Boolean(dvw.stack[dvw.frame].key, b))
	return nil
}

func (dvw *documentValueWriter) WriteCodeWithScope(code string) (DocumentWriter, error) {
	if err := dvw.ensureValidTransition(); err != nil {
		return nil, err
	}

	scope := NewDocument()
	dvw.stack[dvw.frame].d.Append(EC.CodeWithScope(dvw.stack[dvw.frame].key, code, scope))
	dvw.push(mDocument)
	dvw.stack[dvw.frame].d = scope
	return dvw, nil
}

func (dvw *documentValueWriter) WriteDBPointer(ns string, oid objectid.ObjectID) error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteDateTime(dt int64) error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteDecimal128(decimal.Decimal128) error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteDouble(float64) error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteInt32(i32 int32) error {
	if err := dvw.ensureValidTransition(); err != nil {
		return err
	}
	defer dvw.pop()

	dvw.stack[dvw.frame].d.Append(EC.Int32(dvw.stack[dvw.frame].key, i32))
	return nil
}

func (dvw *documentValueWriter) WriteInt64(i64 int64) error {
	if err := dvw.ensureValidTransition(); err != nil {
		return err
	}
	defer dvw.pop()

	dvw.stack[dvw.frame].d.Append(EC.Int64(dvw.stack[dvw.frame].key, i64))
	return nil
}

func (dvw *documentValueWriter) WriteJavascript(code string) error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteMaxKey() error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteMinKey() error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteNull() error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteObjectID(objectid.ObjectID) error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteRegex(pattern string, options string) error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteString(string) error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteDocument() (DocumentWriter, error) {
	if dvw.stack[dvw.frame].mode == mTopLevel {
		return dvw, nil
	}

	if err := dvw.ensureValidTransition(); err != nil {
		return nil, err
	}

	d := NewDocument()
	dvw.stack[dvw.frame].d.Append(EC.SubDocument(dvw.stack[dvw.frame].key, d))
	dvw.push(mDocument)
	dvw.stack[dvw.frame].d = d
	return dvw, nil
}

func (dvw *documentValueWriter) WriteSymbol(symbol string) error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteTimestamp(t uint32, i uint32) error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteUndefined() error {
	panic("not implemented")
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
