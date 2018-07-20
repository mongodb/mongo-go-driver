package bson

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"reflect"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/internal/llbson"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

func bytesFromDoc(doc *Document) []byte {
	b, err := doc.MarshalBSON()
	if err != nil {
		panic(fmt.Errorf("Couldn't marshal BSON document: %v", err))
	}
	return b
}

func TestNewBSONValueWriter(t *testing.T) {
	_, got := NewBSONValueWriter(nil)
	want := errNilWriter
	if !compareErrors(got, want) {
		t.Errorf("Returned error did not match what was expected. got %v; want %v", got, want)
	}

	vw, got := NewBSONValueWriter(errWriter{})
	want = nil
	if !compareErrors(got, want) {
		t.Errorf("Returned error did not match what was expected. got %v; want %v", got, want)
	}
	if vw == nil {
		t.Errorf("Expected non-nil ValueWriter to be returned from NewBSONValueWriter")
	}
}

func TestValueWriter(t *testing.T) {
	header := []byte{0x00, 0x00, 0x00, 0x00}
	oid := objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
	testCases := []struct {
		name   string
		fn     interface{}
		params []interface{}
		want   []byte
	}{
		{
			"WriteBinary",
			(*valueWriter).WriteBinary,
			[]interface{}{[]byte{0x01, 0x02, 0x03}},
			llbson.AppendBinaryElement(header, "foo", 0x00, []byte{0x01, 0x02, 0x03}),
		},
		{
			"WriteBinaryWithSubtype (not 0x02)",
			(*valueWriter).WriteBinaryWithSubtype,
			[]interface{}{[]byte{0x01, 0x02, 0x03}, byte(0xFF)},
			llbson.AppendBinaryElement(header, "foo", 0xFF, []byte{0x01, 0x02, 0x03}),
		},
		{
			"WriteBinaryWithSubtype (0x02)",
			(*valueWriter).WriteBinaryWithSubtype,
			[]interface{}{[]byte{0x01, 0x02, 0x03}, byte(0x02)},
			llbson.AppendBinaryElement(header, "foo", 0x02, []byte{0x01, 0x02, 0x03}),
		},
		{
			"WriteBoolean",
			(*valueWriter).WriteBoolean,
			[]interface{}{true},
			llbson.AppendBooleanElement(header, "foo", true),
		},
		{
			"WriteDBPointer",
			(*valueWriter).WriteDBPointer,
			[]interface{}{"bar", oid},
			llbson.AppendDBPointerElement(header, "foo", "bar", oid),
		},
		{
			"WriteDateTime",
			(*valueWriter).WriteDateTime,
			[]interface{}{int64(12345678)},
			llbson.AppendDateTimeElement(header, "foo", 12345678),
		},
		{
			"WriteDecimal128",
			(*valueWriter).WriteDecimal128,
			[]interface{}{decimal.NewDecimal128(10, 20)},
			llbson.AppendDecimal128Element(header, "foo", decimal.NewDecimal128(10, 20)),
		},
		{
			"WriteDouble",
			(*valueWriter).WriteDouble,
			[]interface{}{float64(3.14159)},
			llbson.AppendDoubleElement(header, "foo", 3.14159),
		},
		{
			"WriteInt32",
			(*valueWriter).WriteInt32,
			[]interface{}{int32(123456)},
			llbson.AppendInt32Element(header, "foo", 123456),
		},
		{
			"WriteInt64",
			(*valueWriter).WriteInt64,
			[]interface{}{int64(1234567890)},
			llbson.AppendInt64Element(header, "foo", 1234567890),
		},
		{
			"WriteJavascript",
			(*valueWriter).WriteJavascript,
			[]interface{}{"var foo = 'bar';"},
			llbson.AppendJavaScriptElement(header, "foo", "var foo = 'bar';"),
		},
		{
			"WriteMaxKey",
			(*valueWriter).WriteMaxKey,
			[]interface{}{},
			llbson.AppendMaxKeyElement(header, "foo"),
		},
		{
			"WriteMinKey",
			(*valueWriter).WriteMinKey,
			[]interface{}{},
			llbson.AppendMinKeyElement(header, "foo"),
		},
		{
			"WriteNull",
			(*valueWriter).WriteNull,
			[]interface{}{},
			llbson.AppendNullElement(header, "foo"),
		},
		{
			"WriteObjectID",
			(*valueWriter).WriteObjectID,
			[]interface{}{oid},
			llbson.AppendObjectIDElement(header, "foo", oid),
		},
		{
			"WriteRegex",
			(*valueWriter).WriteRegex,
			[]interface{}{"bar", "baz"},
			llbson.AppendRegexElement(header, "foo", "bar", "baz"),
		},
		{
			"WriteString",
			(*valueWriter).WriteString,
			[]interface{}{"hello, world!"},
			llbson.AppendStringElement(header, "foo", "hello, world!"),
		},
		{
			"WriteSymbol",
			(*valueWriter).WriteSymbol,
			[]interface{}{"symbollolz"},
			llbson.AppendSymbolElement(header, "foo", "symbollolz"),
		},
		{
			"WriteTimestamp",
			(*valueWriter).WriteTimestamp,
			[]interface{}{uint32(10), uint32(20)},
			llbson.AppendTimestampElement(header, "foo", 10, 20),
		},
		{
			"WriteUndefined",
			(*valueWriter).WriteUndefined,
			[]interface{}{},
			llbson.AppendUndefinedElement(header, "foo"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fn := reflect.ValueOf(tc.fn)
			if fn.Kind() != reflect.Func {
				t.Fatalf("fn must be of kind Func but it is a %v", fn.Kind())
			}
			if fn.Type().NumIn() != len(tc.params)+1 || fn.Type().In(0) != reflect.TypeOf((*valueWriter)(nil)) {
				t.Fatalf("fn must have at least one parameter and the first parameter must be a *valueWriter")
			}
			if fn.Type().NumOut() != 1 || fn.Type().Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
				t.Fatalf("fn must have one return value and it must be an error.")
			}
			params := make([]reflect.Value, 1, len(tc.params)+1)
			vw := newValueWriter(ioutil.Discard)
			params[0] = reflect.ValueOf(vw)
			for _, param := range tc.params {
				params = append(params, reflect.ValueOf(param))
			}
			_, err := vw.WriteDocument()
			noerr(t, err)
			_, err = vw.WriteDocumentElement("foo")
			noerr(t, err)

			results := fn.Call(params)
			if !results[0].IsValid() {
				err = results[0].Interface().(error)
			} else {
				err = nil
			}
			noerr(t, err)
			got := vw.buf
			want := tc.want
			if !bytes.Equal(got, want) {
				t.Errorf("Bytes are not equal.\n\tgot %v\n\twant %v", got, want)
			}

			t.Run("incorrect transition", func(t *testing.T) {
				vw = newValueWriter(ioutil.Discard)
				results := fn.Call(params)
				got := results[0].Interface().(error)
				want := transitionError{current: mTopLevel}
				if !compareErrors(got, want) {
					t.Errorf("Errors do not match. got %v; want %v", got, want)
				}
			})
		})
	}

	t.Run("WriteArray", func(t *testing.T) {
		vw := newValueWriter(ioutil.Discard)
		vw.push(mArray)
		want := transitionError{current: mArray, destination: mArray, parent: mTopLevel}
		_, got := vw.WriteArray()
		if !compareErrors(got, want) {
			t.Errorf("Did not get expected error. got %v; want %v", got, want)
		}
	})
	t.Run("WriteCodeWithScope", func(t *testing.T) {
		vw := newValueWriter(ioutil.Discard)
		vw.push(mArray)
		want := transitionError{current: mArray, destination: mCodeWithScope, parent: mTopLevel}
		_, got := vw.WriteCodeWithScope("")
		if !compareErrors(got, want) {
			t.Errorf("Did not get expected error. got %v; want %v", got, want)
		}
	})
	t.Run("WriteDocument", func(t *testing.T) {
		vw := newValueWriter(ioutil.Discard)
		vw.push(mArray)
		want := transitionError{current: mArray, destination: mDocument, parent: mTopLevel}
		_, got := vw.WriteDocument()
		if !compareErrors(got, want) {
			t.Errorf("Did not get expected error. got %v; want %v", got, want)
		}
	})
	t.Run("WriteDocumentElement", func(t *testing.T) {
		vw := newValueWriter(ioutil.Discard)
		vw.push(mElement)
		want := transitionError{current: mElement, destination: mElement, parent: mTopLevel}
		_, got := vw.WriteDocumentElement("")
		if !compareErrors(got, want) {
			t.Errorf("Did not get expected error. got %v; want %v", got, want)
		}
	})
	t.Run("WriteDocumentEnd", func(t *testing.T) {
		vw := newValueWriter(ioutil.Discard)
		vw.push(mElement)
		want := fmt.Errorf("incorrect mode to end document: %s", mElement)
		got := vw.WriteDocumentEnd()
		if !compareErrors(got, want) {
			t.Errorf("Did not get expected error. got %v; want %v", got, want)
		}
		vw.pop()
		vw.buf = append(vw.buf, make([]byte, 1023)...)
		maxSize = 512
		want = errMaxDocumentSizeExceeded{size: 1024}
		got = vw.WriteDocumentEnd()
		if !compareErrors(got, want) {
			t.Errorf("Did not get expected error. got %v; want %v", got, want)
		}
		maxSize = math.MaxInt32
		want = errors.New("what a nice fake error we have here")
		vw.w = errWriter{err: want}
		got = vw.WriteDocumentEnd()
		if !compareErrors(got, want) {
			t.Errorf("Did not get expected error. got %v; want %v", got, want)
		}
	})
	t.Run("WriteArrayElement", func(t *testing.T) {
		vw := newValueWriter(ioutil.Discard)
		vw.push(mElement)
		want := transitionError{current: mElement, destination: mValue, parent: mTopLevel}
		_, got := vw.WriteArrayElement()
		if !compareErrors(got, want) {
			t.Errorf("Did not get expected error. got %v; want %v", got, want)
		}
	})
	t.Run("WriteArrayEnd", func(t *testing.T) {
		vw := newValueWriter(ioutil.Discard)
		vw.push(mElement)
		want := fmt.Errorf("incorrect mode to end array: %s", mElement)
		got := vw.WriteArrayEnd()
		if !compareErrors(got, want) {
			t.Errorf("Did not get expected error. got %v; want %v", got, want)
		}
		vw.push(mArray)
		vw.buf = append(vw.buf, make([]byte, 1019)...)
		maxSize = 512
		want = errMaxDocumentSizeExceeded{size: 1024}
		got = vw.WriteArrayEnd()
		if !compareErrors(got, want) {
			t.Errorf("Did not get expected error. got %v; want %v", got, want)
		}
		maxSize = math.MaxInt32
	})
}

