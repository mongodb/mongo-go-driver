package bsoncodec

import (
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

// Copier is a type that allows copying between ValueReaders, ValueWriters, and
// []byte values.
type Copier struct {
	r *Registry
}

// NewCopier creates a new copier with the given registry. If a nil registry is provided
// a default registry is used.
func NewCopier(r *Registry) Copier {
	return Copier{r: r}
}

func (c Copier) getRegistry() *Registry {
	if c.r != nil {
		return c.r
	}

	return defaultRegistry
}

// CopyDocument handles copying a document from src to dst.
func CopyDocument(dst ValueWriter, src ValueReader) error {
	return Copier{}.CopyDocument(dst, src)
}

// CopyDocument handles copying one document from the src to the dst.
func (c Copier) CopyDocument(dst ValueWriter, src ValueReader) error {
	dr, err := src.ReadDocument()
	if err != nil {
		return err
	}

	dw, err := dst.WriteDocument()
	if err != nil {
		return err
	}

	return c.copyDocumentCore(dw, dr)
}

// CopyDocumentFromBytes copies the values from a BSON document represented as a
// []byte to a ValueWriter.
func (c Copier) CopyDocumentFromBytes(dst ValueWriter, src []byte) error {
	dw, err := dst.WriteDocument()
	if err != nil {
		return err
	}

	itr, err := bson.Reader(src).Iterator()
	if err != nil {
		return err
	}

	for itr.Next() {
		elem := itr.Element()
		dvw, err := dw.WriteDocumentElement(elem.Key())
		if err != nil {
			return err
		}

		val := elem.Value()
		err = defaultValueEncoders.encodeValue(EncodeContext{Registry: c.getRegistry()}, dvw, val)

		if err != nil {
			return err
		}
	}

	if err := itr.Err(); err != nil {
		return err
	}

	return dw.WriteDocumentEnd()
}

// CopyDocumentToBytes copies an entire document from the ValueReader and
// returns it as bytes.
func (c Copier) CopyDocumentToBytes(src ValueReader) ([]byte, error) {
	return c.AppendDocumentBytes(nil, src)
}

// AppendDocumentBytes functions the same as CopyDocumentToBytes, but will
// append the result to dst.
func (c Copier) AppendDocumentBytes(dst []byte, src ValueReader) ([]byte, error) {
	if vr, ok := src.(*valueReader); ok {
		length, err := vr.peakLength()
		if err != nil {
			return dst, err
		}
		dst = append(dst, vr.d[vr.offset:vr.offset+int64(length)]...)
		vr.offset += int64(length)
		vr.pop()
		return dst, nil
	}

	vw := vwPool.Get().(*valueWriter)
	defer vwPool.Put(vw)

	vw.reset(dst)

	err := c.CopyDocument(vw, src)
	dst = vw.buf
	return dst, err
}

// CopyValueFromBytes will write the value represtend by t and src to dst.
func (c Copier) CopyValueFromBytes(dst ValueWriter, t bson.Type, src []byte) error {
	if wvb, ok := dst.(BytesWriter); ok {
		return wvb.WriteValueBytes(t, src)
	}

	vr := vrPool.Get().(*valueReader)
	defer vrPool.Put(vr)

	vr.reset(src)
	vr.pushElement(t)

	return c.CopyValue(dst, vr)
}

// CopyValueToBytes copies a value from src and returns it as a bson.Type and a
// []byte.
func (c Copier) CopyValueToBytes(src ValueReader) (bson.Type, []byte, error) {
	return c.AppendValueBytes(nil, src)
}

// AppendValueBytes functions the same as CopyValueToBytes, but will append the
// result to dst.
func (c Copier) AppendValueBytes(dst []byte, src ValueReader) (bson.Type, []byte, error) {
	if br, ok := src.(BytesReader); ok {
		return br.ReadValueBytes(dst)
	}

	vw := vwPool.Get().(*valueWriter)
	defer vwPool.Put(vw)

	start := len(dst)

	vw.reset(dst)
	vw.push(mElement)

	err := c.CopyValue(vw, src)
	if err != nil {
		return 0, dst, err
	}

	return bson.Type(vw.buf[start]), vw.buf[start+2:], nil
}

// CopyValue will copy a single value from src to dst.
func (c Copier) CopyValue(dst ValueWriter, src ValueReader) error {
	var err error
	switch src.Type() {
	case bson.TypeDouble:
		var f64 float64
		f64, err = src.ReadDouble()
		if err != nil {
			break
		}
		err = dst.WriteDouble(f64)
	case bson.TypeString:
		var str string
		str, err = src.ReadString()
		if err != nil {
			return err
		}
		err = dst.WriteString(str)
	case bson.TypeEmbeddedDocument:
		err = c.CopyDocument(dst, src)
	case bson.TypeArray:
		err = c.copyArray(dst, src)
	case bson.TypeBinary:
		var data []byte
		var subtype byte
		data, subtype, err = src.ReadBinary()
		if err != nil {
			break
		}
		err = dst.WriteBinaryWithSubtype(data, subtype)
	case bson.TypeUndefined:
		err = src.ReadUndefined()
		if err != nil {
			break
		}
		err = dst.WriteUndefined()
	case bson.TypeObjectID:
		var oid objectid.ObjectID
		oid, err = src.ReadObjectID()
		if err != nil {
			break
		}
		err = dst.WriteObjectID(oid)
	case bson.TypeBoolean:
		var b bool
		b, err = src.ReadBoolean()
		if err != nil {
			break
		}
		err = dst.WriteBoolean(b)
	case bson.TypeDateTime:
		var dt int64
		dt, err = src.ReadDateTime()
		if err != nil {
			break
		}
		err = dst.WriteDateTime(dt)
	case bson.TypeNull:
		err = src.ReadNull()
		if err != nil {
			break
		}
		err = dst.WriteNull()
	case bson.TypeRegex:
		var pattern, options string
		pattern, options, err = src.ReadRegex()
		if err != nil {
			break
		}
		err = dst.WriteRegex(pattern, options)
	case bson.TypeDBPointer:
		var ns string
		var pointer objectid.ObjectID
		ns, pointer, err = src.ReadDBPointer()
		if err != nil {
			break
		}
		err = dst.WriteDBPointer(ns, pointer)
	case bson.TypeJavaScript:
		var js string
		js, err = src.ReadJavascript()
		if err != nil {
			break
		}
		err = dst.WriteJavascript(js)
	case bson.TypeSymbol:
		var symbol string
		symbol, err = src.ReadSymbol()
		if err != nil {
			break
		}
		err = dst.WriteSymbol(symbol)
	case bson.TypeCodeWithScope:
		var code string
		var srcScope DocumentReader
		code, srcScope, err = src.ReadCodeWithScope()
		if err != nil {
			break
		}

		var dstScope DocumentWriter
		dstScope, err = dst.WriteCodeWithScope(code)
		if err != nil {
			break
		}
		err = c.copyDocumentCore(dstScope, srcScope)
	case bson.TypeInt32:
		var i32 int32
		i32, err = src.ReadInt32()
		if err != nil {
			break
		}
		err = dst.WriteInt32(i32)
	case bson.TypeTimestamp:
		var t, i uint32
		t, i, err = src.ReadTimestamp()
		if err != nil {
			break
		}
		err = dst.WriteTimestamp(t, i)
	case bson.TypeInt64:
		var i64 int64
		i64, err = src.ReadInt64()
		if err != nil {
			break
		}
		err = dst.WriteInt64(i64)
	case bson.TypeDecimal128:
		var d128 decimal.Decimal128
		d128, err = src.ReadDecimal128()
		if err != nil {
			break
		}
		err = dst.WriteDecimal128(d128)
	case bson.TypeMinKey:
		err = src.ReadMinKey()
		if err != nil {
			break
		}
		err = dst.WriteMinKey()
	case bson.TypeMaxKey:
		err = src.ReadMaxKey()
		if err != nil {
			break
		}
		err = dst.WriteMaxKey()
	default:
		err = fmt.Errorf("Cannot copy unknown BSON type %s", src.Type())
	}

	return err
}

func (c Copier) copyArray(dst ValueWriter, src ValueReader) error {
	ar, err := src.ReadArray()
	if err != nil {
		return err
	}

	aw, err := dst.WriteArray()
	if err != nil {
		return err
	}

	for {
		vr, err := ar.ReadValue()
		if err == ErrEOA {
			break
		}
		if err != nil {
			return err
		}

		vw, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		err = c.CopyValue(vw, vr)
		if err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (c Copier) copyDocumentCore(dw DocumentWriter, dr DocumentReader) error {
	for {
		key, vr, err := dr.ReadElement()
		if err == ErrEOD {
			break
		}
		if err != nil {
			return err
		}

		vw, err := dw.WriteDocumentElement(key)
		if err != nil {
			return err
		}

		err = c.CopyValue(vw, vr)
		if err != nil {
			return err
		}
	}

	return dw.WriteDocumentEnd()
}
