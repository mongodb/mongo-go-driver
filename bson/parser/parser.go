// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package parser

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"math"

	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/parser/ast"
)

// ErrCorruptDocument is returned when the parser reaches a corrupt point
// within a BSON document.
var ErrCorruptDocument = errors.New("bson/parser: corrupted document")

// ErrUnknownSubtype is returned when the subtype of a binary node is undefined.
var ErrUnknownSubtype = errors.New("bson/parser: unknown binary subtype")

// ErrNilReader is returned when a nil reader is passed to NewBSONParser.
var ErrNilReader = errors.New("bson/parser: nil or invalid reader provided")

// Parser is a BSON parser.
type Parser struct {
	r *bufio.Reader
}

// NewBSONParser instantiates a new BSON Parser with the given reader.
func NewBSONParser(r io.Reader) (*Parser, error) {
	if r == nil {
		return nil, ErrNilReader
	}
	return &Parser{r: bufio.NewReader(r)}, nil
}

// readInt32 reads a single int32 from the parser's reader.
func (p *Parser) readInt32() (int32, error) {
	var i int32
	err := binary.Read(p.r, binary.LittleEndian, &i)
	return i, err
}

// readInt64 reads a single int64 from the parser's reader.
func (p *Parser) readInt64() (int64, error) {
	var i int64
	err := binary.Read(p.r, binary.LittleEndian, &i)
	return i, err
}

// readUint64 reads a single uint64 from the parser's reader.
func (p *Parser) readUint64() (uint64, error) {
	var u uint64
	err := binary.Read(p.r, binary.LittleEndian, &u)
	return u, err
}

// readObjectID reads a single objectID from the parser's reader.
func (p *Parser) readObjectID() ([12]byte, error) {
	var id [12]byte
	b := make([]byte, 12)
	_, err := io.ReadFull(p.r, b)
	if err != nil {
		return id, err
	}
	copy(id[:], b)
	return id, nil
}

// readBoolean reads a single boolean from the parser's reader.
func (p *Parser) readBoolean() (bool, error) {
	var bv bool
	b, err := p.r.ReadByte()
	if err != nil {
		return false, err
	}
	switch b {
	case '\x00':
		bv = false
	case '\x01':
		bv = true
	default:
		return false, ErrCorruptDocument
	}
	return bv, nil
}

// ParseDocument parses an entire document from the parser's reader.
func (p *Parser) ParseDocument() (*ast.Document, error) {
	doc := new(ast.Document)
	// Lex document length
	l, err := p.readInt32()
	if err != nil {
		return nil, err
	}
	doc.Length = l
	// Lex and parse each item of the list
	elist, err := p.ParseEList()
	if err != nil {
		return nil, err
	}
	doc.EList = elist

	// ensure the document ends with \x00
	eol, err := p.r.ReadByte()
	if err != nil {
		return nil, err
	}
	if eol != '\x00' {
		return nil, ErrCorruptDocument
	}

	return doc, nil
}

// ParseEList parses an entire element list from the parser's reader.
func (p *Parser) ParseEList() ([]ast.Element, error) {
	var element ast.Element
	var err error
	var idents []byte
	list := make([]ast.Element, 0)

	for {
		idents, err = p.r.Peek(1)
		if err != nil {
			return list, err
		}

		if idents[0] == '\x00' {
			break
		}

		element, err = p.ParseElement()
		if err != nil {
			return list, err
		}
		list = append(list, element)
	}
	return list, nil
}