type errWriter struct {
	err error
}

func (ew errWriter) Write([]byte) (int, error) { return 0, ew.err }

func TestValueWriterOLD(t *testing.T) {
	testCases := []struct {
		name string
		fn   func(*testing.T, *valueWriter)
		want []byte
	}{
		{
			"simple document",
			vwBasicDoc,
			bytesFromDoc(NewDocument(EC.Boolean("foo", true))),
		},
		{
			"nested document",
			vwNestedDoc,
			bytesFromDoc(NewDocument(EC.SubDocumentFromElements("foo", EC.Boolean("bar", true)))),
		},
		{
			"simple array",
			vwBasicArray,
			bytesFromDoc(NewDocument(EC.ArrayFromElements("foo", VC.Boolean(true)))),
		},
		{
			"code with scope",
			vwCodeWithScopeNoNested,
			bytesFromDoc(NewDocument(EC.CodeWithScope("foo", "var hello = world;", NewDocument(EC.Boolean("bar", false))))),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := make(writer, 0, 1024)
			vw := newValueWriter(&got)
			tc.fn(t, vw)
			if !bytes.Equal(got, tc.want) {
				t.Errorf("Documents are not equal. got %v; want %v", Reader(got), Reader(tc.want))
				t.Errorf("Bytes:\n%v\n%v", got, tc.want)
			}
		})
	}
}

