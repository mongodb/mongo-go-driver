// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"unicode"
)

type jsonTokenType byte

const (
	beginObjectTokenType = jsonTokenType(iota)
	endObjectTokenType
	beginArrayTokenType
	endArrayTokenType
	colonTokenType
	commaTokenType
	int32TokenType
	int64TokenType
	doubleTokenType
	stringTokenType
	boolTokenType
	nullTokenType
	eofTokenType
)

type jsonToken struct {
	t jsonTokenType
	v interface{}
	p int
}

type jsonScanner struct {
	r   io.Reader
	buf []byte
	pos int
}

// nextToken returns the next JSON token if one exists. A token is a character
// of the JSON grammar, a number, a string, or a literal.
func (js *jsonScanner) nextToken() (*jsonToken, error) {
	c, err := js.readNextByte()

	// keep reading until a non-space is encountered (break on read error or EOF)
	for ; isWhiteSpace(c) && err == nil; c, err = js.readNextByte() {
	}

	if err == io.EOF {
		return &jsonToken{t: eofTokenType}, nil
	} else if err != nil {
		return nil, err
	}

	// switch on the character
	switch c {
	case '{':
		return &jsonToken{t: beginObjectTokenType, p: js.pos-1}, nil
	case '}':
		return &jsonToken{t: endObjectTokenType, p: js.pos-1}, nil
	case '[':
		return &jsonToken{t: beginArrayTokenType, p: js.pos-1}, nil
	case ']':
		return &jsonToken{t: endArrayTokenType, p: js.pos-1}, nil
	case ':':
		return &jsonToken{t: colonTokenType, p: js.pos-1}, nil
	case ',':
		return &jsonToken{t: commaTokenType, p: js.pos-1}, nil
	case '"': // RFC-8259 only allows for double quotes (") not single (')
		return js.scanString()
	default:
		// check if it's a number
		if c == '-' || isDigit(c) {
			return js.scanNumber(c)
		} else if c == 't' || c == 'f' || c == 'n' {
			// maybe a literal
			return js.scanLiteral(c)
		} else {
			return nil, errors.New(fmt.Sprintf("invalid JSON input. Position: %d. Character: %c", js.pos-1, c))
		}
	}
}

// readNextByte attempts to read the next byte from the buffer. If the buffer
// has been exhausted, this function calls readIntoBuf, thus refilling the
// buffer and resetting the read position to 0
func (js *jsonScanner) readNextByte() (byte, error) {
	if js.pos >= len(js.buf) {
		err := js.readIntoBuf()

		if err != nil {
			return 0, err
		}
	}

	b := js.buf[js.pos]
	js.pos++

	return b, nil
}

// readIntoBuf reads up to 512 bytes from the scanner's io.Reader into the buffer
func (js *jsonScanner) readIntoBuf() error {
	if cap(js.buf) == 0 {
		js.buf = make([]byte, 0, 512)
	}

	n, err := js.r.Read(js.buf[:cap(js.buf)])
	js.buf = js.buf[:n]
	js.pos = 0

	return err
}

func isWhiteSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n'
}

func isDigit(c byte) bool {
	return unicode.IsDigit(rune(c))
}

func isValueTerminator(c byte) bool {
	return c == ',' || c == '}' || c == ']' || isWhiteSpace(c)
}

// scanString reads from an opening '"' to a closing '"' and handles escaped characters
func (js *jsonScanner) scanString() (*jsonToken, error) {
	var b bytes.Buffer
	var c byte
	var err error

	p := js.pos - 1

	for {
		c, err = js.readNextByte()
		if err != nil {
			if err == io.EOF {
				return nil, errors.New("end of file in JSON string")
			}
			return nil, err
		}

		switch c {
		case '\\':
			c, err = js.readNextByte()
			switch c {
			case '"', '\\', '/', '\'':
				b.WriteByte(c)
			case 'b':
				b.WriteByte('\b')
			case 'f':
				b.WriteByte('\f')
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case 'u':
				u1, err := js.readNextByte()
				u2, err := js.readNextByte()
				u3, err := js.readNextByte()
				u4, err := js.readNextByte()
				if err == nil {
					// convert this
					hexString := []byte{u1, u2, u3, u4}
					decoded := make([]byte, 2)
					hex.Decode(decoded, hexString)
					b.Write(decoded)
				}
			default:
				return nil, errors.New(fmt.Sprintf("invalid escape sequence in JSON string '\\%c'", c))
			}
		case '"':
			return &jsonToken{t: stringTokenType, v: b.String(), p: p}, nil
		default:
			b.WriteByte(c)
		}
	}
}

