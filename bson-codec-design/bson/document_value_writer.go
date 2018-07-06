package bson

import (
	"errors"
	"fmt"
	"sync"

	"github.com/mongodb/mongo-go-driver/bson-codec-design/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson-codec-design/bson/objectid"
)

// This pool is used to keep the allocations of documentValueWriters down.
// documentValueWriters retrieved from this pool should have reset called
// on them before use.
var dvwPool = sync.Pool{
	New: func() interface{} {
		return new(documentValueWriter)
	},
}

// dvwMode is used to determine what mode the documentValueWriter is currently
// in.
type dvwMode int

// The modes a documentValueWriter can be in are listed below. A new dvw starts in
// the dvwTopLevel mode, which indicates we are constructing a document. From there
// we can transition to the dvwDocument mode (this is because only the dvwDocument
// mode is used for writing elements). After dvwDocument we transition to dvwElement,
// which represents writing the value of an element. From there are can move to either
// dvwDocument (a subdocument), dvwArray (an array), dvwCodeWithScope (the scope
// component of code with scope), or back to dvwElement (once we've written an element).
// The dvwValue mode operates like the dvwElement mode, but for an array. The difference
// is that dvwElement will use the key that is part of the dvwState, while dvwValue will
// not reference this key.
const (
	_                dvwMode = iota
	dvwTopLevel              // A proper BSON document, not a subdocument
	dvwDocument              // A subdocument, includes things like code with scope
	dvwArray                 // An array
	dvwValue                 // An element of an array, has no key
	dvwElement               // An element of a document, has a key
	dvwCodeWithScope         // A code with scope element, used when assembling the scope component
)

func (dm dvwMode) String() string {
	var str string

	switch dm {
	case dvwTopLevel:
		str = "TopDocumentMode"
	case dvwDocument:
		str = "DocumentMode"
	case dvwArray:
		str = "ArrayMode"
	case dvwValue:
		str = "ArrayValueMode"
	case dvwElement:
		str = "DocumentElementMode"
	case dvwCodeWithScope:
		str = "CodeWithScopeMode"
	default:
		str = "UnknownMode"
	}

	return str
}

type documentValueWriter struct {
	state dvwState
}

type dvwState struct {
	prev *dvwState

	arr   *Array
	doc   *Document
	mode  dvwMode
	key   string
	code  string // used for code with scope
	depth int
}

func (dvws *dvwState) push() {
	dvws.prev = &dvwState{
		prev:  dvws.prev,
		arr:   dvws.arr,
		doc:   dvws.doc,
		mode:  dvws.mode,
		key:   dvws.key,
		code:  dvws.code,
		depth: dvws.depth,
	}
	dvws.depth++
}

func (dvws *dvwState) pop() {
	dvws.arr = dvws.prev.arr
	dvws.doc = dvws.prev.doc
	dvws.mode = dvws.prev.mode
	dvws.key = dvws.prev.key
	dvws.code = dvws.prev.code
	dvws.depth = dvws.prev.depth
	dvws.prev = dvws.prev.prev
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
	return &documentValueWriter{
		state: dvwState{
			mode: dvwTopLevel,
			doc:  d,
		},
	}
}

func (dvw *documentValueWriter) reset(d *Document) {
	dvw.state = dvwState{
		mode: dvwTopLevel,
		doc:  d,
	}
}

func (dvw *documentValueWriter) WriteArray() (ArrayWriter, error) {
	switch dvw.state.mode {
	case dvwElement:
		if dvw.state.doc == nil {
			return nil, errors.New("invalid document state")
		}
	case dvwValue:
		if dvw.state.arr == nil {
			return nil, errors.New("invalid document state")
		}
	default:
		return nil, fmt.Errorf("invalid state transition %s -> %s", dvw.state.prev.mode, dvw.state.mode)
	}

	dvw.state.push()
	dvw.state.arr = NewArray()
	dvw.state.mode = dvwArray
	return dvw, nil
}

