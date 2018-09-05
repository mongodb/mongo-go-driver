package bsoncodec

import (
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

type Copier struct{}

// CopyDocument handles copying a document from src to dst.
func CopyDocument(dst ValueWriter, src ValueReader) error {
	return Copier{}.copyDocument(dst, src)
}

func (c Copier) CopyDocument(dst ValueWriter, src ValueReader) error {
	return c.copyDocument(dst, src)
}

func (c Copier) CopyDocumentFromBytes(dst ValueWriter, src []byte) error         { return nil }
func (c Copier) CopyDocumentToBytes(src ValueReader) ([]byte, error)             { return nil, nil }
func (c Copier) AppendDocumentBytes(dst []byte, src ValueReader) ([]byte, error) { return nil, nil }

func (c Copier) CopyElement(dst DocumentWriter, src DocumentReader) error          { return nil }
func (c Copier) CopyElementFromBytes(dst DocumentWriter, src []byte) error         { return nil }
func (c Copier) CopyElementToBytes(src DocumentReader) ([]byte, error)             { return nil, nil }
func (c Copier) AppendElementBytes(dst []byte, src DocumentReader) ([]byte, error) { return nil, nil }

func (c Copier) copyDocument(dst ValueWriter, src ValueReader) error {
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

		err = c.copyElement(vw, vr)
		if err != nil {
			return err
		}
	}

	return dw.WriteDocumentEnd()
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

func (c Copier) copyElement(dst ValueWriter, src ValueReader) error {
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
		err = c.copyDocument(dst, src)
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
