package bson

import (
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/mongodb/mongo-go-driver/bson-codec-design/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson-codec-design/bson/objectid"
)

type vwMode int

const (
	_ vwMode = iota
	vwTopLevel
	vwDocument
	vwArray
	vwValue
	vwElement
	vwCodeWithScope
)

func (vm vwMode) String() string {
	var str string

	switch vm {
	case vwTopLevel:
		str = "TopLevel"
	case vwDocument:
		str = "DocumentMode"
	case vwArray:
		str = "ArrayMode"
	case vwValue:
		str = "ValueMode"
	case vwElement:
		str = "ElementMode"
	case vwCodeWithScope:
		str = "CodeWithScopeMode"
	default:
		str = "UnknownMode"
	}

	return str
}

type valueWriter struct {
	w     io.Writer // Will be nil if original io.Writer was also an io.WriterAt
	wa    io.WriterAt
	state vwState
}

type vwState struct {
	prev *vwState

	wa     io.WriterAt
	mode   vwMode
	key    string
	arrkey int
	depth  int

	offset int64 // where to write the next set of bytes
	size   int64 // current size of the document
	start  int64 // start of current document
}

func (vws *vwState) push() {
	vws.prev = &vwState{
		prev:   vws.prev,
		wa:     vws.wa,
		mode:   vws.mode,
		key:    vws.key,
		arrkey: vws.arrkey,
		depth:  vws.depth,
		offset: vws.offset,
		size:   vws.size,
		start:  vws.start,
	}
	vws.depth++
	// TODO: When we push we should set the size to zero and when we pop we should add the current
	// frame's size to the previous frame's size.
}

func (vws *vwState) pop() {
	vws.wa = vws.prev.wa
	vws.mode = vws.prev.mode
	vws.key = vws.prev.key
	vws.arrkey = vws.prev.arrkey
	vws.depth = vws.prev.depth
	vws.offset = vws.prev.offset
	// TODO: When we pop we should set the new frame's size to the previous frame's size plus the
	// current frame's size. We should do this here since this is the behavior we will always want.
	vws.size = vws.prev.size
	vws.start = vws.prev.start

	vws.prev = vws.prev.prev
}

// NewBSONValueWriter creates a ValueWriter that writes BSON to w.
//
// If w is an io.WriterAt it will be used directly during the value writing
// process. If it is not an io.WriterAt, an internal io.WriterAt will be used
// and the entire document will be buffered before being written to w.
func NewBSONValueWriter(w io.Writer) (ValueWriter, error) {
	if w == nil {
		return nil, errors.New("cannot create a ValueWriter from a nil io.Writer")
	}
	return newValueWriter(w), nil
}

func newValueWriter(w io.Writer) *valueWriter {
	vw := new(valueWriter)
	if wa, ok := w.(io.WriterAt); !ok {
		// TODO: figure out how large we should make this by default.
		sliceWriter := make(writer, 0, 256)
		vw.wa = &sliceWriter
		vw.w = w
	} else {
		vw.wa = wa
	}
	vw.state = vwState{
		mode: vwTopLevel,
		wa:   vw.wa,
	}
	return vw
}

func (vw *valueWriter) invalidTransitionErr() error {
	if vw.state.prev == nil {
		return fmt.Errorf("invalid state transition <nil> -> %s", vw.state.mode)
	}
	return fmt.Errorf("invalid state transition %s -> %s", vw.state.prev.mode, vw.state.mode)
}

func (vw *valueWriter) writeElementKey(t Type) error {
	// We need to write the type and the key
	arr := [1]byte{byte(t)}
	n, err := vw.state.wa.WriteAt(arr[:], vw.state.offset)
	vw.state.offset += int64(n)
	vw.state.size += int64(n)
	if err != nil {
		return err
	}
	n, err = vw.state.wa.WriteAt([]byte(vw.state.key), vw.state.offset)
	vw.state.offset += int64(n)
	vw.state.size += int64(n)
	if err != nil {
		return err
	}

	return vw.writeNullByte()
}

func (vw *valueWriter) writeArrayKey(t Type) error {
	// We need to write the type and the array key
	arr := [1]byte{byte(t)}
	n, err := vw.state.wa.WriteAt(arr[:], vw.state.offset)
	vw.state.offset += int64(n)
	vw.state.size += int64(n)
	if err != nil {
		return err
	}
	// TODO: Do this with a cache of the first 1000 or so array keys.
	n, err = vw.state.wa.WriteAt([]byte(strconv.Itoa(vw.state.arrkey)), vw.state.offset)
	vw.state.offset += int64(n)
	vw.state.size += int64(n)
	if err != nil {
		return err
	}

	return vw.writeNullByte()
}