func (dvw *documentValueWriter) WriteBinary(b []byte) error {
	switch dvw.state.mode {
	case dvwElement:
		if dvw.state.doc == nil {
			return errors.New("invalid document state")
		}
		dvw.state.doc.Append(EC.Binary(dvw.state.key, b))
	case dvwValue:
		if dvw.state.arr == nil {
			return errors.New("invalid array state")
		}
		dvw.state.arr.Append(VC.Binary(b))
	default:
		return fmt.Errorf("invalid state transition %s -> %s", dvw.state.prev.mode, dvw.state.mode)
	}

	dvw.state.pop()
	return nil
}

func (dvw *documentValueWriter) WriteBinaryWithSubtype(b []byte, btype byte) error {
	switch dvw.state.mode {
	case dvwElement:
		if dvw.state.doc == nil {
			return errors.New("invalid document state")
		}
		dvw.state.doc.Append(EC.BinaryWithSubtype(dvw.state.key, b, btype))
	case dvwValue:
		if dvw.state.arr == nil {
			return errors.New("invalid array state")
		}
		dvw.state.arr.Append(VC.BinaryWithSubtype(b, btype))
	default:
		return fmt.Errorf("invalid state transition %s -> %s", dvw.state.prev.mode, dvw.state.mode)
	}

	dvw.state.pop()
	return nil
}

func (dvw *documentValueWriter) WriteBoolean(b bool) error {
	switch dvw.state.mode {
	case dvwElement:
		if dvw.state.doc == nil {
			return errors.New("invalid document state")
		}
		dvw.state.doc.Append(EC.Boolean(dvw.state.key, b))
	case dvwValue:
		if dvw.state.arr == nil {
			return errors.New("invalid array state")
		}
		dvw.state.arr.Append(VC.Boolean(b))
	default:
		return fmt.Errorf("invalid state transition %s -> %s", dvw.state.prev.mode, dvw.state.mode)
	}

	dvw.state.pop()
	return nil
}

