package bsoncodec

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/internal/llbson"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

func TestCopier(t *testing.T) {
	t.Run("CopyDocument", func(t *testing.T) {
		t.Run("ReadDocument Error", func(t *testing.T) {
			want := errors.New("ReadDocumentError")
			src := &llValueReaderWriter{t: t, err: want, errAfter: llvrwReadDocument}
			got := Copier{}.CopyDocument(nil, src)
			if !compareErrors(got, want) {
				t.Errorf("Did not receive correct error. got %v; want %v", got, want)
			}
		})
		t.Run("WriteDocument Error", func(t *testing.T) {
			want := errors.New("WriteDocumentError")
			src := &llValueReaderWriter{}
			dst := &llValueReaderWriter{t: t, err: want, errAfter: llvrwWriteDocument}
			got := Copier{}.CopyDocument(dst, src)
			if !compareErrors(got, want) {
				t.Errorf("Did not receive correct error. got %v; want %v", got, want)
			}
		})
		t.Run("success", func(t *testing.T) {
			doc := bson.NewDocument(bson.EC.String("Hello", "world"))
			src := newDocumentValueReader(doc)
			dst := newValueWriterFromSlice(make([]byte, 0))
			want, err := doc.MarshalBSON()
			noerr(t, err)
			err = Copier{}.CopyDocument(dst, src)
			noerr(t, err)
			got := dst.buf
			if !bytes.Equal(got, want) {
				t.Errorf("Bytes are not equal. got %v; want %v", bson.Reader(got), bson.Reader(want))
			}
		})
	})
	t.Run("copyArray", func(t *testing.T) {
		t.Run("ReadArray Error", func(t *testing.T) {
			want := errors.New("ReadArrayError")
			src := &llValueReaderWriter{t: t, err: want, errAfter: llvrwReadArray}
			got := Copier{}.copyArray(nil, src)
			if !compareErrors(got, want) {
				t.Errorf("Did not receive correct error. got %v; want %v", got, want)
			}
		})
		t.Run("WriteArray Error", func(t *testing.T) {
			want := errors.New("WriteArrayError")
			src := &llValueReaderWriter{}
			dst := &llValueReaderWriter{t: t, err: want, errAfter: llvrwWriteArray}
			got := Copier{}.copyArray(dst, src)
			if !compareErrors(got, want) {
				t.Errorf("Did not receive correct error. got %v; want %v", got, want)
			}
		})
		t.Run("success", func(t *testing.T) {
			doc := bson.NewDocument(bson.EC.ArrayFromElements("foo", bson.VC.String("Hello, world!")))
			src := newDocumentValueReader(doc)
			_, err := src.ReadDocument()
			noerr(t, err)
			_, _, err = src.ReadElement()
			noerr(t, err)

			dst := newValueWriterFromSlice(make([]byte, 0))
			_, err = dst.WriteDocument()
			noerr(t, err)
			_, err = dst.WriteDocumentElement("foo")
			noerr(t, err)
			want, err := doc.MarshalBSON()
			noerr(t, err)

			err = Copier{}.copyArray(dst, src)
			noerr(t, err)

			err = dst.WriteDocumentEnd()
			noerr(t, err)

			got := dst.buf
			if !bytes.Equal(got, want) {
				t.Errorf("Bytes are not equal. got %v; want %v", bson.Reader(got), bson.Reader(want))
			}
		})
	})
	t.Run("CopyValue", func(t *testing.T) {
		testCases := []struct {
			name string
			dst  *llValueReaderWriter
			src  *llValueReaderWriter
			err  error
		}{
			{
				"Double/src/error",
				&llValueReaderWriter{},
				&llValueReaderWriter{bsontype: bson.TypeDouble, err: errors.New("1"), errAfter: llvrwReadDouble},
				errors.New("1"),
			},
			{
				"Double/dst/error",
				&llValueReaderWriter{bsontype: bson.TypeDouble, err: errors.New("2"), errAfter: llvrwWriteDouble},
				&llValueReaderWriter{bsontype: bson.TypeDouble, readval: float64(3.14159)},
				errors.New("2"),
			},
			{
				"String/src/error",
				&llValueReaderWriter{},
				&llValueReaderWriter{bsontype: bson.TypeString, err: errors.New("1"), errAfter: llvrwReadString},
				errors.New("1"),
			},
			{
				"String/dst/error",
				&llValueReaderWriter{bsontype: bson.TypeString, err: errors.New("2"), errAfter: llvrwWriteString},
				&llValueReaderWriter{bsontype: bson.TypeString, readval: string("hello, world")},
				errors.New("2"),
			},
			{
				"Document/src/error",
				&llValueReaderWriter{},
				&llValueReaderWriter{bsontype: bson.TypeEmbeddedDocument, err: errors.New("1"), errAfter: llvrwReadDocument},
				errors.New("1"),
			},
			{
				"Array/dst/error",
				&llValueReaderWriter{},
				&llValueReaderWriter{bsontype: bson.TypeArray, err: errors.New("2"), errAfter: llvrwReadArray},
				errors.New("2"),
			},
			{
				"Binary/src/error",
				&llValueReaderWriter{},
				&llValueReaderWriter{bsontype: bson.TypeBinary, err: errors.New("1"), errAfter: llvrwReadBinary},
				errors.New("1"),
			},
			{
				"Binary/dst/error",
				&llValueReaderWriter{bsontype: bson.TypeBinary, err: errors.New("2"), errAfter: llvrwWriteBinaryWithSubtype},
				&llValueReaderWriter{bsontype: bson.TypeBinary, readval: bson.Binary{Subtype: 0xFF, Data: []byte{0x01, 0x02, 0x03}}},
				errors.New("2"),
			},
			{
				"Undefined/src/error",
				&llValueReaderWriter{},
				&llValueReaderWriter{bsontype: bson.TypeUndefined, err: errors.New("1"), errAfter: llvrwReadUndefined},
				errors.New("1"),
			},
			{
				"Undefined/dst/error",
				&llValueReaderWriter{bsontype: bson.TypeUndefined, err: errors.New("2"), errAfter: llvrwWriteUndefined},
				&llValueReaderWriter{bsontype: bson.TypeUndefined},
				errors.New("2"),
			},
			{
				"ObjectID/src/error",
				&llValueReaderWriter{},
				&llValueReaderWriter{bsontype: bson.TypeObjectID, err: errors.New("1"), errAfter: llvrwReadObjectID},
				errors.New("1"),
			},
			{
				"ObjectID/dst/error",
				&llValueReaderWriter{bsontype: bson.TypeObjectID, err: errors.New("2"), errAfter: llvrwWriteObjectID},
				&llValueReaderWriter{bsontype: bson.TypeObjectID, readval: objectid.ObjectID{0x01, 0x02, 0x03}},
				errors.New("2"),
			},
			{
				"Boolean/src/error",
				&llValueReaderWriter{},
				&llValueReaderWriter{bsontype: bson.TypeBoolean, err: errors.New("1"), errAfter: llvrwReadBoolean},
				errors.New("1"),
			},
			{
				"Boolean/dst/error",
				&llValueReaderWriter{bsontype: bson.TypeBoolean, err: errors.New("2"), errAfter: llvrwWriteBoolean},
				&llValueReaderWriter{bsontype: bson.TypeBoolean, readval: bool(true)},
				errors.New("2"),
			},
			{
				"DateTime/src/error",
				&llValueReaderWriter{},
				&llValueReaderWriter{bsontype: bson.TypeDateTime, err: errors.New("1"), errAfter: llvrwReadDateTime},
				errors.New("1"),
			},
			{
				"DateTime/dst/error",
				&llValueReaderWriter{bsontype: bson.TypeDateTime, err: errors.New("2"), errAfter: llvrwWriteDateTime},
				&llValueReaderWriter{bsontype: bson.TypeDateTime, readval: int64(1234567890)},
				errors.New("2"),
			},
			{
				"Null/src/error",
				&llValueReaderWriter{},
				&llValueReaderWriter{bsontype: bson.TypeNull, err: errors.New("1"), errAfter: llvrwReadNull},
				errors.New("1"),
			},
			{
				"Null/dst/error",
				&llValueReaderWriter{bsontype: bson.TypeNull, err: errors.New("2"), errAfter: llvrwWriteNull},
				&llValueReaderWriter{bsontype: bson.TypeNull},
				errors.New("2"),
			},
			{
				"Regex/src/error",
				&llValueReaderWriter{},
				&llValueReaderWriter{bsontype: bson.TypeRegex, err: errors.New("1"), errAfter: llvrwReadRegex},
				errors.New("1"),
			},
			{
				"Regex/dst/error",
				&llValueReaderWriter{bsontype: bson.TypeRegex, err: errors.New("2"), errAfter: llvrwWriteRegex},
				&llValueReaderWriter{bsontype: bson.TypeRegex, readval: bson.Regex{Pattern: "hello", Options: "world"}},
				errors.New("2"),
			},
			{
				"DBPointer/src/error",
				&llValueReaderWriter{},
				&llValueReaderWriter{bsontype: bson.TypeDBPointer, err: errors.New("1"), errAfter: llvrwReadDBPointer},
				errors.New("1"),
			},
			{
				"DBPointer/dst/error",
				&llValueReaderWriter{bsontype: bson.TypeDBPointer, err: errors.New("2"), errAfter: llvrwWriteDBPointer},
				&llValueReaderWriter{bsontype: bson.TypeDBPointer, readval: bson.DBPointer{DB: "foo", Pointer: objectid.ObjectID{0x01, 0x02, 0x03}}},
				errors.New("2"),
			},
			{
				"Javascript/src/error",
				&llValueReaderWriter{},
				&llValueReaderWriter{bsontype: bson.TypeJavaScript, err: errors.New("1"), errAfter: llvrwReadJavascript},
				errors.New("1"),
			},
			{
				"Javascript/dst/error",
				&llValueReaderWriter{bsontype: bson.TypeJavaScript, err: errors.New("2"), errAfter: llvrwWriteJavascript},
				&llValueReaderWriter{bsontype: bson.TypeJavaScript, readval: string("hello, world")},
				errors.New("2"),
			},
			{
				"Symbol/src/error",
				&llValueReaderWriter{},
				&llValueReaderWriter{bsontype: bson.TypeSymbol, err: errors.New("1"), errAfter: llvrwReadSymbol},
				errors.New("1"),
			},
			{
				"Symbol/dst/error",
				&llValueReaderWriter{bsontype: bson.TypeSymbol, err: errors.New("2"), errAfter: llvrwWriteSymbol},
				&llValueReaderWriter{bsontype: bson.TypeSymbol, readval: bson.Symbol("hello, world")},
				errors.New("2"),
			},
			{
				"CodeWithScope/src/error",
				&llValueReaderWriter{},
				&llValueReaderWriter{bsontype: bson.TypeCodeWithScope, err: errors.New("1"), errAfter: llvrwReadCodeWithScope},
				errors.New("1"),
			},
			{
				"CodeWithScope/dst/error",
				&llValueReaderWriter{bsontype: bson.TypeCodeWithScope, err: errors.New("2"), errAfter: llvrwWriteCodeWithScope},
				&llValueReaderWriter{bsontype: bson.TypeCodeWithScope},
				errors.New("2"),
			},
			{
				"CodeWithScope/dst/copyDocumentCore error",
				&llValueReaderWriter{err: errors.New("3"), errAfter: llvrwWriteDocumentElement},
				&llValueReaderWriter{bsontype: bson.TypeCodeWithScope},
				errors.New("3"),
			},
			{
				"Int32/src/error",
				&llValueReaderWriter{},
				&llValueReaderWriter{bsontype: bson.TypeInt32, err: errors.New("1"), errAfter: llvrwReadInt32},
				errors.New("1"),
			},
			{
				"Int32/dst/error",
				&llValueReaderWriter{bsontype: bson.TypeInt32, err: errors.New("2"), errAfter: llvrwWriteInt32},
				&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(12345)},
				errors.New("2"),
			},
			{
				"Timestamp/src/error",
				&llValueReaderWriter{},
				&llValueReaderWriter{bsontype: bson.TypeTimestamp, err: errors.New("1"), errAfter: llvrwReadTimestamp},
				errors.New("1"),
			},
			{
				"Timestamp/dst/error",
				&llValueReaderWriter{bsontype: bson.TypeTimestamp, err: errors.New("2"), errAfter: llvrwWriteTimestamp},
				&llValueReaderWriter{bsontype: bson.TypeTimestamp, readval: bson.Timestamp{T: 12345, I: 67890}},
				errors.New("2"),
			},
			{
				"Int64/src/error",
				&llValueReaderWriter{},
				&llValueReaderWriter{bsontype: bson.TypeInt64, err: errors.New("1"), errAfter: llvrwReadInt64},
				errors.New("1"),
			},
			{
				"Int64/dst/error",
				&llValueReaderWriter{bsontype: bson.TypeInt64, err: errors.New("2"), errAfter: llvrwWriteInt64},
				&llValueReaderWriter{bsontype: bson.TypeInt64, readval: int64(1234567890)},
				errors.New("2"),
			},
			{
				"Decimal128/src/error",
				&llValueReaderWriter{},
				&llValueReaderWriter{bsontype: bson.TypeDecimal128, err: errors.New("1"), errAfter: llvrwReadDecimal128},
				errors.New("1"),
			},
			{
				"Decimal128/dst/error",
				&llValueReaderWriter{bsontype: bson.TypeDecimal128, err: errors.New("2"), errAfter: llvrwWriteDecimal128},
				&llValueReaderWriter{bsontype: bson.TypeDecimal128, readval: decimal.NewDecimal128(12345, 67890)},
				errors.New("2"),
			},
			{
				"MinKey/src/error",
				&llValueReaderWriter{},
				&llValueReaderWriter{bsontype: bson.TypeMinKey, err: errors.New("1"), errAfter: llvrwReadMinKey},
				errors.New("1"),
			},
			{
				"MinKey/dst/error",
				&llValueReaderWriter{bsontype: bson.TypeMinKey, err: errors.New("2"), errAfter: llvrwWriteMinKey},
				&llValueReaderWriter{bsontype: bson.TypeMinKey},
				errors.New("2"),
			},
			{
				"MaxKey/src/error",
				&llValueReaderWriter{},
				&llValueReaderWriter{bsontype: bson.TypeMaxKey, err: errors.New("1"), errAfter: llvrwReadMaxKey},
				errors.New("1"),
			},
			{
				"MaxKey/dst/error",
				&llValueReaderWriter{bsontype: bson.TypeMaxKey, err: errors.New("2"), errAfter: llvrwWriteMaxKey},
				&llValueReaderWriter{bsontype: bson.TypeMaxKey},
				errors.New("2"),
			},
			{
				"Unknown BSON type error",
				&llValueReaderWriter{},
				&llValueReaderWriter{},
				fmt.Errorf("Cannot copy unknown BSON type %s", bson.Type(0)),
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				tc.dst.t, tc.src.t = t, t
				err := Copier{}.CopyValue(tc.dst, tc.src)
				if !compareErrors(err, tc.err) {
					t.Errorf("Did not receive expected error. got %v; want %v", err, tc.err)
				}
			})
		}
	})
	t.Run("CopyValueFromBytes", func(t *testing.T) {
		t.Run("BytesWriter", func(t *testing.T) {
			vw := newValueWriterFromSlice(make([]byte, 0))
			_, err := vw.WriteDocument()
			noerr(t, err)
			_, err = vw.WriteDocumentElement("foo")
			noerr(t, err)
			err = Copier{}.CopyValueFromBytes(vw, bson.TypeString, llbson.AppendString(nil, "bar"))
			noerr(t, err)
			err = vw.WriteDocumentEnd()
			noerr(t, err)
			want, err := bson.NewDocument(bson.EC.String("foo", "bar")).MarshalBSON()
			noerr(t, err)
			got := vw.buf
			if !bytes.Equal(got, want) {
				t.Errorf("Bytes are not equal. got %v; want %v", bson.Reader(got), bson.Reader(want))
			}
		})
		t.Run("Non BytesWriter", func(t *testing.T) {
			llvrw := &llValueReaderWriter{t: t}
			err := Copier{}.CopyValueFromBytes(llvrw, bson.TypeString, llbson.AppendString(nil, "bar"))
			noerr(t, err)
			got, want := llvrw.invoked, llvrwWriteString
			if got != want {
				t.Errorf("Incorrect method invoked on llvrw. got %v; want %v", got, want)
			}
		})
	})
	t.Run("CopyValueToBytes", func(t *testing.T) {
		t.Run("BytesReader", func(t *testing.T) {
			b, err := bson.NewDocument(bson.EC.String("hello", "world")).MarshalBSON()
			noerr(t, err)
			vr := newValueReader(b)
			_, err = vr.ReadDocument()
			noerr(t, err)
			_, _, err = vr.ReadElement()
			noerr(t, err)
			bsontype, got, err := Copier{}.CopyValueToBytes(vr)
			noerr(t, err)
			want := llbson.AppendString(nil, "world")
			if bsontype != bson.TypeString {
				t.Errorf("Incorrect type returned. got %v; want %v", bsontype, bson.TypeString)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("Bytes do not match. got %v; want %v", got, want)
			}
		})
		t.Run("Non BytesReader", func(t *testing.T) {
			llvrw := &llValueReaderWriter{t: t, bsontype: bson.TypeString, readval: string("Hello, world!")}
			bsontype, got, err := Copier{}.CopyValueToBytes(llvrw)
			noerr(t, err)
			want := llbson.AppendString(nil, "Hello, world!")
			if bsontype != bson.TypeString {
				t.Errorf("Incorrect type returned. got %v; want %v", bsontype, bson.TypeString)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("Bytes do not match. got %v; want %v", got, want)
			}
		})
	})
	t.Run("AppendValueBytes", func(t *testing.T) {
		t.Run("BytesReader", func(t *testing.T) {
			b, err := bson.NewDocument(bson.EC.String("hello", "world")).MarshalBSON()
			noerr(t, err)
			vr := newValueReader(b)
			_, err = vr.ReadDocument()
			noerr(t, err)
			_, _, err = vr.ReadElement()
			noerr(t, err)
			bsontype, got, err := Copier{}.AppendValueBytes(nil, vr)
			noerr(t, err)
			want := llbson.AppendString(nil, "world")
			if bsontype != bson.TypeString {
				t.Errorf("Incorrect type returned. got %v; want %v", bsontype, bson.TypeString)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("Bytes do not match. got %v; want %v", got, want)
			}
		})
		t.Run("Non BytesReader", func(t *testing.T) {
			llvrw := &llValueReaderWriter{t: t, bsontype: bson.TypeString, readval: string("Hello, world!")}
			bsontype, got, err := Copier{}.AppendValueBytes(nil, llvrw)
			noerr(t, err)
			want := llbson.AppendString(nil, "Hello, world!")
			if bsontype != bson.TypeString {
				t.Errorf("Incorrect type returned. got %v; want %v", bsontype, bson.TypeString)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("Bytes do not match. got %v; want %v", got, want)
			}
		})
		t.Run("CopyValue error", func(t *testing.T) {
			want := errors.New("CopyValue error")
			llvrw := &llValueReaderWriter{t: t, bsontype: bson.TypeString, err: want, errAfter: llvrwReadString}
			_, _, got := Copier{}.AppendValueBytes(make([]byte, 0), llvrw)
			if !compareErrors(got, want) {
				t.Errorf("Errors do not match. got %v; want %v", got, want)
			}
		})
	})
}
