package bson

import (
	"bytes"
	"testing"

	"github.com/10gen/mongo-go-driver/bson/decimal"
	"github.com/stretchr/testify/require"
)

func requireElementsEqual(t *testing.T, expected *Element, actual *Element) {
	requireValuesEqual(t, expected.value, actual.value)
}

func requireValuesEqual(t *testing.T, expected *Value, actual *Value) {
	require.Equal(t, expected.start, actual.start)
	require.Equal(t, expected.offset, actual.offset)

	require.True(t, bytes.Equal(expected.data, actual.data))

	if expected.d == nil {
		require.Nil(t, actual.d)
	} else {
		require.NotNil(t, actual.d)
		require.Equal(t, expected.d.IgnoreNilInsert, actual.d.IgnoreNilInsert)

		require.Equal(t, len(expected.d.elems), len(actual.d.elems))
		for i := range expected.d.elems {
			requireElementsEqual(t, expected.d.elems[i], actual.d.elems[i])
		}

		require.Equal(t, len(expected.d.index), len(actual.d.index))
		for i := range expected.d.index {
			require.Equal(t, expected.d.index[i], actual.d.index[i])
		}
	}
}

func TestConstructor(t *testing.T) {
	t.Run("Document", func(t *testing.T) {
		t.Run("double", func(t *testing.T) {
			buf := []byte{
				// type
				0x1,
				// key
				0x66, 0x6f, 0x6f, 0x0,
				// value
				0x6e, 0x86, 0x1b, 0xf0, 0xf9,
				0x21, 0x9, 0x40,
			}

			expected := &Element{&Value{start: 0, offset: 5, data: buf, d: nil}}
			actual := C.Double("foo", 3.14159)

			requireElementsEqual(t, expected, actual)
		})

		t.Run("String", func(t *testing.T) {
			buf := []byte{
				// type
				0x2,
				// key
				0x66, 0x6f, 0x6f, 0x0,
				// value - string length
				0x4, 0x0, 0x0, 0x0,
				// value - string
				0x62, 0x61, 0x72, 0x0,
			}

			expected := &Element{&Value{start: 0, offset: 5, data: buf, d: nil}}
			actual := C.String("foo", "bar")

			requireElementsEqual(t, expected, actual)
		})

		t.Run("SubDocument", func(t *testing.T) {
			buf := []byte{
				// type
				0x3,
				// key
				0x66, 0x6f, 0x6f, 0x0,
			}
			d := NewDocument(C.String("bar", "baz"))

			expected := &Element{&Value{start: 0, offset: 5, data: buf, d: d}}
			actual := C.SubDocument("foo", d)

			requireElementsEqual(t, expected, actual)
		})

		t.Run("SubDocumentFromElements", func(t *testing.T) {
			buf := []byte{
				// type
				0x3,
				// key
				0x66, 0x6f, 0x6f, 0x0,
			}
			e := C.String("bar", "baz")
			d := NewDocument(e)

			expected := &Element{&Value{start: 0, offset: 5, data: buf, d: d}}
			actual := C.SubDocumentFromElements("foo", e)

			requireElementsEqual(t, expected, actual)
		})

		t.Run("SubDocumentFromReader", func(t *testing.T) {
			buf := []byte{
				// type
				0x3,
				// key
				0x66, 0x6f, 0x6f, 0x0,
				0x0A, 0x00, 0x00, 0x00,
				0x0A, 'b', 'a', 'r', 0x00,
				0x00,
			}
			rdr := Reader{
				0x0A, 0x00, 0x00, 0x00,
				0x0A, 'b', 'a', 'r', 0x00,
				0x00,
			}

			expected := &Element{&Value{start: 0, offset: 5, data: buf}}
			actual := C.SubDocumentFromReader("foo", rdr)

			requireElementsEqual(t, expected, actual)
		})

		t.Run("Array", func(t *testing.T) {
			buf := []byte{
				// type
				0x4,
				// key
				0x66, 0x6f, 0x6f, 0x0,
			}
			a := NewArray(AC.String("bar"), AC.Double(-2.7))

			expected := &Element{&Value{start: 0, offset: 5, data: buf, d: a.doc}}
			actual := C.Array("foo", a)

			requireElementsEqual(t, expected, actual)
		})

		t.Run("ArrayFromElements", func(t *testing.T) {
			buf := []byte{
				// type
				0x4,
				// key
				0x66, 0x6f, 0x6f, 0x0,
			}
			e1 := AC.String("bar")
			e2 := AC.Double(-2.7)
			a := NewArray(e1, e2)

			expected := &Element{&Value{start: 0, offset: 5, data: buf, d: a.doc}}
			actual := C.ArrayFromElements("foo", e1, e2)

			requireElementsEqual(t, expected, actual)
		})

		t.Run("binary", func(t *testing.T) {
			buf := []byte{
				// type
				0x5,
				// key
				0x66, 0x6f, 0x6f, 0x0,
				// value - binary length
				0x7, 0x0, 0x0, 0x0,
				// value - binary subtype
				0x0,
				// value - binary data
				0x8, 0x6, 0x7, 0x5, 0x3, 0x0, 0x9,
			}

			expected := &Element{&Value{start: 0, offset: 5, data: buf, d: nil}}
			actual := C.Binary("foo", []byte{8, 6, 7, 5, 3, 0, 9})

			requireElementsEqual(t, expected, actual)
		})

		t.Run("BinaryWithSubtype", func(t *testing.T) {
			buf := []byte{
				// type
				0x5,
				// key
				0x66, 0x6f, 0x6f, 0x0,
				// value - binary length
				0xb, 0x0, 0x0, 0x0,
				// value - binary subtype
				0x2,
				//
				0x07, 0x00, 0x00, 0x00,
				// value - binary data
				0x8, 0x6, 0x7, 0x5, 0x3, 0x0, 0x9,
			}

			expected := &Element{&Value{start: 0, offset: 5, data: buf, d: nil}}
			actual := C.BinaryWithSubtype("foo", []byte{8, 6, 7, 5, 3, 0, 9}, 2)

			requireElementsEqual(t, expected, actual)
		})

		t.Run("undefined", func(t *testing.T) {
			buf := []byte{
				// type
				0x6,
				// key
				0x66, 0x6f, 0x6f, 0x0,
			}

			expected := &Element{&Value{start: 0, offset: 5, data: buf, d: nil}}
			actual := C.Undefined("foo")

			requireElementsEqual(t, expected, actual)
		})

		t.Run("objectID", func(t *testing.T) {
			buf := []byte{
				// type
				0x7,
				// key
				0x66, 0x6f, 0x6f, 0x0,
				// value
				0x5a, 0x15, 0xd0, 0xa4, 0xd5, 0xda, 0xa5, 0xf1, 0x0a, 0x5e, 0x10, 0x89,
			}

			expected := &Element{&Value{start: 0, offset: 5, data: buf, d: nil}}
			actual := C.ObjectID(
				"foo",
				[12]byte{0x5a, 0x15, 0xd0, 0xa4, 0xd5, 0xda, 0xa5, 0xf1, 0x0a, 0x5e, 0x10, 0x89},
			)

			requireElementsEqual(t, expected, actual)
		})

		t.Run("Boolean", func(t *testing.T) {
			buf := []byte{
				// type
				0x8,
				// key
				0x66, 0x6f, 0x6f, 0x0,
				// value
				0x0,
			}

			expected := &Element{&Value{start: 0, offset: 5, data: buf, d: nil}}
			actual := C.Boolean("foo", false)

			requireElementsEqual(t, expected, actual)
		})

		t.Run("dateTime", func(t *testing.T) {
			buf := []byte{
				// type
				0x9,
				// key
				0x66, 0x6f, 0x6f, 0x0,
				// value
				0x11, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			}

			expected := &Element{&Value{start: 0, offset: 5, data: buf, d: nil}}
			actual := C.DateTime("foo", 17)

			requireElementsEqual(t, expected, actual)
		})

		t.Run("Null", func(t *testing.T) {
			buf := []byte{
				// type
				0xa,
				// key
				0x66, 0x6f, 0x6f, 0x0,
			}

			expected := &Element{&Value{start: 0, offset: 5, data: buf, d: nil}}
			actual := C.Null("foo")

			requireElementsEqual(t, expected, actual)
		})

		t.Run("regex", func(t *testing.T) {
			buf := []byte{
				// type
				0xb,
				// key
				0x66, 0x6f, 0x6f, 0x0,
				// value - pattern
				0x62, 0x61, 0x72, 0x0,
				// value - options
				0x69, 0x0,
			}

			expected := &Element{&Value{start: 0, offset: 5, data: buf, d: nil}}
			actual := C.Regex("foo", "bar", "i")

			requireElementsEqual(t, expected, actual)
		})

		t.Run("dbPointer", func(t *testing.T) {
			buf := []byte{
				// type
				0xc,
				// key
				0x66, 0x6f, 0x6f, 0x0,
				// value - namespace length
				0x4, 0x0, 0x0, 0x0,
				// value - namespace
				0x62, 0x61, 0x72, 0x0,
				// value - oid
				0x5a, 0x15, 0xd0, 0xa4, 0xd5, 0xda, 0xa5, 0xf1, 0x0a, 0x5e, 0x10, 0x89,
			}

			expected := &Element{&Value{start: 0, offset: 5, data: buf, d: nil}}
			actual := C.DBPointer(
				"foo",
				"bar",
				[12]byte{0x5a, 0x15, 0xd0, 0xa4, 0xd5, 0xda, 0xa5, 0xf1, 0x0a, 0x5e, 0x10, 0x89},
			)

			requireElementsEqual(t, expected, actual)
		})

		t.Run("JavaScriptCode", func(t *testing.T) {
			buf := []byte{
				// type
				0xd,
				// key
				0x66, 0x6f, 0x6f, 0x0,
				// value - code length
				0xd, 0x0, 0x0, 0x0,
				// value - code
				0x76, 0x61, 0x72, 0x20, 0x62, 0x61, 0x72, 0x20, 0x3d, 0x20, 0x33, 0x3b, 0x0,
			}

			expected := &Element{&Value{start: 0, offset: 5, data: buf, d: nil}}
			actual := C.JavaScript("foo", "var bar = 3;")

			requireElementsEqual(t, expected, actual)
		})

		t.Run("symbol", func(t *testing.T) {
			buf := []byte{
				// type
				0xe,
				// key
				0x66, 0x6f, 0x6f, 0x0,
				// value - string length
				0x4, 0x0, 0x0, 0x0,
				// value - string
				0x62, 0x61, 0x72, 0x0,
			}

			expected := &Element{&Value{start: 0, offset: 5, data: buf, d: nil}}
			actual := C.Symbol("foo", "bar")

			requireElementsEqual(t, expected, actual)
		})

		t.Run("CodeWithScope", func(t *testing.T) {
			buf := []byte{
				0xf,
				// key
				0x66, 0x6f, 0x6f, 0x0,
				// value - code length
				0x1a, 0x0, 0x0, 0x0,
				// value - length
				0xd, 0x0, 0x0, 0x0,
				// value - code
				0x76, 0x61, 0x72, 0x20, 0x62, 0x61, 0x72, 0x20, 0x3d, 0x20, 0x78, 0x3b, 0x0,
			}
			scope := NewDocument(C.Null("x"))

			expected := &Element{&Value{start: 0, offset: 5, data: buf, d: scope}}
			actual := C.CodeWithScope("foo", "var bar = x;", scope)

			requireElementsEqual(t, expected, actual)
		})

		t.Run("int32", func(t *testing.T) {
			buf := []byte{
				// type
				0x10,
				// key
				0x66, 0x6f, 0x6f, 0x0,
				// value
				0xe5, 0xff, 0xff, 0xff,
			}

			expected := &Element{&Value{start: 0, offset: 5, data: buf, d: nil}}
			actual := C.Int32("foo", -27)

			requireElementsEqual(t, expected, actual)
		})

		t.Run("timestamp", func(t *testing.T) {
			buf := []byte{
				// type
				0x11,
				// key
				0x66, 0x6f, 0x6f, 0x0,
				// value
				0x11, 0x0, 0x0, 0x0, 0x8, 0x0, 0x0, 0x0,
			}

			expected := &Element{&Value{start: 0, offset: 5, data: buf, d: nil}}
			actual := C.Timestamp("foo", 8, 17)

			requireElementsEqual(t, expected, actual)
		})

		t.Run("int64Type", func(t *testing.T) {
			buf := []byte{
				// type
				0x12,
				// key
				0x66, 0x6f, 0x6f, 0x0,
				// value
				0xe5, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			}

			expected := &Element{&Value{start: 0, offset: 5, data: buf, d: nil}}
			actual := C.Int64("foo", -27)

			requireElementsEqual(t, expected, actual)
		})

		t.Run("Decimal128", func(t *testing.T) {
			buf := []byte{
				// type
				0x13,
				// key
				0x66, 0x6f, 0x6f, 0x0,
				// value
				0xee, 0x02, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x3c, 0xb0,
			}
			d, _ := decimal.ParseDecimal128("-7.50")

			expected := &Element{&Value{start: 0, offset: 5, data: buf, d: nil}}
			actual := C.Decimal128("foo", d)

			requireElementsEqual(t, expected, actual)
		})

		t.Run("minKey", func(t *testing.T) {
			buf := []byte{
				// type
				0xff,
				// key
				0x66, 0x6f, 0x6f, 0x0,
			}

			expected := &Element{&Value{start: 0, offset: 5, data: buf, d: nil}}
			actual := C.MinKey("foo")

			requireElementsEqual(t, expected, actual)
		})

		t.Run("maxKey", func(t *testing.T) {
			buf := []byte{
				// type
				0x7f,
				// key
				0x66, 0x6f, 0x6f, 0x0,
			}

			expected := &Element{&Value{start: 0, offset: 5, data: buf, d: nil}}
			actual := C.MaxKey("foo")

			requireElementsEqual(t, expected, actual)
		})
	})

	t.Run("Array", func(t *testing.T) {
		t.Run("double", func(t *testing.T) {
			buf := []byte{
				// type
				0x1,
				// key
				0x0,
				// value
				0x6e, 0x86, 0x1b, 0xf0, 0xf9,
				0x21, 0x9, 0x40,
			}

			expected := &Value{start: 0, offset: 2, data: buf, d: nil}
			actual := AC.Double(3.14159)

			requireValuesEqual(t, expected, actual)
		})

		t.Run("String", func(t *testing.T) {
			buf := []byte{
				// type
				0x2,
				// key
				0x0,
				// value - string length
				0x4, 0x0, 0x0, 0x0,
				// value - string
				0x62, 0x61, 0x72, 0x0,
			}

			expected := &Value{start: 0, offset: 2, data: buf, d: nil}
			actual := AC.String("bar")

			requireValuesEqual(t, expected, actual)
		})

		t.Run("SubDocument", func(t *testing.T) {
			buf := []byte{
				// type
				0x3,
				// key
				0x0,
			}
			d := NewDocument(C.String("bar", "baz"))

			expected := &Value{start: 0, offset: 2, data: buf, d: d}
			actual := AC.Document(d)

			requireValuesEqual(t, expected, actual)
		})

		t.Run("SubDocumentFromElements", func(t *testing.T) {
			buf := []byte{
				// type
				0x3,
				// key
				0x0,
			}
			e := C.String("bar", "baz")
			d := NewDocument(e)

			expected := &Value{start: 0, offset: 2, data: buf, d: d}
			actual := AC.DocumentFromElements(e)

			requireValuesEqual(t, expected, actual)
		})

		t.Run("SubDocumentFromReader", func(t *testing.T) {
			buf := []byte{
				// type
				0x3,
				// key
				0x0,
				0x10, 0x00, 0x00, 0x00,
				0x01, '0', 0x00,
				0x01, 0x02, 0x03, 0x04,
				0x05, 0x06, 0x07, 0x08,
				0x00,
			}
			rdr := Reader{
				0x10, 0x00, 0x00, 0x00,
				0x01, '0', 0x00,
				0x01, 0x02, 0x03, 0x04,
				0x05, 0x06, 0x07, 0x08,
				0x00,
			}

			expected := &Value{start: 0, offset: 2, data: buf}
			actual := AC.DocumentFromReader(rdr)

			requireValuesEqual(t, expected, actual)
		})

		t.Run("Array", func(t *testing.T) {
			buf := []byte{
				// type
				0x4,
				// key
				0x0,
			}
			a := NewArray(AC.String("bar"), AC.Double(-2.7))

			expected := &Value{start: 0, offset: 2, data: buf, d: a.doc}
			actual := AC.Array(a)

			requireValuesEqual(t, expected, actual)
		})

		t.Run("ArrayFromElements", func(t *testing.T) {
			buf := []byte{
				// type
				0x4,
				// key
				0x0,
			}
			e1 := AC.String("bar")
			e2 := AC.Double(-2.7)
			a := NewArray(e1, e2)

			expected := &Value{start: 0, offset: 2, data: buf, d: a.doc}
			actual := AC.ArrayFromValues(e1, e2)

			requireValuesEqual(t, expected, actual)
		})

		t.Run("binary", func(t *testing.T) {
			buf := []byte{
				// type
				0x5,
				// key
				0x0,
				// value - binary length
				0x7, 0x0, 0x0, 0x0,
				// value - binary subtype
				0x0,
				// value - binary data
				0x8, 0x6, 0x7, 0x5, 0x3, 0x0, 0x9,
			}

			expected := &Value{start: 0, offset: 2, data: buf, d: nil}
			actual := AC.Binary([]byte{8, 6, 7, 5, 3, 0, 9})

			requireValuesEqual(t, expected, actual)
		})

		t.Run("BinaryWithSubtype", func(t *testing.T) {
			buf := []byte{
				// type
				0x5,
				// key
				0x0,
				// value - binary length
				0xb, 0x0, 0x0, 0x0,
				// value - binary subtype
				0x2,
				//
				0x07, 0x00, 0x00, 0x00,
				// value - binary data
				0x8, 0x6, 0x7, 0x5, 0x3, 0x0, 0x9,
			}

			expected := &Value{start: 0, offset: 2, data: buf, d: nil}
			actual := AC.BinaryWithSubtype([]byte{8, 6, 7, 5, 3, 0, 9}, 2)

			requireValuesEqual(t, expected, actual)
		})

		t.Run("undefined", func(t *testing.T) {
			buf := []byte{
				// type
				0x6,
				// key
				0x0,
			}

			expected := &Value{start: 0, offset: 2, data: buf, d: nil}
			actual := AC.Undefined()

			requireValuesEqual(t, expected, actual)
		})

		t.Run("objectID", func(t *testing.T) {
			buf := []byte{
				// type
				0x7,
				// key
				0x0,
				// value
				0x5a, 0x15, 0xd0, 0xa4, 0xd5, 0xda, 0xa5, 0xf1, 0x0a, 0x5e, 0x10, 0x89,
			}

			expected := &Value{start: 0, offset: 2, data: buf, d: nil}
			actual := AC.ObjectID(

				[12]byte{0x5a, 0x15, 0xd0, 0xa4, 0xd5, 0xda, 0xa5, 0xf1, 0x0a, 0x5e, 0x10, 0x89},
			)

			requireValuesEqual(t, expected, actual)
		})

		t.Run("Boolean", func(t *testing.T) {
			buf := []byte{
				// type
				0x8,
				// key
				0x0,
				// value
				0x0,
			}

			expected := &Value{start: 0, offset: 2, data: buf, d: nil}
			actual := AC.Boolean(false)

			requireValuesEqual(t, expected, actual)
		})

		t.Run("dateTime", func(t *testing.T) {
			buf := []byte{
				// type
				0x9,
				// key
				0x0,
				// value
				0x11, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			}

			expected := &Value{start: 0, offset: 2, data: buf, d: nil}
			actual := AC.DateTime(17)

			requireValuesEqual(t, expected, actual)
		})

		t.Run("Null", func(t *testing.T) {
			buf := []byte{
				// type
				0xa,
				// key
				0x0,
			}

			expected := &Value{start: 0, offset: 2, data: buf, d: nil}
			actual := AC.Null()

			requireValuesEqual(t, expected, actual)
		})

		t.Run("regex", func(t *testing.T) {
			buf := []byte{
				// type
				0xb,
				// key
				0x0,
				// value - pattern
				0x62, 0x61, 0x72, 0x0,
				// value - options
				0x69, 0x0,
			}

			expected := &Value{start: 0, offset: 2, data: buf, d: nil}
			actual := AC.Regex("bar", "i")

			requireValuesEqual(t, expected, actual)
		})

		t.Run("dbPointer", func(t *testing.T) {
			buf := []byte{
				// type
				0xc,
				// key
				0x0,
				// value - namespace length
				0x4, 0x0, 0x0, 0x0,
				// value - namespace
				0x62, 0x61, 0x72, 0x0,
				// value - oid
				0x5a, 0x15, 0xd0, 0xa4, 0xd5, 0xda, 0xa5, 0xf1, 0x0a, 0x5e, 0x10, 0x89,
			}

			expected := &Value{start: 0, offset: 2, data: buf, d: nil}
			actual := AC.DBPointer(

				"bar",
				[12]byte{0x5a, 0x15, 0xd0, 0xa4, 0xd5, 0xda, 0xa5, 0xf1, 0x0a, 0x5e, 0x10, 0x89},
			)

			requireValuesEqual(t, expected, actual)
		})

		t.Run("JavaScriptCode", func(t *testing.T) {
			buf := []byte{
				// type
				0xd,
				// key
				0x0,
				// value - code length
				0xd, 0x0, 0x0, 0x0,
				// value - code
				0x76, 0x61, 0x72, 0x20, 0x62, 0x61, 0x72, 0x20, 0x3d, 0x20, 0x33, 0x3b, 0x0,
			}

			expected := &Value{start: 0, offset: 2, data: buf, d: nil}
			actual := AC.JavaScript("var bar = 3;")

			requireValuesEqual(t, expected, actual)
		})

		t.Run("symbol", func(t *testing.T) {
			buf := []byte{
				// type
				0xe,
				// key
				0x0,
				// value - string length
				0x4, 0x0, 0x0, 0x0,
				// value - string
				0x62, 0x61, 0x72, 0x0,
			}

			expected := &Value{start: 0, offset: 2, data: buf, d: nil}
			actual := AC.Symbol("bar")

			requireValuesEqual(t, expected, actual)
		})

		t.Run("CodeWithScope", func(t *testing.T) {
			buf := []byte{
				0xf,
				// key
				0x0,
				// value - code length
				0x17, 0x0, 0x0, 0x0,
				// value - length
				0xd, 0x0, 0x0, 0x0,
				// value - code
				0x76, 0x61, 0x72, 0x20, 0x62, 0x61, 0x72, 0x20, 0x3d, 0x20, 0x78, 0x3b, 0x0,
			}
			scope := NewDocument(C.Null("x"))

			expected := &Value{start: 0, offset: 2, data: buf, d: scope}
			actual := AC.CodeWithScope("var bar = x;", scope)

			requireValuesEqual(t, expected, actual)
		})

		t.Run("int32", func(t *testing.T) {
			buf := []byte{
				// type
				0x10,
				// key
				0x0,
				// value
				0xe5, 0xff, 0xff, 0xff,
			}

			expected := &Value{start: 0, offset: 2, data: buf, d: nil}
			actual := AC.Int32(-27)

			requireValuesEqual(t, expected, actual)
		})

		t.Run("timestamp", func(t *testing.T) {
			buf := []byte{
				// type
				0x11,
				// key
				0x0,
				// value
				0x11, 0x0, 0x0, 0x0, 0x8, 0x0, 0x0, 0x0,
			}

			expected := &Value{start: 0, offset: 2, data: buf, d: nil}
			actual := AC.Timestamp(8, 17)

			requireValuesEqual(t, expected, actual)
		})

		t.Run("int64Type", func(t *testing.T) {
			buf := []byte{
				// type
				0x12,
				// key
				0x0,
				// value
				0xe5, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			}

			expected := &Value{start: 0, offset: 2, data: buf, d: nil}
			actual := AC.Int64(-27)

			requireValuesEqual(t, expected, actual)
		})

		t.Run("Decimal128", func(t *testing.T) {
			buf := []byte{
				// type
				0x13,
				// key
				0x0,
				// value
				0xee, 0x02, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x3c, 0xb0,
			}
			d, _ := decimal.ParseDecimal128("-7.50")

			expected := &Value{start: 0, offset: 2, data: buf, d: nil}
			actual := AC.Decimal128(d)

			requireValuesEqual(t, expected, actual)
		})

		t.Run("minKey", func(t *testing.T) {
			buf := []byte{
				// type
				0xff,
				// key
				0x0,
			}

			expected := &Value{start: 0, offset: 2, data: buf, d: nil}
			actual := AC.MinKey()

			requireValuesEqual(t, expected, actual)
		})

		t.Run("maxKey", func(t *testing.T) {
			buf := []byte{
				// type
				0x7f,
				// key
				0x0,
			}

			expected := &Value{start: 0, offset: 2, data: buf, d: nil}
			actual := AC.MaxKey()

			requireValuesEqual(t, expected, actual)
		})
	})
}