func (dvw *documentValueWriter) WriteCodeWithScope(code string) (DocumentWriter, error) {
	switch dvw.state.mode {
	case dvwElement:
		if dvw.state.doc == nil {
			return nil, errors.New("invalid document state")
		}
	case dvwValue:
		if dvw.state.arr == nil {
			return nil, errors.New("invalid array state")
		}
	default:
		return nil, fmt.Errorf("invalid state transition %s -> %s", dvw.state.prev.mode, dvw.state.mode)
	}

	dvw.state.push()
	dvw.state.mode = dvwCodeWithScope
	dvw.state.code = code
	dvw.state.doc = NewDocument()
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

func (dvw *documentValueWriter) WriteInt32(i int32) error {
	switch dvw.state.mode {
	case dvwElement:
		if dvw.state.doc == nil {
			return errors.New("invalid document state")
		}
		dvw.state.doc.Append(EC.Int32(dvw.state.key, i))
	case dvwValue:
		if dvw.state.arr == nil {
			return errors.New("invalid array state")
		}
		dvw.state.arr.Append(VC.Int32(i))
	default:
		return fmt.Errorf("invalid state transition %s -> %s", dvw.state.prev.mode, dvw.state.mode)
	}

	dvw.state.pop()
	return nil
}

func (dvw *documentValueWriter) WriteInt64(i int64) error {
	switch dvw.state.mode {
	case dvwElement:
		if dvw.state.doc == nil {
			return errors.New("invalid document state")
		}
		dvw.state.doc.Append(EC.Int64(dvw.state.key, i))
	case dvwValue:
		if dvw.state.arr == nil {
			return errors.New("invalid array state")
		}
		dvw.state.arr.Append(VC.Int64(i))
	default:
		return fmt.Errorf("invalid state transition %s -> %s", dvw.state.prev.mode, dvw.state.mode)
	}

	dvw.state.pop()
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
	switch dvw.state.mode {
	case dvwTopLevel:
		dvw.state.push()
		return dvw, nil
	case dvwElement:
		if dvw.state.doc == nil {
			return nil, errors.New("invalid document state")
		}
	case dvwValue:
		if dvw.state.arr == nil {
			return nil, errors.New("invalid document state")
		}
	default:
		return nil, errors.New("invalid top level state transition")
	}

	dvw.state.push()
	dvw.state.doc = NewDocument()
	dvw.state.mode = dvwDocument
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
	switch dvw.state.mode {
	case dvwDocument, dvwTopLevel, dvwCodeWithScope:
		if dvw.state.doc == nil {
			return nil, errors.New("invalid document state")
		}
	default:
		return nil, fmt.Errorf("invalid state transition %s -> %s", dvw.state.prev.mode, dvw.state.mode)
	}

	dvw.state.push()
	dvw.state.key = key
	dvw.state.mode = dvwElement
	return dvw, nil
}

func (dvw *documentValueWriter) WriteDocumentEnd() error {
	switch dvw.state.mode {
	case dvwElement, dvwValue:
		// We're closing a subdocument, we have to pop a stack frame
		dvw.state.pop()
	}

	if dvw.state.mode != dvwTopLevel && dvw.state.mode != dvwDocument && dvw.state.mode != dvwCodeWithScope {
		return fmt.Errorf("incorrect mode to end document %s", dvw.state.mode)
	}

	switch dvw.state.prev.mode {
	case dvwTopLevel:
	case dvwElement:
		if dvw.state.doc == nil || dvw.state.prev.doc == nil {
			return errors.New("invalid document state")
		}
		if dvw.state.mode == dvwCodeWithScope {
			code, scope := dvw.state.code, dvw.state.doc
			dvw.state.prev.doc.Append(EC.CodeWithScope(dvw.state.key, code, scope))
			break
		}
		dvw.state.prev.doc.Append(EC.SubDocument(dvw.state.key, dvw.state.doc))
	case dvwValue:
		if dvw.state.doc == nil || dvw.state.prev.arr == nil {
			return errors.New("invalid array state")
		}
		if dvw.state.mode == dvwCodeWithScope {
			code, scope := dvw.state.code, dvw.state.doc
			dvw.state.prev.arr.Append(VC.CodeWithScope(code, scope))
			break
		}
		dvw.state.prev.arr.Append(VC.Document(dvw.state.doc))
	default:
		return fmt.Errorf("invalid state transition %s -> %s", dvw.state.mode, dvw.state.prev.mode)
	}

	dvw.state.pop()
	return nil
}

func (dvw *documentValueWriter) WriteArrayElement() (ValueWriter, error) {
	switch dvw.state.mode {
	case dvwArray:
		if dvw.state.arr == nil {
			return nil, errors.New("invalid array state")
		}
	default:
		return nil, fmt.Errorf("invalid state transition %s -> %s", dvw.state.prev.mode, dvw.state.mode)
	}

	dvw.state.push()
	dvw.state.mode = dvwValue
	return dvw, nil
}

func (dvw *documentValueWriter) WriteArrayEnd() error {
	switch dvw.state.mode {
	case dvwElement, dvwValue:
		// We're closing a subdocument, we have to pop a stack frame
		dvw.state.pop()
	}

	if dvw.state.mode != dvwArray {
		return fmt.Errorf("incorrect mode to end document %s", dvw.state.mode)
	}

	switch dvw.state.prev.mode {
	case dvwTopLevel:
	case dvwElement:
		if dvw.state.arr == nil || dvw.state.prev.doc == nil {
			return errors.New("invalid document state")
		}
		dvw.state.prev.doc.Append(EC.Array(dvw.state.key, dvw.state.arr))
	case dvwValue:
		if dvw.state.arr == nil || dvw.state.prev.arr == nil {
			return errors.New("invalid array state")
		}
		dvw.state.prev.arr.Append(VC.Array(dvw.state.arr))
	default:
		return fmt.Errorf("invalid state transition %s -> %s", dvw.state.mode, dvw.state.prev.mode)
	}

	dvw.state.pop()
	return nil
}
