package bson

import (
	"encoding/binary"
	"math"
	"time"

	"github.com/10gen/mongo-go-driver/bson/decimal"
	"github.com/10gen/mongo-go-driver/bson/objectid"
)

// Value represents a BSON value. It can be obtained as part of a bson.Element or created for use
// in a bson.Array with the bson.AC constructors.
type Value struct {
	// NOTE: For subdocuments, arrays, and code with scope, the data slice of
	// bytes may contain just the key, or the key and the code in the case of
	// code with scope. If this is the case, the start will be 0, the value will
	// be the length of the slice, and d will be non-nil.

	// start is the offset into the data slice of bytes where this element
	// begins.
	start uint32
	// offset is the offset into the data slice of bytes where this element's
	// value begins.
	offset uint32
	// data is a potentially shared slice of bytes that contains the actual
	// element. Most of the methods of this type directly index into this slice
	// of bytes.
	data []byte

	d *Document
}

func (v *Value) validate(sizeOnly bool) (uint32, error) {
	if v.data == nil {
		return 0, ErrUninitializedElement
	}

	var total uint32

	switch v.data[v.start] {
	case '\x06', '\x0A', '\xFF', '\x7F':
	case '\x01':
		if int(v.offset+8) > len(v.data) {
			return total, ErrTooSmall
		}
		total += 8
	case '\x02', '\x0D', '\x0E':
		if int(v.offset+4) > len(v.data) {
			return total, ErrTooSmall
		}
		l := readi32(v.data[v.offset : v.offset+4])
		total += 4
		if int32(v.offset)+4+l > int32(len(v.data)) {
			return total, ErrTooSmall
		}
		// We check if the value that is the last element of the string is a
		// null terminator. We take the value offset, add 4 to account for the
		// length, add the length of the string, and subtract one since the size
		// isn't zero indexed.
		if !sizeOnly && v.data[v.offset+4+uint32(l)-1] != 0x00 {
			return total, ErrInvalidString
		}
		total += uint32(l)
	case '\x03':
		if v.d != nil {
			n, err := v.d.Validate()
			total += uint32(n)
			if err != nil {
				return total, err
			}
			break
		}

		if int(v.offset+4) > len(v.data) {
			return total, ErrTooSmall
		}
		l := readi32(v.data[v.offset : v.offset+4])
		total += 4
		if l < 5 {
			return total, ErrInvalidReadOnlyDocument
		}
		if int32(v.offset)+l > int32(len(v.data)) {
			return total, ErrTooSmall
		}
		if !sizeOnly {
			n, err := Reader(v.data[v.offset : v.offset+uint32(l)]).Validate()
			total += n - 4
			if err != nil {
				return total, err
			}
			break
		}
		total += uint32(l) - 4
	case '\x04':
		if v.d != nil {
			n, err := (&Array{v.d}).Validate()
			total += uint32(n)
			if err != nil {
				return total, err
			}
			break
		}

		if int(v.offset+4) > len(v.data) {
			return total, ErrTooSmall
		}
		l := readi32(v.data[v.offset : v.offset+4])
		total += 4
		if l < 5 {
			return total, ErrInvalidReadOnlyDocument
		}
		if int32(v.offset)+l > int32(len(v.data)) {
			return total, ErrTooSmall
		}
		if !sizeOnly {
			n, err := Reader(v.data[v.offset : v.offset+uint32(l)]).Validate()
			total += n - 4
			if err != nil {
				return total, err
			}
			break
		}
		total += uint32(l) - 4
	case '\x05':
		if int(v.offset+5) > len(v.data) {
			return total, ErrTooSmall
		}
		l := readi32(v.data[v.offset : v.offset+4])
		total += 5
		if v.data[v.offset+4] > '\x05' && v.data[v.offset+4] < '\x80' {
			return total, ErrInvalidBinarySubtype
		}
		if int32(v.offset)+5+l > int32(len(v.data)) {
			return total, ErrTooSmall
		}
		total += uint32(l)
	case '\x07':
		if int(v.offset+12) > len(v.data) {
			return total, ErrTooSmall
		}
		total += 12
	case '\x08':
		if int(v.offset+1) > len(v.data) {
			return total, ErrTooSmall
		}
		total++
		if v.data[v.offset] != '\x00' && v.data[v.offset] != '\x01' {
			return total, ErrInvalidBooleanType
		}
	case '\x09':
		if int(v.offset+8) > len(v.data) {
			return total, ErrTooSmall
		}
		total += 8
	case '\x0B':
		i := v.offset
		for ; int(i) < len(v.data) && v.data[i] != '\x00'; i++ {
			total++
		}
		if int(i) == len(v.data) || v.data[i] != '\x00' {
			return total, ErrInvalidString
		}
		i++
		total++
		for ; int(i) < len(v.data) && v.data[i] != '\x00'; i++ {
			total++
		}
		if int(i) == len(v.data) || v.data[i] != '\x00' {
			return total, ErrInvalidString
		}
		total++
	case '\x0C':
		if int(v.offset+4) > len(v.data) {
			return total, ErrTooSmall
		}
		l := readi32(v.data[v.offset : v.offset+4])
		total += 4
		if int32(v.offset)+4+l+12 > int32(len(v.data)) {
			return total, ErrTooSmall
		}
		total += uint32(l) + 12
	case '\x0F':
		if v.d != nil {
			// NOTE: For code with scope specifically, we write the length as
			// we are marshaling the element and the constructor doesn't know
			// the length of the document when it constructs the element.
			// Because of that we don't check the length here and just validate
			// the string and the document.
			if int(v.offset+8) > len(v.data) {
				return total, ErrTooSmall
			}
			total += 8
			sLength := readi32(v.data[v.offset+4 : v.offset+8])
			if int(sLength) > len(v.data)+8 {
				return total, ErrTooSmall
			}
			total += uint32(sLength)
			if !sizeOnly && v.data[v.offset+8+uint32(sLength)-1] != 0x00 {
				return total, ErrInvalidString
			}

			n, err := v.d.Validate()
			total += uint32(n)
			if err != nil {
				return total, err
			}
			break
		}
		if int(v.offset+4) > len(v.data) {
			return total, ErrTooSmall
		}
		l := readi32(v.data[v.offset : v.offset+4])
		total += 4
		if int32(v.offset)+l > int32(len(v.data)) {
			return total, ErrTooSmall
		}
		if !sizeOnly {
			sLength := readi32(v.data[v.offset+4 : v.offset+8])
			total += 4
			// If the length of the string is larger than the total length of the
			// field minus the int32 for length, 5 bytes for a minimum document
			// size, and an int32 for the string length the value is invalid.
			//
			// TODO(skriptble): We should actually validate that the string
			// doesn't consume any of the bytes used by the document.
			if sLength > l-13 {
				return total, ErrStringLargerThanContainer
			}
			// We check if the value that is the last element of the string is a
			// null terminator. We take the value offset, add 4 to account for the
			// length, add the length of the string, and subtract one since the size
			// isn't zero indexed.
			if v.data[v.offset+8+uint32(sLength)-1] != 0x00 {
				return total, ErrInvalidString
			}
			total += uint32(sLength)
			n, err := Reader(v.data[v.offset+8+uint32(sLength) : v.offset+uint32(l)]).Validate()
			total += n
			if err != nil {
				return total, err
			}
			break
		}
		total += uint32(l) - 4
	case '\x10':
		if int(v.offset+4) > len(v.data) {
			return total, ErrTooSmall
		}
		total += 4
	case '\x11', '\x12':
		if int(v.offset+8) > len(v.data) {
			return total, ErrTooSmall
		}
		total += 8
	case '\x13':
		if int(v.offset+16) > len(v.data) {
			return total, ErrTooSmall
		}
		total += 16

	default:
		return total, ErrInvalidElement
	}

	return total, nil
}