// scanLiteral reads an unquoted sequence of characters and determines if it is one of
// three valid JSON literals (true, false, null); if so, it returns the appropriate
// jsonToken; otherwise, it returns an error
func (js *jsonScanner) scanLiteral(first byte) (*jsonToken, error) {
	p := js.pos - 1

	c2, err := js.readNextByte()
	c3, err := js.readNextByte()
	c4, err := js.readNextByte()

	if err != nil {
		return nil, err
	}

	c5, err := js.readNextByte()

	lit := []byte{first, c2, c3, c4}

	if bytes.Equal([]byte("true"), lit) && (isValueTerminator(c5) || err == io.EOF) {
		js.pos = int(math.Max(0, float64(js.pos-1)))
		return &jsonToken{t: boolTokenType, v: true, p: p}, nil
	} else if bytes.Equal([]byte("null"), lit) && (isValueTerminator(c5) || err == io.EOF) {
		js.pos = int(math.Max(0, float64(js.pos-1)))
		return &jsonToken{t: nullTokenType, v: nil, p: p}, nil
	} else if bytes.Equal([]byte("fals"), lit) {
		if c5 == 'e' {
			c5, err = js.readNextByte()

			if isValueTerminator(c5) || err == io.EOF {
				js.pos = int(math.Max(0, float64(js.pos-1)))
				return &jsonToken{t: boolTokenType, v: false, p: p}, nil
			}
		}
	}

	return nil, errors.New(fmt.Sprintf("invalid JSON literal. Position: %d", js.pos-4))
}

type numberScanState byte

const (
	sawLeadingMinus = iota
	sawLeadingZero
	sawIntegerDigits
	sawDecimalPoint
	sawFractionDigits
	sawExponentLetter
	sawExponentSign
	sawExponentDigits
	doneNumberState
	invalidNumberState
)

// scanNumber reads a JSON number (according to RFC-8259)
func (js *jsonScanner) scanNumber(first byte) (*jsonToken, error) {
	var b bytes.Buffer
	var s numberScanState
	var c byte
	var err error

	t := int64TokenType // assume it's an int64 until the type can be determined
	start := js.pos - 1

	b.WriteByte(first)

	switch first {
	case '-':
		s = sawLeadingMinus
	case '0':
		s = sawLeadingZero
	default:
		s = sawIntegerDigits
	}

	for {
		c, err = js.readNextByte()

		if err != nil && err != io.EOF {
			return nil, err
		}

		switch s {
		case sawLeadingMinus:
			switch c {
			case '0':
				s = sawLeadingZero
				b.WriteByte(c)
			default:
				if isDigit(c) {
					s = sawIntegerDigits
					b.WriteByte(c)
				} else {
					s = invalidNumberState
				}
			}
		case sawLeadingZero:
			switch c {
			case '.':
				s = sawDecimalPoint
				b.WriteByte(c)
			case 'e', 'E':
				s = sawExponentLetter
				b.WriteByte(c)
			case '}', ']', ',':
				s = doneNumberState
			default:
				if isWhiteSpace(c) || err == io.EOF {
					s = doneNumberState
				} else {
					s = invalidNumberState
				}
			}
		case sawIntegerDigits:
			switch c {
			case '.':
				s = sawDecimalPoint
				b.WriteByte(c)
			case 'e', 'E':
				s = sawExponentLetter
				b.WriteByte(c)
			case '}', ']', ',':
				s = doneNumberState
			default:
				if isWhiteSpace(c) || err == io.EOF {
					s = doneNumberState
				} else if isDigit(c) {
					s = sawIntegerDigits
					b.WriteByte(c)
				} else {
					s = invalidNumberState
				}
			}
		case sawDecimalPoint:
			t = doubleTokenType
			if isDigit(c) {
				s = sawFractionDigits
				b.WriteByte(c)
			} else {
				s = invalidNumberState
			}
		case sawFractionDigits:
			switch c {
			case 'e', 'E':
				s = sawExponentLetter
				b.WriteByte(c)
			case '}', ']', ',':
				s = doneNumberState
			default:
				if isWhiteSpace(c) || err == io.EOF {
					s = doneNumberState
				} else if isDigit(c) {
					s = sawFractionDigits
					b.WriteByte(c)
				} else {
					s = invalidNumberState
				}
			}
		case sawExponentLetter:
			t = doubleTokenType
			switch c {
			case '+', '-':
				s = sawExponentSign
				b.WriteByte(c)
			default:
				if isDigit(c) {
					s = sawExponentDigits
					b.WriteByte(c)
				} else {
					s = invalidNumberState
				}
			}
		case sawExponentSign:
			if isDigit(c) {
				s = sawExponentDigits
				b.WriteByte(c)
			} else {
				s = invalidNumberState
			}
		case sawExponentDigits:
			switch c {
			case '}', ']', ',':
				s = doneNumberState
			default:
				if isWhiteSpace(c) || err == io.EOF {
					s = doneNumberState
				} else if isDigit(c) {
					s = sawExponentDigits
					b.WriteByte(c)
				} else {
					s = invalidNumberState
				}
			}
		}

		switch s {
		case invalidNumberState:
			return nil, errors.New(fmt.Sprintf("invalid JSON number. Position: %d", start))
		case doneNumberState:
			js.pos = int(math.Max(0, float64(js.pos-1)))
			if t == doubleTokenType {
				v, err := strconv.ParseFloat(b.String(), 64)
				if err != nil {
					return nil, err
				}

				return &jsonToken{t:t, v: v, p: start}, nil
			}

			v, err := strconv.ParseInt(b.String(), 10, 64)
			if err != nil {
				return nil, err
			}

			if v < math.MinInt32 || v > math.MaxInt32 {
				return &jsonToken{t: t, v: v, p: start}, nil
			}

			return &jsonToken{t: int32TokenType, v: int32(v), p: start}, nil
		}
	}
}