func vwBasicDoc(t *testing.T, vw *valueWriter) {
	dw, err := vw.WriteDocument()
	noerr(t, err)
	vw2, err := dw.WriteDocumentElement("foo")
	noerr(t, err)
	err = vw2.WriteBoolean(true)
	noerr(t, err)
	err = dw.WriteDocumentEnd()
	noerr(t, err)

	return
}

func vwBasicArray(t *testing.T, vw *valueWriter) {
	dw, err := vw.WriteDocument()
	noerr(t, err)
	vw2, err := dw.WriteDocumentElement("foo")
	noerr(t, err)
	aw, err := vw2.WriteArray()
	noerr(t, err)
	vw2, err = aw.WriteArrayElement()
	noerr(t, err)
	err = vw2.WriteBoolean(true)
	noerr(t, err)
	err = aw.WriteArrayEnd()
	noerr(t, err)
	err = dw.WriteDocumentEnd()
	noerr(t, err)

	return
}

func vwNestedDoc(t *testing.T, vw *valueWriter) {
	dw, err := vw.WriteDocument()
	noerr(t, err)
	vw2, err := dw.WriteDocumentElement("foo")
	noerr(t, err)
	dw2, err := vw2.WriteDocument()
	noerr(t, err)
	vw3, err := dw2.WriteDocumentElement("bar")
	noerr(t, err)
	err = vw3.WriteBoolean(true)
	noerr(t, err)
	err = dw2.WriteDocumentEnd()
	noerr(t, err)
	err = dw.WriteDocumentEnd()
	noerr(t, err)

	return
}

func vwCodeWithScopeNoNested(t *testing.T, vw *valueWriter) {
	dw, err := vw.WriteDocument()
	noerr(t, err)
	vw2, err := dw.WriteDocumentElement("foo")
	noerr(t, err)
	dw2, err := vw2.WriteCodeWithScope("var hello = world;")
	noerr(t, err)
	vw2, err = dw2.WriteDocumentElement("bar")
	noerr(t, err)
	err = vw2.WriteBoolean(false)
	noerr(t, err)
	err = dw2.WriteDocumentEnd()
	noerr(t, err)
	err = dw.WriteDocumentEnd()
	noerr(t, err)

	return
}