func (vw *valueWriter) writeNullByte() error {
	arr := [1]byte{0x00}
	n, err := vw.state.wa.WriteAt(arr[:], vw.state.offset)
	vw.state.offset += int64(n)
	vw.state.size += int64(n)
	if err != nil {
		return err
	}

	return nil
}

func (vw *valueWriter) writeString(str string) error {
	size := len(str) + 1
	err := vw.writeSize(int64(size))
	if err != nil {
		return err
	}
	n, err := vw.state.wa.WriteAt([]byte(str), vw.state.offset)
	vw.state.offset += int64(n)
	vw.state.size += int64(n)
	if err != nil {
		return err
	}

	return vw.writeNullByte()
}

func (vw *valueWriter) WriteArray() (ArrayWriter, error) {
	switch vw.state.mode {
	case vwElement:
		err := vw.writeElementKey(TypeArray)
		if err != nil {
			return nil, err
		}
		vw.state.push()
		vw.state.arrkey = -1 // Set to negative one so we can always do arrkey++ when writing array element
		vw.state.mode = vwArray
		vw.state.start = vw.state.offset
		vw.state.offset += 4 // Make room for the document length
		vw.state.size = 4    // New document, we need to set the size to 4 to start (we don't count the null byte)
	case vwValue:
		err := vw.writeArrayKey(TypeArray)
		if err != nil {
			return nil, err
		}
		vw.state.push()
		vw.state.arrkey = -1 // Set to negative one so we can always do arrkey++ when writing array element
		vw.state.mode = vwArray
		vw.state.start = vw.state.offset
		vw.state.offset += 4 // Make room for the document length
		vw.state.size = 4    // New document, we need to set the size to 4 to start (we don't count the null byte)
	default:
		return nil, fmt.Errorf("invalid state transition %s -> %s", vw.state.prev.mode, vw.state.mode)
	}

	return vw, nil
}

func (vw *valueWriter) WriteBinary(b []byte) error {
	panic("not implemented")
}

func (vw *valueWriter) WriteBinaryWithSubtype(b []byte, btype byte) error {
	panic("not implemented")
}

func (vw *valueWriter) WriteBoolean(b bool) error {
	switch vw.state.mode {
	case vwElement:
		err := vw.writeElementKey(TypeBoolean)
		if err != nil {
			return err
		}
	case vwValue:
		err := vw.writeArrayKey(TypeBoolean)
		if err != nil {
			return err
		}
	default:
		return vw.invalidTransitionErr()
	}

	var val byte
	if b {
		val = 0x01
	} else {
		val = 0x00
	}
	n, err := vw.state.wa.WriteAt([]byte{val}, vw.state.offset)
	vw.state.offset += int64(n)
	vw.state.size += int64(n)
	if err != nil {
		return err
	}
	// We need to advance the offset and size of the previous stack frame
	offset, size := vw.state.offset, vw.state.size
	vw.state.pop()
	vw.state.offset, vw.state.size = offset, size
	return nil
}

func (vw *valueWriter) WriteCodeWithScope(code string) (DocumentWriter, error) {
	switch vw.state.mode {
	case vwElement:
		err := vw.writeElementKey(TypeCodeWithScope)
		if err != nil {
			return nil, err
		}
	case vwValue:
		err := vw.writeArrayKey(TypeCodeWithScope)
		if err != nil {
			return nil, err
		}
	default:
		return nil, vw.invalidTransitionErr()
	}

	vw.state.push()
	vw.state.start = vw.state.offset
	vw.state.offset += 4 //length
	vw.state.size = 4    // New document, so we set the size to 4 (we don't count the null byte)
	vw.state.mode = vwCodeWithScope

	err := vw.writeString(code)
	if err != nil {
		return nil, err
	}

	return vw.WriteDocument()
}

func (vw *valueWriter) WriteDBPointer(ns string, oid objectid.ObjectID) error {
	panic("not implemented")
}

func (vw *valueWriter) WriteDateTime(dt int64) error {
	panic("not implemented")
}

func (vw *valueWriter) WriteDecimal128(decimal.Decimal128) error {
	panic("not implemented")
}

func (vw *valueWriter) WriteDouble(float64) error {
	panic("not implemented")
}

func (vw *valueWriter) WriteInt32(int32) error {
	panic("not implemented")
}

func (vw *valueWriter) WriteInt64(int64) error {
	panic("not implemented")
}

func (vw *valueWriter) WriteJavascript(code string) error {
	panic("not implemented")
}

func (vw *valueWriter) WriteMaxKey() error {
	panic("not implemented")
}

func (vw *valueWriter) WriteMinKey() error {
	panic("not implemented")
}

func (vw *valueWriter) WriteNull() error {
	panic("not implemented")
}

func (vw *valueWriter) WriteObjectID(objectid.ObjectID) error {
	panic("not implemented")
}