// valueSize returns the size of the value in bytes.
func (v *Value) valueSize() (uint32, error) {
	return v.validate(true)
}

// Type returns the identifying element byte for this element.
// It panics if e is uninitialized.
func (v *Value) Type() Type {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	return Type(v.data[v.start])
}

// Double returns the float64 value for this element.
// It panics if e's BSON type is not double ('\x01') or if e is uninitialized.
func (v *Value) Double() float64 {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x01' {
		panic(ElementTypeError{"compact.Element.double", Type(v.data[v.start])})
	}
	bits := binary.LittleEndian.Uint64(v.data[v.offset : v.offset+8])
	return math.Float64frombits(bits)
}

// StringValue returns the string balue for this element.
// It panics if e's BSON type is not StringValue ('\x02') or if e is uninitialized.
//
// NOTE: This method is called StringValue to avoid it implementing the
// fmt.Stringer interface.
func (v *Value) StringValue() string {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x02' {
		panic(ElementTypeError{"compact.Element.String", Type(v.data[v.start])})
	}
	l := readi32(v.data[v.offset : v.offset+4])
	return string(v.data[v.offset+4 : int32(v.offset)+4+l-1])
}

// ReaderDocument returns the BSON document the Value represents as a bson.Reader. It panics if the
// value is a BSON type other than document.
func (v *Value) ReaderDocument() Reader {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}

	if v.data[v.start] != '\x03' {
		panic(ElementTypeError{"compact.Element.Document", Type(v.data[v.start])})
	}

	var r Reader
	if v.d == nil {
		l := readi32(v.data[v.offset : v.offset+4])
		r = Reader(v.data[v.offset : v.offset+uint32(l)])
	} else {
		scope, err := v.d.MarshalBSON()
		if err != nil {
			panic(err)
		}

		r = Reader(scope)
	}

	return r
}

