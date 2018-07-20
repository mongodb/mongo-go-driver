package bson

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

func TestValueReader(t *testing.T) {
	t.Run("ReadBinary", func(t *testing.T) {
		testCases := []struct {
			name   string
			data   []byte
			offset int64
			btype  byte
			b      []byte
			err    error
			vType  Type
		}{
			{
				"incorrect type",
				[]byte{},
				0,
				0,
				nil,
				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeBinary),
				TypeEmbeddedDocument,
			},
			{
				"length too short",
				[]byte{},
				0,
				0,
				nil,
				io.EOF,
				TypeBinary,
			},
			{
				"no byte available",
				[]byte{0x00, 0x00, 0x00, 0x00},
				0,
				0,
				nil,
				io.EOF,
				TypeBinary,
			},
			{
				"not enough bytes for binary",
				[]byte{0x05, 0x00, 0x00, 0x00, 0x00},
				0,
				0,
				nil,
				io.EOF,
				TypeBinary,
			},
			{
				"success",
				[]byte{0x03, 0x00, 0x00, 0x00, 0xEA, 0x01, 0x02, 0x03},
				0,
				0xEA,
				[]byte{0x01, 0x02, 0x03},
				nil,
				TypeBinary,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				vr := &valueReader{
					offset: tc.offset,
					d:      tc.data,
					stack: []vrState{
						{mode: vrTopLevel},
						{
							mode:   vrElement,
							vType:  tc.vType,
							offset: tc.offset,
						},
					},
					frame: 1,
				}

				b, btype, err := vr.ReadBinary()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if btype != tc.btype {
					t.Errorf("Incorrect binary type returned. got %v; want %v", btype, tc.btype)
				}
				if !bytes.Equal(b, tc.b) {
					t.Errorf("Binary data does not match. got %v; want %v", b, tc.b)
				}
			})
		}
	})
	t.Run("ReadBoolean", func(t *testing.T) {
		testCases := []struct {
			name    string
			data    []byte
			offset  int64
			boolean bool
			err     error
			vType   Type
		}{
			{
				"incorrect type",
				[]byte{},
				0,
				false,
				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeBoolean),
				TypeEmbeddedDocument,
			},
			{
				"no byte available",
				[]byte{},
				0,
				false,
				io.EOF,
				TypeBoolean,
			},
			{
				"invalid byte for boolean",
				[]byte{0x03},
				0,
				false,
				fmt.Errorf("invalid byte for boolean, %b", 0x03),
				TypeBoolean,
			},
			{
				"success",
				[]byte{0x01},
				0,
				true,
				nil,
				TypeBoolean,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				vr := &valueReader{
					offset: tc.offset,
					d:      tc.data,
					stack: []vrState{
						{mode: vrTopLevel},
						{
							mode:   vrElement,
							vType:  tc.vType,
							offset: tc.offset,
						},
					},
					frame: 1,
				}

				boolean, err := vr.ReadBoolean()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if boolean != tc.boolean {
					t.Errorf("Incorrect boolean returned. got %v; want %v", boolean, tc.boolean)
				}
			})
		}
	})
	t.Run("ReadDocument", func(t *testing.T) {
		t.Run("TopLevel", func(t *testing.T) {
			doc := []byte{0x05, 0x00, 0x00, 0x00, 0x00}
			vr := &valueReader{
				offset: 0,
				stack:  []vrState{{mode: vrTopLevel}},
				frame:  0,
			}

			// invalid length
			vr.d = []byte{0x00, 0x00}
			_, err := vr.ReadDocument()
			if err != io.EOF {
				t.Errorf("Expected io.EOF with document length too small. got %v; want %v", err, io.EOF)
			}

			vr.d = doc
			_, err = vr.ReadDocument()
			noerr(t, err)
			if vr.size != 5 {
				t.Errorf("Incorrect size for document. got %d; want %d", vr.size, 5)
			}
			if vr.stack[vr.frame].size != 5 {
				t.Errorf("Incorrect size for document. got %d; want %d", vr.stack[vr.frame].size, 5)
			}
		})
		t.Run("EmbeddedDocument", func(t *testing.T) {
			vr := &valueReader{
				offset: 0,
				stack: []vrState{
					{mode: vrTopLevel},
					{mode: vrElement, vType: TypeBoolean},
				},
				frame: 1,
			}

			var wanterr error = (&valueReader{stack: []vrState{{mode: vrElement, vType: TypeBoolean}}}).typeError(TypeEmbeddedDocument)
			_, err := vr.ReadDocument()
			if err == nil || err.Error() != wanterr.Error() {
				t.Errorf("Incorrect returned error. got %v; want %v", err, wanterr)
			}

			vr.stack[1].mode = vrArray
			wanterr = vr.invalidTransitionErr(vrDocument)
			_, err = vr.ReadDocument()
			if err == nil || err.Error() != wanterr.Error() {
				t.Errorf("Incorrect returned error. got %v; want %v", err, wanterr)
			}

			vr.stack[1].mode, vr.stack[1].vType = vrElement, TypeEmbeddedDocument
			vr.d = []byte{0x0A, 0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00}
			vr.offset, vr.size = 4, 10
			_, err = vr.ReadDocument()
			noerr(t, err)
			if len(vr.stack) != 3 {
				t.Errorf("Incorrect number of stack frames. got %d; want %d", len(vr.stack), 3)
			}
			if vr.stack[2].mode != vrDocument {
				t.Errorf("Incorrect mode set. got %v; want %v", vr.stack[2].mode, vrDocument)
			}
			if vr.stack[2].offset != 4 {
				t.Errorf("Offset not set corerctly. got %d; want %d", vr.stack[2].offset, 4)
			}
			if vr.stack[2].size != 5 {
				t.Errorf("Size of embedded document is not correct. got %d; want %d", vr.stack[2].size, 5)
			}
			if vr.offset != 8 {
				t.Errorf("Offset not incremented correctly. got %d; want %d", vr.offset, 8)
			}

			vr.frame--
			_, err = vr.ReadDocument()
			if err != io.EOF {
				t.Errorf("Should return error when attempting to read length with not enough bytes. got %v; want %v", err, io.EOF)
			}
		})
	})
	t.Run("ReadBinary", func(t *testing.T) {
		codeWithScope := []byte{
			0x11, 0x00, 0x00, 0x00, // total length
			0x4, 0x00, 0x00, 0x00, // string length
			'f', 'o', 'o', 0x00, // string
			0x05, 0x00, 0x00, 0x00, 0x00, // document
		}
		mismatchCodeWithScope := []byte{
			0x11, 0x00, 0x00, 0x00, // total length
			0x4, 0x00, 0x00, 0x00, // string length
			'f', 'o', 'o', 0x00, // string
			0x07, 0x00, 0x00, 0x00, // document
			0x0A, 0x00, // null element, empty key
			0x00, // document end
		}
		testCases := []struct {
			name   string
			data   []byte
			offset int64
			err    error
			vType  Type
		}{
			{
				"incorrect type",
				[]byte{},
				0,
				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeCodeWithScope),
				TypeEmbeddedDocument,
			},
			{
				"total length not enough bytes",
				[]byte{},
				0,
				io.EOF,
				TypeCodeWithScope,
			},
			{
				"string length not enough bytes",
				codeWithScope[:4],
				0,
				io.EOF,
				TypeCodeWithScope,
			},
			{
				"not enough string bytes",
				codeWithScope[:8],
				0,
				io.EOF,
				TypeCodeWithScope,
			},
			{
				"document length not enough bytes",
				codeWithScope[:12],
				0,
				io.EOF,
				TypeCodeWithScope,
			},
			{
				"length mismatch",
				mismatchCodeWithScope,
				0,
				fmt.Errorf("length of CodeWithScope does not match lengths of components; total: %d; components: %d", 17, 19),
				TypeCodeWithScope,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				vr := &valueReader{
					offset: tc.offset,
					d:      tc.data,
					stack: []vrState{
						{mode: vrTopLevel},
						{
							mode:   vrElement,
							vType:  tc.vType,
							offset: tc.offset,
						},
					},
					frame: 1,
				}

				_, _, err := vr.ReadCodeWithScope()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
			})
		}

		t.Run("success", func(t *testing.T) {
			doc := []byte{0x00, 0x00, 0x00, 0x00}
			doc = append(doc, codeWithScope...)
			doc = append(doc, 0x00)
			vr := &valueReader{
				offset: 4,
				d:      doc,
				stack: []vrState{
					{mode: vrTopLevel, offset: 4},
					{mode: vrElement, vType: TypeCodeWithScope, offset: 4},
				},
				frame: 1,
			}

			code, _, err := vr.ReadCodeWithScope()
			noerr(t, err)
			if code != "foo" {
				t.Errorf("Code does not match. got %s; want %s", code, "foo")
			}
			if len(vr.stack) != 3 {
				t.Errorf("Incorrect number of stack frames. got %d; want %d", len(vr.stack), 3)
			}
			if vr.stack[2].mode != vrCodeWithScope {
				t.Errorf("Incorrect mode set. got %v; want %v", vr.stack[2].mode, vrDocument)
			}
			if vr.stack[2].offset != 4 {
				t.Errorf("Offset not set corerctly. got %d; want %d", vr.stack[2].offset, 4)
			}
			if vr.stack[2].size != 5 {
				t.Errorf("Size of scope is not correct. got %d; want %d", vr.stack[2].size, 5)
			}
			if vr.offset != 20 {
				t.Errorf("Offset not incremented correctly. got %d; want %d", vr.offset, 20)
			}
		})
	})
	t.Run("ReadDBPointer", func(t *testing.T) {
		testCases := []struct {
			name   string
			data   []byte
			offset int64
			ns     string
			oid    objectid.ObjectID
			err    error
			vType  Type
		}{
			{
				"incorrect type",
				[]byte{},
				0,
				"",
				objectid.ObjectID{},
				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeDBPointer),
				TypeEmbeddedDocument,
			},
			{
				"length too short",
				[]byte{},
				0,
				"",
				objectid.ObjectID{},
				io.EOF,
				TypeDBPointer,
			},
			{
				"not enough bytes for namespace",
				[]byte{0x04, 0x00, 0x00, 0x00},
				0,
				"",
				objectid.ObjectID{},
				io.EOF,
				TypeDBPointer,
			},
			{
				"not enough bytes for objectID",
				[]byte{0x04, 0x00, 0x00, 0x00, 'f', 'o', 'o', 0x00},
				0,
				"",
				objectid.ObjectID{},
				io.EOF,
				TypeDBPointer,
			},
			{
				"success",
				[]byte{
					0x04, 0x00, 0x00, 0x00, 'f', 'o', 'o', 0x00,
					0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C,
				},
				0,
				"foo",
				objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
				nil,
				TypeDBPointer,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				vr := &valueReader{
					offset: tc.offset,
					d:      tc.data,
					stack: []vrState{
						{mode: vrTopLevel},
						{
							mode:   vrElement,
							vType:  tc.vType,
							offset: tc.offset,
						},
					},
					frame: 1,
				}

				ns, oid, err := vr.ReadDBPointer()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if ns != tc.ns {
					t.Errorf("Incorrect namespace returned. got %v; want %v", ns, tc.ns)
				}
				if !bytes.Equal(oid[:], tc.oid[:]) {
					t.Errorf("ObjectIDs did not match. got %v; want %v", oid, tc.oid)
				}
			})
		}
	})
	t.Run("ReadDateTime", func(t *testing.T) {
		testCases := []struct {
			name   string
			data   []byte
			offset int64
			dt     int64
			err    error
			vType  Type
		}{
			{
				"incorrect type",
				[]byte{},
				0,
				0,
				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeDateTime),
				TypeEmbeddedDocument,
			},
			{
				"length too short",
				[]byte{},
				0,
				0,
				io.EOF,
				TypeDateTime,
			},
			{
				"success",
				[]byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
				0,
				255,
				nil,
				TypeDateTime,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				vr := &valueReader{
					offset: tc.offset,
					d:      tc.data,
					stack: []vrState{
						{mode: vrTopLevel},
						{
							mode:   vrElement,
							vType:  tc.vType,
							offset: tc.offset,
						},
					},
					frame: 1,
				}

				dt, err := vr.ReadDateTime()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if dt != tc.dt {
					t.Errorf("Incorrect datetime returned. got %d; want %d", dt, tc.dt)
				}
			})
		}
	})
	t.Run("ReadDecimal128", func(t *testing.T) {
		testCases := []struct {
			name   string
			data   []byte
			offset int64
			dc128  decimal.Decimal128
			err    error
			vType  Type
		}{
			{
				"incorrect type",
				[]byte{},
				0,
				decimal.Decimal128{},
				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeDecimal128),
				TypeEmbeddedDocument,
			},
			{
				"length too short",
				[]byte{},
				0,
				decimal.Decimal128{},
				io.EOF,
				TypeDecimal128,
			},
			{
				"success",
				[]byte{
					0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Low
					0x00, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // High
				},
				0,
				decimal.NewDecimal128(65280, 255),
				nil,
				TypeDecimal128,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				vr := &valueReader{
					offset: tc.offset,
					d:      tc.data,
					stack: []vrState{
						{mode: vrTopLevel},
						{
							mode:   vrElement,
							vType:  tc.vType,
							offset: tc.offset,
						},
					},
					frame: 1,
				}

				dc128, err := vr.ReadDecimal128()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				gotHigh, gotLow := dc128.GetBytes()
				wantHigh, wantLow := tc.dc128.GetBytes()
				if gotHigh != wantHigh {
					t.Errorf("Retuired high byte does not match. got %d; want %d", gotHigh, wantHigh)
				}
				if gotLow != wantLow {
					t.Errorf("Returned low byte does not match. got %d; want %d", gotLow, wantLow)
				}
			})
		}
	})
	t.Run("ReadDouble", func(t *testing.T) {
		testCases := []struct {
			name   string
			data   []byte
			offset int64
			double float64
			err    error
			vType  Type
		}{
			{
				"incorrect type",
				[]byte{},
				0,
				0,
				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeDouble),
				TypeEmbeddedDocument,
			},
			{
				"length too short",
				[]byte{},
				0,
				0,
				io.EOF,
				TypeDouble,
			},
			{
				"success",
				[]byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
				0,
				math.Float64frombits(255),
				nil,
				TypeDouble,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				vr := &valueReader{
					offset: tc.offset,
					d:      tc.data,
					stack: []vrState{
						{mode: vrTopLevel},
						{
							mode:   vrElement,
							vType:  tc.vType,
							offset: tc.offset,
						},
					},
					frame: 1,
				}

				double, err := vr.ReadDouble()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if double != tc.double {
					t.Errorf("Incorrect double returned. got %f; want %f", double, tc.double)
				}
			})
		}
	})
	t.Run("ReadInt32", func(t *testing.T) {
		testCases := []struct {
			name   string
			data   []byte
			offset int64
			i32    int32
			err    error
			vType  Type
		}{
			{
				"incorrect type",
				[]byte{},
				0,
				0,
				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeInt32),
				TypeEmbeddedDocument,
			},
			{
				"length too short",
				[]byte{},
				0,
				0,
				io.EOF,
				TypeInt32,
			},
			{
				"success",
				[]byte{0xFF, 0x00, 0x00, 0x00},
				0,
				255,
				nil,
				TypeInt32,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				vr := &valueReader{
					offset: tc.offset,
					d:      tc.data,
					stack: []vrState{
						{mode: vrTopLevel},
						{
							mode:   vrElement,
							vType:  tc.vType,
							offset: tc.offset,
						},
					},
					frame: 1,
				}

				i32, err := vr.ReadInt32()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if i32 != tc.i32 {
					t.Errorf("Incorrect int32 returned. got %d; want %d", i32, tc.i32)
				}
			})
		}
	})
	t.Run("ReadInt32", func(t *testing.T) {
		testCases := []struct {
			name   string
			data   []byte
			offset int64
			i64    int64
			err    error
			vType  Type
		}{
			{
				"incorrect type",
				[]byte{},
				0,
				0,
				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeInt64),
				TypeEmbeddedDocument,
			},
			{
				"length too short",
				[]byte{},
				0,
				0,
				io.EOF,
				TypeInt64,
			},
			{
				"success",
				[]byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
				0,
				255,
				nil,
				TypeInt64,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				vr := &valueReader{
					offset: tc.offset,
					d:      tc.data,
					stack: []vrState{
						{mode: vrTopLevel},
						{
							mode:   vrElement,
							vType:  tc.vType,
							offset: tc.offset,
						},
					},
					frame: 1,
				}

				i64, err := vr.ReadInt64()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if i64 != tc.i64 {
					t.Errorf("Incorrect int64 returned. got %d; want %d", i64, tc.i64)
				}
			})
		}
	})
	t.Run("ReadJavascript/ReadString/ReadSymbol", func(t *testing.T) {
		testCases := []struct {
			name   string
			data   []byte
			offset int64
			fn     func(*valueReader) (string, error)
			css    string // code, string, symbol :P
			err    error
			vType  Type
		}{
			{
				"ReadJavascript/incorrect type",
				[]byte{},
				0,
				(*valueReader).ReadJavascript,
				"",
				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeJavaScript),
				TypeEmbeddedDocument,
			},
			{
				"ReadString/incorrect type",
				[]byte{},
				0,
				(*valueReader).ReadString,
				"",
				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeString),
				TypeEmbeddedDocument,
			},
			{
				"ReadSymbol/incorrect type",
				[]byte{},
				0,
				(*valueReader).ReadSymbol,
				"",
				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeSymbol),
				TypeEmbeddedDocument,
			},
			{
				"ReadJavascript/length too short",
				[]byte{},
				0,
				(*valueReader).ReadJavascript,
				"",
				io.EOF,
				TypeJavaScript,
			},
			{
				"ReadString/length too short",
				[]byte{},
				0,
				(*valueReader).ReadString,
				"",
				io.EOF,
				TypeString,
			},
			{
				"ReadSymbol/length too short",
				[]byte{},
				0,
				(*valueReader).ReadSymbol,
				"",
				io.EOF,
				TypeSymbol,
			},
			{
				"ReadJavascript/incorrect end byte",
				[]byte{0x01, 0x00, 0x00, 0x00, 0x05},
				0,
				(*valueReader).ReadJavascript,
				"",
				fmt.Errorf("string does not end with null byte, but with %v", 0x05),
				TypeJavaScript,
			},
			{
				"ReadString/incorrect end byte",
				[]byte{0x01, 0x00, 0x00, 0x00, 0x05},
				0,
				(*valueReader).ReadString,
				"",
				fmt.Errorf("string does not end with null byte, but with %v", 0x05),
				TypeString,
			},
			{
				"ReadSymbol/incorrect end byte",
				[]byte{0x01, 0x00, 0x00, 0x00, 0x05},
				0,
				(*valueReader).ReadSymbol,
				"",
				fmt.Errorf("string does not end with null byte, but with %v", 0x05),
				TypeSymbol,
			},
			{
				"ReadJavascript/success",
				[]byte{0x04, 0x00, 0x00, 0x00, 'f', 'o', 'o', 0x00},
				0,
				(*valueReader).ReadJavascript,
				"foo",
				nil,
				TypeJavaScript,
			},
			{
				"ReadString/success",
				[]byte{0x04, 0x00, 0x00, 0x00, 'f', 'o', 'o', 0x00},
				0,
				(*valueReader).ReadString,
				"foo",
				nil,
				TypeString,
			},
			{
				"ReadSymbol/success",
				[]byte{0x04, 0x00, 0x00, 0x00, 'f', 'o', 'o', 0x00},
				0,
				(*valueReader).ReadSymbol,
				"foo",
				nil,
				TypeSymbol,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				vr := &valueReader{
					offset: tc.offset,
					d:      tc.data,
					stack: []vrState{
						{mode: vrTopLevel},
						{
							mode:   vrElement,
							vType:  tc.vType,
							offset: tc.offset,
						},
					},
					frame: 1,
				}

				css, err := tc.fn(vr)
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if css != tc.css {
					t.Errorf("Incorrect (JavaScript,String,Symbol) returned. got %s; want %s", css, tc.css)
				}
			})
		}
	})
	t.Run("ReadMaxKey/ReadMinKey/ReadNull/ReadUndefined", func(t *testing.T) {
		testCases := []struct {
			name  string
			fn    func(*valueReader) error
			err   error
			vType Type
		}{
			{
				"ReadMaxKey/incorrect type",
				(*valueReader).ReadMaxKey,
				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeMaxKey),
				TypeEmbeddedDocument,
			},
			{
				"ReadMaxKey/success",
				(*valueReader).ReadMaxKey,
				nil,
				TypeMaxKey,
			},
			{
				"ReadMinKey/incorrect type",
				(*valueReader).ReadMinKey,
				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeMinKey),
				TypeEmbeddedDocument,
			},
			{
				"ReadMinKey/success",
				(*valueReader).ReadMinKey,
				nil,
				TypeMinKey,
			},
			{
				"ReadNull/incorrect type",
				(*valueReader).ReadNull,
				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeNull),
				TypeEmbeddedDocument,
			},
			{
				"ReadNull/success",
				(*valueReader).ReadNull,
				nil,
				TypeNull,
			},
			{
				"ReadUndefined/incorrect type",
				(*valueReader).ReadUndefined,
				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeUndefined),
				TypeEmbeddedDocument,
			},
			{
				"ReadUndefined/success",
				(*valueReader).ReadUndefined,
				nil,
				TypeUndefined,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				vr := &valueReader{
					stack: []vrState{
						{mode: vrTopLevel},
						{
							mode:  vrElement,
							vType: tc.vType,
						},
					},
					frame: 1,
				}

				err := tc.fn(vr)
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
			})
		}
	})
	t.Run("ReadObjectID", func(t *testing.T) {
		testCases := []struct {
			name   string
			data   []byte
			offset int64
			oid    objectid.ObjectID
			err    error
			vType  Type
		}{
			{
				"incorrect type",
				[]byte{},
				0,
				objectid.ObjectID{},
				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeObjectID),
				TypeEmbeddedDocument,
			},
			{
				"not enough bytes for objectID",
				[]byte{},
				0,
				objectid.ObjectID{},
				io.EOF,
				TypeObjectID,
			},
			{
				"success",
				[]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
				0,
				objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
				nil,
				TypeObjectID,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				vr := &valueReader{
					offset: tc.offset,
					d:      tc.data,
					stack: []vrState{
						{mode: vrTopLevel},
						{
							mode:   vrElement,
							vType:  tc.vType,
							offset: tc.offset,
						},
					},
					frame: 1,
				}

				oid, err := vr.ReadObjectID()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if !bytes.Equal(oid[:], tc.oid[:]) {
					t.Errorf("ObjectIDs did not match. got %v; want %v", oid, tc.oid)
				}
			})
		}
	})
	t.Run("ReadRegex", func(t *testing.T) {
		testCases := []struct {
			name    string
			data    []byte
			offset  int64
			pattern string
			options string
			err     error
			vType   Type
		}{
			{
				"incorrect type",
				[]byte{},
				0,
				"",
				"",
				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeRegex),
				TypeEmbeddedDocument,
			},
			{
				"length too short",
				[]byte{},
				0,
				"",
				"",
				io.EOF,
				TypeRegex,
			},
			{
				"not enough bytes for options",
				[]byte{'f', 'o', 'o', 0x00},
				0,
				"",
				"",
				io.EOF,
				TypeRegex,
			},
			{
				"success",
				[]byte{'f', 'o', 'o', 0x00, 'b', 'a', 'r', 0x00},
				0,
				"foo",
				"bar",
				nil,
				TypeRegex,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				vr := &valueReader{
					offset: tc.offset,
					d:      tc.data,
					stack: []vrState{
						{mode: vrTopLevel},
						{
							mode:   vrElement,
							vType:  tc.vType,
							offset: tc.offset,
						},
					},
					frame: 1,
				}

				pattern, options, err := vr.ReadRegex()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if pattern != tc.pattern {
					t.Errorf("Incorrect pattern returned. got %s; want %s", pattern, tc.pattern)
				}
				if options != tc.options {
					t.Errorf("Incorrect options returned. got %s; want %s", options, tc.options)
				}
			})
		}
	})
	t.Run("ReadTimestamp", func(t *testing.T) {
		testCases := []struct {
			name   string
			data   []byte
			offset int64
			ts     uint32
			incr   uint32
			err    error
			vType  Type
		}{
			{
				"incorrect type",
				[]byte{},
				0,
				0,
				0,
				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeTimestamp),
				TypeEmbeddedDocument,
			},
			{
				"not enough bytes for increment",
				[]byte{},
				0,
				0,
				0,
				io.EOF,
				TypeTimestamp,
			},
			{
				"not enough bytes for timestamp",
				[]byte{0x01, 0x02, 0x03, 0x04},
				0,
				0,
				0,
				io.EOF,
				TypeTimestamp,
			},
			{
				"success",
				[]byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00},
				0,
				256,
				255,
				nil,
				TypeTimestamp,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				vr := &valueReader{
					offset: tc.offset,
					d:      tc.data,
					stack: []vrState{
						{mode: vrTopLevel},
						{
							mode:   vrElement,
							vType:  tc.vType,
							offset: tc.offset,
						},
					},
					frame: 1,
				}

				ts, incr, err := vr.ReadTimestamp()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if ts != tc.ts {
					t.Errorf("Incorrect timestamp returned. got %d; want %d", ts, tc.ts)
				}
				if incr != tc.incr {
					t.Errorf("Incorrect increment returned. got %d; want %d", incr, tc.incr)
				}
			})
		}
	})
}

func errequal(t *testing.T, err1, err2 error) bool {
	t.Helper()
	if err1 == nil && err2 == nil { // If they are both nil, they are equal
		return true
	}
	if err1 == nil || err2 == nil { // If only one is nil, they are not equal
		return false
	}

	if err1 == err2 { // They are the same error, they are equal
		return true
	}

	if err1.Error() == err2.Error() { // They string formats match, they are equal
		return true
	}

	return false
}