func (vw *valueWriter) WriteRegex(pattern string, options string) error {
	panic("not implemented")
}

func (vw *valueWriter) WriteString(string) error {
	panic("not implemented")
}

func (vw *valueWriter) WriteDocument() (DocumentWriter, error) {
	switch vw.state.mode {
	case vwTopLevel:
		vw.state.start = vw.state.offset
		vw.state.offset += 4 // length
		vw.state.size += 4   // New document, so we set the size to 4 (we don't count the null byte)
		return vw, nil
	case vwElement:
		err := vw.writeElementKey(TypeEmbeddedDocument)
		if err != nil {
			return nil, err
		}
	case vwValue:
		err := vw.writeArrayKey(TypeEmbeddedDocument)
		if err != nil {
			return nil, err
		}
	case vwCodeWithScope:
	default:
		return nil, vw.invalidTransitionErr()
	}

	vw.state.push()
	vw.state.start = vw.state.offset
	vw.state.offset += 4 //length
	vw.state.size = 4    // New document, so we set the size to 4 (we don't count the null byte)
	vw.state.mode = vwDocument
	return vw, nil
}

func (vw *valueWriter) WriteSymbol(symbol string) error {
	panic("not implemented")
}

func (vw *valueWriter) WriteTimestamp(t uint32, i uint32) error {
	panic("not implemented")
}

func (vw *valueWriter) WriteUndefined() error {
	panic("not implemented")
}

func (vw *valueWriter) WriteDocumentElement(key string) (ValueWriter, error) {
	switch vw.state.mode {
	case vwDocument, vwTopLevel:
	default:
		return nil, vw.invalidTransitionErr()
	}

	vw.state.push()
	vw.state.key = key
	vw.state.mode = vwElement
	return vw, nil
}

func (vw *valueWriter) WriteDocumentEnd() error {
	if vw.state.mode != vwTopLevel && vw.state.mode != vwDocument {
		return fmt.Errorf("incorrect mode to end document %s", vw.state.mode)
	}

	err := vw.writeNullByte()
	if err != nil {
		return err
	}
	err = vw.writeDocumentSize()
	if err != nil {
		return err
	}

	if vw.state.prev != nil && vw.state.prev.mode == vwCodeWithScope {
		offset, size := vw.state.offset, vw.state.size
		vw.state.pop()
		vw.state.size += size
		vw.state.offset = offset
		err = vw.writeDocumentSize()
		if err != nil {
			return err
		}
	}

	switch vw.state.mode {
	case vwDocument, vwCodeWithScope:
		offset, size := vw.state.offset, vw.state.size
		vw.state.pop()                     // pop: vwDocument -> vwElement
		vw.state.prev.size = vw.state.size // We need to carry the size backward....
		vw.state.pop()                     // pop: vwElement -> vwDocment/vwArray/vwTopLevel
		vw.state.offset = offset
		vw.state.size += size
	}

	return nil
}

func (vw *valueWriter) WriteArrayElement() (ValueWriter, error) {
	if vw.state.mode != vwArray {
		return nil, vw.invalidTransitionErr()
	}

	vw.state.arrkey++
	vw.state.push()
	vw.state.mode = vwValue
	return vw, nil
}

func (vw *valueWriter) WriteArrayEnd() error {
	if vw.state.mode != vwArray {
		return fmt.Errorf("incorrect mode to end array %s", vw.state.mode)
	}

	err := vw.writeNullByte()
	if err != nil {
		return err
	}
	err = vw.writeDocumentSize()
	if err != nil {
		return err
	}

	switch vw.state.mode {
	case vwArray:
		offset, size := vw.state.offset, vw.state.size
		vw.state.pop()                     // pop: vwArray -> vwElement
		vw.state.prev.size = vw.state.size // We need to carry the size backward....
		vw.state.pop()                     // pop: vwElement -> vwDocment/vwArray/vwTopLevel
		vw.state.offset = offset
		vw.state.size += size
	}

	return nil
}

func (vw *valueWriter) writeSize(size int64) error {
	var arr [4]byte
	arr[0] = byte(size)
	arr[1] = byte(size >> 8)
	arr[2] = byte(size >> 16)
	arr[3] = byte(size >> 24)
	_, err := vw.state.wa.WriteAt(arr[:], vw.state.offset)
	vw.state.offset += 4
	vw.state.size += 4
	return err
}

func (vw *valueWriter) writeDocumentSize() error {
	var arr [4]byte
	arr[0] = byte(vw.state.size)
	arr[1] = byte(vw.state.size >> 8)
	arr[2] = byte(vw.state.size >> 16)
	arr[3] = byte(vw.state.size >> 24)
	_, err := vw.state.wa.WriteAt(arr[:], vw.state.start)
	return err
}