// MutableDocument returns the subdocument for this element.
func (v *Value) MutableDocument() *Document {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x03' {
		panic(ElementTypeError{"compact.Element.Document", Type(v.data[v.start])})
	}
	if v.d == nil {
		var err error
		l := int32(binary.LittleEndian.Uint32(v.data[v.offset : v.offset+4]))
		v.d, err = ReadDocument(v.data[v.offset : v.offset+uint32(l)])
		if err != nil {
			panic(err)
		}
	}
	return v.d
}

// ReaderArray returns the BSON document the Value represents as a bson.Reader. It panics if the
// value is a BSON type other than array.
func (v *Value) ReaderArray() Reader {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}

	if v.data[v.start] != '\x04' {
		panic(ElementTypeError{"compact.Element.Array", Type(v.data[v.start])})
	}

	var r Reader
	if v.d == nil {
		l := readi32(v.data[v.offset : v.offset+4])
		r = Reader(v.data[v.offset : v.offset+uint32(l)])
	} else {
		scope, err := v.d.MarshalBSON()
		if err != nil {
			panic(err)
		}

		r = Reader(scope)
	}

	return r
}

// MutableArray returns the array for this element.
func (v *Value) MutableArray() *Array {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x04' {
		panic(ElementTypeError{"compact.Element.Array", Type(v.data[v.start])})
	}
	if v.d == nil {
		var err error
		l := int32(binary.LittleEndian.Uint32(v.data[v.offset : v.offset+4]))
		v.d, err = ReadDocument(v.data[v.offset : v.offset+uint32(l)])
		if err != nil {
			panic(err)
		}
	}
	return &Array{v.d}
}

// Binary returns the BSON binary value the Value represents. It panics if the value is a BSON type
// other than binary.
func (v *Value) Binary() (subtype byte, data []byte) {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x05' {
		panic(ElementTypeError{"compact.Element.binary", Type(v.data[v.start])})
	}
	l := readi32(v.data[v.offset : v.offset+4])
	st := v.data[v.offset+4]
	b := make([]byte, l)
	copy(b, v.data[v.offset+5:int32(v.offset)+5+l])
	return st, b
}

// ObjectID returns the BSON objectid value the Value represents. It panics if the value is a BSON
// type other than objectid.
func (v *Value) ObjectID() objectid.ObjectID {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x07' {
		panic(ElementTypeError{"compact.Element.ObejctID", Type(v.data[v.start])})
	}
	var arr [12]byte
	copy(arr[:], v.data[v.offset:v.offset+12])
	return arr
}

// Boolean returns the boolean value the Value represents. It panics if the
// value is a BSON type other than boolean.
func (v *Value) Boolean() bool {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x08' {
		panic(ElementTypeError{"compact.Element.Boolean", Type(v.data[v.start])})
	}
	return v.data[v.offset] == '\x01'
}

// DateTime returns the BSON datetime value the Value represents. It panics if the value is a BSON
// type other than datetime.
func (v *Value) DateTime() time.Time {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x09' {
		panic(ElementTypeError{"compact.Element.dateTime", Type(v.data[v.start])})
	}
	i := binary.LittleEndian.Uint64(v.data[v.offset : v.offset+8])
	return time.Unix(int64(i)/1000, int64(i)%1000*1000000)
}

// Regex returns the BSON regex value the Value represents. It panics if the value is a BSON
// type other than regex.
func (v *Value) Regex() (pattern, options string) {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x0B' {
		panic(ElementTypeError{"compact.Element.regex", Type(v.data[v.start])})
	}
	// TODO(skriptble): Use the elements package here.
	var pstart, pend, ostart, oend uint32
	i := v.offset
	pstart = i
	for ; v.data[i] != '\x00'; i++ {
	}
	pend = i
	i++
	ostart = i
	for ; v.data[i] != '\x00'; i++ {
	}
	oend = i

	return string(v.data[pstart:pend]), string(v.data[ostart:oend])
}

// DBPointer returns the BSON dbpointer value the Value represents. It panics if the value is a BSON
// type other than DBPointer.
func (v *Value) DBPointer() (string, objectid.ObjectID) {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x0C' {
		panic(ElementTypeError{"compact.Element.dbPointer", Type(v.data[v.start])})
	}
	l := readi32(v.data[v.offset : v.offset+4])
	var p [12]byte
	copy(p[:], v.data[v.offset+4+uint32(l):v.offset+4+uint32(l)+12])
	return string(v.data[v.offset+4 : int32(v.offset)+4+l-1]), p
}