// ParseElement parses an element from the parser's reader.
func (p *Parser) ParseElement() (ast.Element, error) {
	var ident byte
	var err error
	var key *ast.ElementKeyName
	var el ast.Element

	ident, err = p.r.ReadByte()
	if err != nil {
		return nil, err
	}
	if ident == '\x00' {
		return nil, nil
	}

	key, err = p.ParseEName()
	if err != nil {
		return nil, err
	}

	switch ident {
	case '\x01':
		f, err := p.ParseDouble()
		if err != nil {
			return nil, err
		}
		el = &ast.FloatElement{
			Name:   key,
			Double: f,
		}
	case '\x02':
		str, err := p.ParseString()
		if err != nil {
			return nil, err
		}
		el = &ast.StringElement{
			Name:   key,
			String: str,
		}
	case '\x03':
		doc, err := p.ParseDocument()
		if err != nil {
			return nil, err
		}
		el = &ast.DocumentElement{
			Name:     key,
			Document: doc,
		}
	case '\x04':
		doc, err := p.ParseDocument()
		if err != nil {
			return nil, err
		}
		el = &ast.ArrayElement{
			Name:  key,
			Array: doc,
		}
	case '\x05':
		bin, err := p.ParseBinary()
		if err != nil {
			return nil, err
		}
		el = &ast.BinaryElement{
			Name:   key,
			Binary: bin,
		}
	case '\x06':
		el = &ast.UndefinedElement{
			Name: key,
		}
	case '\x07':
		id, err := p.readObjectID()
		if err != nil {
			return nil, err
		}
		el = &ast.ObjectIDElement{
			Name: key,
			ID:   id,
		}
	case '\x08':
		bl, err := p.readBoolean()
		if err != nil {
			return nil, err
		}
		el = &ast.BoolElement{
			Name: key,
			Bool: bl,
		}
	case '\x09':
		i64, err := p.readInt64()
		if err != nil {
			return nil, err
		}
		el = &ast.DateTimeElement{
			Name:     key,
			DateTime: i64,
		}
	case '\x0A':
		el = &ast.NullElement{
			Name: key,
		}
	case '\x0B':
		pattern, err := p.ParseCString()
		if err != nil {
			return nil, err
		}
		options, err := p.ParseCString()
		if err != nil {
			return nil, err
		}
		el = &ast.RegexElement{
			Name:         key,
			RegexPattern: &ast.CString{String: pattern},
			RegexOptions: &ast.CString{String: options},
		}
	case '\x0C':
		str, err := p.ParseString()
		if err != nil {
			return nil, err
		}
		pointer, err := p.readObjectID()
		if err != nil {
			return nil, err
		}
		el = &ast.DBPointerElement{
			Name:    key,
			String:  str,
			Pointer: pointer,
		}
	case '\x0D':
		str, err := p.ParseString()
		if err != nil {
			return nil, err
		}
		el = &ast.JavaScriptElement{
			Name:   key,
			String: str,
		}
	case '\x0E':
		str, err := p.ParseString()
		if err != nil {
			return nil, err
		}
		el = &ast.SymbolElement{
			Name:   key,
			String: str,
		}
	case '\x0F':
		cws, err := p.ParseCodeWithScope()
		if err != nil {
			return nil, err
		}
		el = &ast.CodeWithScopeElement{
			Name:          key,
			CodeWithScope: cws,
		}
	case '\x10':
		i, err := p.readInt32()
		if err != nil {
			return nil, err
		}
		el = &ast.Int32Element{
			Name:  key,
			Int32: i,
		}
	case '\x11':
		u, err := p.readUint64()
		if err != nil {
			return nil, err
		}
		el = &ast.TimestampElement{
			Name:      key,
			Timestamp: u,
		}
	case '\x12':
		i, err := p.readInt64()
		if err != nil {
			return nil, err
		}
		el = &ast.Int64Element{
			Name:  key,
			Int64: i,
		}
	case '\x13':
		l, err := p.readUint64()
		if err != nil {
			return nil, err
		}
		h, err := p.readUint64()
		if err != nil {
			return nil, err
		}
		d := decimal.NewDecimal128(h, l)
		el = &ast.DecimalElement{
			Name:       key,
			Decimal128: d,
		}
	case '\xFF':
		el = &ast.MinKeyElement{
			Name: key,
		}
	case '\x7F':
		el = &ast.MaxKeyElement{
			Name: key,
		}
	}

	return el, nil
}

