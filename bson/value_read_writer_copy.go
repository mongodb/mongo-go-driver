package bson

import (
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

type copier struct{}

// CopyDocument handles copying a document from src to dst.
func CopyDocument(dst ValueWriter, src ValueReader) error {
	return copier{}.copyDocument(dst, src)
}

func (c copier) copyDocument(dst ValueWriter, src ValueReader) error {
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

func (c copier) copyDocumentCore(dw DocumentWriter, dr DocumentReader) error {
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

		err = c.copyElement(vw, vr)
		if err != nil {
			return err
		}
	}

	return dw.WriteDocumentEnd()
}

func (c copier) copyArray(dst ValueWriter, src ValueReader) error {
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
		if err == ErrEOD {
			break
		}
		if err != nil {
			return err
		}

		vw, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		err = c.copyElement(vw, vr)
		if err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (c copier) copyElement(dst ValueWriter, src ValueReader) error {
	var err error
	switch src.Type() {
	case TypeDouble:
		var f64 float64
		f64, err = src.ReadDouble()
		if err != nil {
			break
		}
		err = dst.WriteDouble(f64)
	case TypeString:
		var str string
		str, err = src.ReadString()
		if err != nil {
			return err
		}
		err = dst.WriteString(str)
	case TypeEmbeddedDocument:
		err = c.copyDocument(dst, src)
	case TypeArray:
		err = c.copyArray(dst, src)
	case TypeBinary:
		var data []byte
		var subtype byte
		data, subtype, err = src.ReadBinary()
		if err != nil {
			break
		}
		err = dst.WriteBinaryWithSubtype(data, subtype)
	case TypeUndefined:
		err = src.ReadUndefined()
		if err != nil {
			break
		}
		err = dst.WriteUndefined()
	case TypeObjectID:
		var oid objectid.ObjectID
		oid, err = src.ReadObjectID()
		if err != nil {
			break
		}
		err = dst.WriteObjectID(oid)
	case TypeBoolean:
		var b bool
		b, err = src.ReadBoolean()
		if err != nil {
			break
		}
		err = dst.WriteBoolean(b)
	case TypeDateTime:
		var dt int64
		dt, err = src.ReadDateTime()
		if err != nil {
			break
		}
		err = dst.WriteDateTime(dt)
	case TypeNull:
		err = src.ReadNull()
		if err != nil {
			break
		}
		err = dst.WriteNull()
	case TypeRegex:
		var pattern, options string
		pattern, options, err = src.ReadRegex()
		if err != nil {
			break
		}
		err = dst.WriteRegex(pattern, options)
	case TypeDBPointer:
		var ns string
		var pointer objectid.ObjectID
		ns, pointer, err = src.ReadDBPointer()
		if err != nil {
			break
		}
		err = dst.WriteDBPointer(ns, pointer)
	case TypeJavaScript:
		var js string
		js, err = src.ReadJavascript()
		if err != nil {
			break
		}
		err = dst.WriteJavascript(js)
	case TypeSymbol:
		var symbol string
		symbol, err = src.ReadSymbol()
		if err != nil {
			break
		}
		err = dst.WriteSymbol(symbol)
	case TypeCodeWithScope:
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
	case TypeInt32:
		var i32 int32
		i32, err = src.ReadInt32()
		if err != nil {
			break
		}
		err = dst.WriteInt32(i32)
	case TypeTimestamp:
		var t, i uint32
		t, i, err = src.ReadTimestamp()
		if err != nil {
			break
		}
		err = dst.WriteTimestamp(t, i)
	case TypeInt64:
		var i64 int64
		i64, err = src.ReadInt64()
		if err != nil {
			break
		}
		err = dst.WriteInt64(i64)
	case TypeDecimal128:
		var d128 decimal.Decimal128
		d128, err = src.ReadDecimal128()
		if err != nil {
			break
		}
		err = dst.WriteDecimal128(d128)
	case TypeMinKey:
		err = src.ReadMinKey()
		if err != nil {
			break
		}
		err = dst.WriteMinKey()
	case TypeMaxKey:
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
