// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson
/*
import (
	"strings"

	"github.com/mongodb/mongo-go-driver/bson/builder"
)

// ParseExtJSONObject parses a JSON object string into a *Document.
func ParseExtJSONObject(s string) (*Document, error) {
	b := builder.NewDocumentBuilder()

	ejvr := NewExtJSONValueReader(strings.NewReader(s))
	dr, err := ejvr.ReadDocument()
	if err != nil {
		return nil, err
	}

	if err = parseObjectToBuilder(b, dr); err != nil {
		return nil, err
	}

	buf := make([]byte, b.RequiredBytes())
	if _, err = b.WriteDocument(buf); err != nil {
		return nil, err
	}

	return ReadDocument(buf)
}

// ParseExtJSONArray parses a JSON array string into a *Array.
func ParseExtJSONArray(s string) (*Array, error) {
	ejvr := NewExtJSONValueReader(strings.NewReader(s))
	ar, err := ejvr.ReadArray()

	b, err := parseArrayToBuilder(ar)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, b.RequiredBytes())
	if _, err = b.WriteDocument(buf); err != nil {
		return nil, err
	}

	doc, err := ReadDocument(buf)
	if err != nil {
		return nil, err
	}

	array := ArrayFromDocument(doc)

	return array, nil
}

func parseObjectToBuilder(b *builder.DocumentBuilder, dr DocumentReader) error {
	k, vr, err := dr.ReadElement()
	for ; err == nil; k, vr, err = dr.ReadElement() {
		switch vr.Type() {
		case TypeDouble:
			f, err := vr.ReadDouble()
			if err != nil {
				return err
			}

			b.Append(builder.C.Double(k, f))
		case TypeString:
			s, err := vr.ReadString()
			if err != nil {
				return err
			}

			b.Append(builder.C.String(k, s))
		case TypeBinary:
			bv, bt, err := vr.ReadBinary()
			if err != nil {
				return err
			}

			b.Append(builder.C.BinaryWithSubtype(k, bv, bt))
		case TypeUndefined:
			err = vr.ReadUndefined()
			if err != nil {
				return err
			}

			b.Append(builder.C.Undefined(k))
		case TypeObjectID:
			oid, err := vr.ReadObjectID()
			if err != nil {
				return err
			}

			b.Append(builder.C.ObjectID(k, oid))
		case TypeBoolean:
			bv, err := vr.ReadBoolean()
			if err != nil {
				return err
			}

			b.Append(builder.C.Boolean(k, bv))
		case TypeDateTime:
			dt, err := vr.ReadDateTime()
			if err != nil {
				return err
			}

			b.Append(builder.C.DateTime(k, dt))
		case TypeNull:
			err := vr.ReadNull()
			if err != nil {
				return err
			}

			b.Append(builder.C.Null(k))
		case TypeRegex:
			p, o, err := vr.ReadRegex()
			if err != nil {
				return err
			}

			b.Append(builder.C.Regex(k, p, o))
		case TypeDBPointer:
			s, oid, err := vr.ReadDBPointer()
			if err != nil {
				return err
			}

			b.Append(builder.C.DBPointer(k, s, oid))
		case TypeJavaScript:
			c, err := vr.ReadJavascript()
			if err != nil {
				return err
			}

			b.Append(builder.C.JavaScriptCode(k, c))
		case TypeSymbol:
			s, err := vr.ReadSymbol()
			if err != nil {
				return err
			}

			b.Append(builder.C.Symbol(k, s))
		case TypeInt32:
			i, err := vr.ReadInt32()
			if err != nil {
				return err
			}

			b.Append(builder.C.Int32(k, i))
		case TypeTimestamp:
			t, i, err := vr.ReadTimestamp()
			if err != nil {
				return err
			}

			b.Append(builder.C.Timestamp(k, t, i))
		case TypeInt64:
			i, err := vr.ReadInt64()
			if err != nil {
				return err
			}

			b.Append(builder.C.Int64(k, i))
		case TypeDecimal128:
			d, err := vr.ReadDecimal128()
			if err != nil {
				return err
			}

			b.Append(builder.C.Decimal(k, d))
		case TypeMinKey:
			err := vr.ReadMinKey()
			if err != nil {
				return err
			}

			b.Append(builder.C.MinKey(k))
		case TypeMaxKey:
			err := vr.ReadMaxKey()
			if err != nil {
				return err
			}

			b.Append(builder.C.MaxKey(k))
		case TypeCodeWithScope:
			c, sr, err := vr.ReadCodeWithScope()
			if err != nil {
				return err
			}

			scopeDoc := builder.NewDocumentBuilder()
			err = parseObjectToBuilder(scopeDoc, sr)
			if err != nil {
				return err
			}

			buf := make([]byte, scopeDoc.RequiredBytes())
			_, err = scopeDoc.WriteDocument(buf)
			if err != nil {
				return err
			}

			b.Append(builder.C.CodeWithScope(k, c, buf))
		case TypeEmbeddedDocument:
			sdr, err := vr.ReadDocument()
			if err != nil {
				return err
			}

			subDoc := builder.NewDocumentBuilder()
			err = parseObjectToBuilder(subDoc, sdr)
			if err != nil {
				return err
			}

			b.Append(builder.C.SubDocument(k, subDoc))
		case TypeArray:
			ar, err := vr.ReadArray()
			if err != nil {
				return err
			}

			ab, err := parseArrayToBuilder(ar)
			if err != nil {
				return err
			}

			b.Append(builder.C.Array(k, ab))
		}
	}

	// expect end of document error
	if err != ErrEOD {
		return err
	}

	return nil
}

func parseArrayToBuilder(ar ArrayReader) (*builder.ArrayBuilder, error) {
	var ab builder.ArrayBuilder

	vr, err := ar.ReadValue()
	for ; err == nil; vr, err = ar.ReadValue() {
		switch vr.Type() {
		case TypeDouble:
			f, err := vr.ReadDouble()
			if err != nil {
				return nil, err
			}

			ab.Append(builder.AC.Double(f))
		case TypeString:
			s, err := vr.ReadString()
			if err != nil {
				return nil, err
			}

			ab.Append(builder.AC.String(s))
		case TypeBinary:
			bv, bt, err := vr.ReadBinary()
			if err != nil {
				return nil, err
			}

			ab.Append(builder.AC.BinaryWithSubtype(bv, bt))
		case TypeUndefined:
			err = vr.ReadUndefined()
			if err != nil {
				return nil, err
			}

			ab.Append(builder.AC.Undefined())
		case TypeObjectID:
			oid, err := vr.ReadObjectID()
			if err != nil {
				return nil, err
			}

			ab.Append(builder.AC.ObjectID(oid))
		case TypeBoolean:
			bv, err := vr.ReadBoolean()
			if err != nil {
				return nil, err
			}

			ab.Append(builder.AC.Boolean(bv))
		case TypeDateTime:
			dt, err := vr.ReadDateTime()
			if err != nil {
				return nil, err
			}

			ab.Append(builder.AC.DateTime(dt))
		case TypeNull:
			err := vr.ReadNull()
			if err != nil {
				return nil, err
			}

			ab.Append(builder.AC.Null())
		case TypeRegex:
			p, o, err := vr.ReadRegex()
			if err != nil {
				return nil, err
			}

			ab.Append(builder.AC.Regex(p, o))
		case TypeDBPointer:
			s, oid, err := vr.ReadDBPointer()
			if err != nil {
				return nil, err
			}

			ab.Append(builder.AC.DBPointer(s, oid))
		case TypeJavaScript:
			c, err := vr.ReadJavascript()
			if err != nil {
				return nil, err
			}

			ab.Append(builder.AC.JavaScriptCode(c))
		case TypeSymbol:
			s, err := vr.ReadSymbol()
			if err != nil {
				return nil, err
			}

			ab.Append(builder.AC.Symbol(s))
		case TypeInt32:
			i, err := vr.ReadInt32()
			if err != nil {
				return nil, err
			}

			ab.Append(builder.AC.Int32(i))
		case TypeTimestamp:
			t, i, err := vr.ReadTimestamp()
			if err != nil {
				return nil, err
			}

			ab.Append(builder.AC.Timestamp(t, i))
		case TypeInt64:
			i, err := vr.ReadInt64()
			if err != nil {
				return nil, err
			}

			ab.Append(builder.AC.Int64(i))
		case TypeDecimal128:
			d, err := vr.ReadDecimal128()
			if err != nil {
				return nil, err
			}

			ab.Append(builder.AC.Decimal(d))
		case TypeMinKey:
			err := vr.ReadMinKey()
			if err != nil {
				return nil, err
			}

			ab.Append(builder.AC.MinKey())
		case TypeMaxKey:
			err := vr.ReadMaxKey()
			if err != nil {
				return nil, err
			}

			ab.Append(builder.AC.MaxKey())
		case TypeCodeWithScope:
			c, sr, err := vr.ReadCodeWithScope()
			if err != nil {
				return nil, err
			}

			scopeDoc := builder.NewDocumentBuilder()
			err = parseObjectToBuilder(scopeDoc, sr)
			if err != nil {
				return nil, err
			}

			buf := make([]byte, scopeDoc.RequiredBytes())
			_, err = scopeDoc.WriteDocument(buf)
			if err != nil {
				return nil, err
			}

			ab.Append(builder.AC.CodeWithScope(c, buf))
		case TypeEmbeddedDocument:
			sdr, err := vr.ReadDocument()
			if err != nil {
				return nil, err
			}

			subDoc := builder.NewDocumentBuilder()
			err = parseObjectToBuilder(subDoc, sdr)
			if err != nil {
				return nil, err
			}

			ab.Append(builder.AC.SubDocument(subDoc))
		case TypeArray:
			ar, err := vr.ReadArray()
			if err != nil {
				return nil, err
			}

			sab, err := parseArrayToBuilder(ar)
			if err != nil {
				return nil, err
			}

			ab.Append(builder.AC.Array(sab))
		}
	}

	if err != ErrEOA {
		return nil, err
	}

	return &ab, nil
}
*/