// ParseEName parses an element's key from the parser's reader.
func (p *Parser) ParseEName() (*ast.ElementKeyName, error) {
	str, err := p.ParseCString()
	if err != nil {
		return nil, err
	}
	ekn := &ast.ElementKeyName{
		Key: str,
	}
	return ekn, nil
}

// ParseString parses a string from the parser's reader.
//
// TODO(skriptble): Should this be a read* method since it's returning a Go
// primitive? That would fit with the rest of the read* methods.
func (p *Parser) ParseString() (string, error) {
	l, err := p.readInt32()
	if err != nil {
		return "", err
	}

	if l > 0 {
		l--
	}

	b := make([]byte, l)
	_, err = io.ReadFull(p.r, b)
	if err != nil {
		return "", err
	}
	eol, err := p.r.ReadByte()
	if err != nil {
		return "", err
	}
	if eol != '\x00' {
		return "", ErrCorruptDocument
	}

	return string(b), nil
}

// ParseCString parses a c-style string from the parser's reader.
//
// TODO(skriptble): Should this be a read* method since it's returning a Go
// primitive? That would fit with the rest of the read* methods.
func (p *Parser) ParseCString() (string, error) {
	b, err := p.r.ReadBytes('\x00')
	if err != nil {
		return "", err
	}
	return string(b[:len(b)-1]), nil
}

// ParseBinary parses a binary node from the parser's reader.
func (p *Parser) ParseBinary() (*ast.Binary, error) {
	l, err := p.readInt32()
	if err != nil {
		return nil, err
	}

	bst, err := p.ParseSubtype()
	if err != nil {
		return nil, err
	}

	b := make([]byte, l)
	_, err = io.ReadFull(p.r, b)
	if err != nil {
		return nil, err
	}

	if bst == ast.SubtypeBinaryOld {
		if len(b) < 4 {
			// TODO(skriptble): Return a more informative error
			return nil, ErrCorruptDocument
		}
		b = b[4:]
	}

	bin := &ast.Binary{
		Subtype: bst,
		Data:    b,
	}

	return bin, err
}

// ParseSubtype parses the subtype for a binary node from the parser's reader.
func (p *Parser) ParseSubtype() (ast.BinarySubtype, error) {
	r, err := p.r.ReadByte()
	if err != nil {
		return 0, err
	}
	var bst ast.BinarySubtype

	switch r {
	case '\x00':
		bst = ast.SubtypeGeneric
	case '\x01':
		bst = ast.SubtypeFunction
	case '\x02':
		bst = ast.SubtypeBinaryOld
	case '\x03':
		bst = ast.SubtypeUUIDOld
	case '\x04':
		bst = ast.SubtypeUUID
	case '\x05':
		bst = ast.SubtypeMD5
	default:
		if r >= '\x80' {
			bst = ast.SubtypeUserDefined
		} else {
			return 0, ErrUnknownSubtype
		}
	}
	return bst, nil
}

// ParseDouble parses a float64 from the parser's reader.
//
// TODO(skriptble): This should be a read* method.
func (p *Parser) ParseDouble() (float64, error) {
	var bits uint64
	if err := binary.Read(p.r, binary.LittleEndian, &bits); err != nil {
		return 0, err
	}
	return math.Float64frombits(bits), nil
}

// ParseCodeWithScope parses a JavaScript Code with Scope node from the
// parser's reader.
func (p *Parser) ParseCodeWithScope() (*ast.CodeWithScope, error) {
	// TODO(skriptble): We should probably keep track of this length
	_, err := p.readInt32()
	if err != nil {
		return nil, err
	}
	str, err := p.ParseString()
	if err != nil {
		return nil, err
	}
	doc, err := p.ParseDocument()
	if err != nil {
		return nil, err
	}
	cws := &ast.CodeWithScope{
		String:   str,
		Document: doc,
	}
	return cws, nil
}