// JavaScript returns the BSON JavaScript code value the Value represents. It panics if the value is
// a BSON type other than JavaScript code.
func (v *Value) JavaScript() string {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x0D' {
		panic(ElementTypeError{"compact.Element.JavaScript", Type(v.data[v.start])})
	}
	l := readi32(v.data[v.offset : v.offset+4])
	return string(v.data[v.offset+4 : int32(v.offset)+4+l-1])
}

// Symbol returns the BSON symbol value the Value represents. It panics if the value is a BSON
// type other than symbol.
func (v *Value) Symbol() string {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x0E' {
		panic(ElementTypeError{"compact.Element.symbol", Type(v.data[v.start])})
	}
	l := readi32(v.data[v.offset : v.offset+4])
	return string(v.data[v.offset+4 : int32(v.offset)+4+l-1])
}

// ReaderJavaScriptWithScope returns the BSON JavaScript code with scope the Value represents, with
// the scope being returned as a bson.Reader. It panics if the value is a BSON type other than
// JavaScript code with scope.
func (v *Value) ReaderJavaScriptWithScope() (string, Reader) {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}

	if v.data[v.start] != '\x0F' {
		panic(ElementTypeError{"compact.Element.JavaScriptWithScope", Type(v.data[v.start])})
	}

	sLength := readi32(v.data[v.offset+4 : v.offset+8])
	// If the length of the string is larger than the total length of the
	// field minus the int32 for length, 5 bytes for a minimum document
	// size, and an int32 for the string length the value is invalid.
	str := string(v.data[v.offset+8 : v.offset+8+uint32(sLength)-1])

	var r Reader
	if v.d == nil {
		l := readi32(v.data[v.offset : v.offset+4])
		r = Reader(v.data[v.offset+8+uint32(sLength) : v.offset+uint32(l)])
	} else {
		scope, err := v.d.MarshalBSON()
		if err != nil {
			panic(err)
		}

		r = Reader(scope)
	}

	return str, r
}

// MutableJavaScriptWithScope returns the javascript code and the scope document for
// this element.
func (v *Value) MutableJavaScriptWithScope() (code string, d *Document) {
	if v == nil || v.offset == 0 {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x0F' {
		panic(ElementTypeError{"compact.Element.JavaScriptWithScope", Type(v.data[v.start])})
	}
	// TODO(skriptble): This is wrong and could cause a panic.
	l := int32(binary.LittleEndian.Uint32(v.data[v.offset : v.offset+4]))
	// TODO(skriptble): This is wrong and could cause a panic.
	sLength := int32(binary.LittleEndian.Uint32(v.data[v.offset+4 : v.offset+8]))
	// If the length of the string is larger than the total length of the
	// field minus the int32 for length, 5 bytes for a minimum document
	// size, and an int32 for the string length the value is invalid.
	str := string(v.data[v.offset+4 : v.offset+4+uint32(sLength)])
	if v.d == nil {
		var err error
		v.d, err = ReadDocument(v.data[v.offset+4+uint32(sLength) : v.offset+uint32(l)])
		if err != nil {
			panic(err)
		}
	}
	return str, v.d
}

// Int32 returns the int32 the Value represents. It panics if the value is a BSON type other than
// int32.
func (v *Value) Int32() int32 {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x10' {
		panic(ElementTypeError{"compact.Element.int32", Type(v.data[v.start])})
	}
	return readi32(v.data[v.offset : v.offset+4])
}

// Timestamp returns the BSON timestamp value the Value represents. It panics if the value is a
// BSON type other than timestamp.
func (v *Value) Timestamp() (uint32, uint32) {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x11' {
		panic(ElementTypeError{"compact.Element.timestamp", Type(v.data[v.start])})
	}
	return binary.LittleEndian.Uint32(v.data[v.offset : v.offset+4]), binary.LittleEndian.Uint32(v.data[v.offset+4 : v.offset+8])
}

// Int64 returns the int64 the Value represents. It panics if the value is a BSON type other than
// int64.
func (v *Value) Int64() int64 {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x12' {
		panic(ElementTypeError{"compact.Element.int64Type", Type(v.data[v.start])})
	}
	return int64(binary.LittleEndian.Uint64(v.data[v.offset : v.offset+8]))
}

// Decimal128 returns the decimal the Value represents. It panics if the value is a BSON type other than
// decimal.
func (v *Value) Decimal128() decimal.Decimal128 {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x13' {
		panic(ElementTypeError{"compact.Element.Decimal128", Type(v.data[v.start])})
	}
	l := binary.LittleEndian.Uint64(v.data[v.offset : v.offset+8])
	h := binary.LittleEndian.Uint64(v.data[v.offset+8 : v.offset+16])
	return decimal.NewDecimal128(h, l)
}
