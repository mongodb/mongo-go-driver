// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
//
// Based on gopkg.in/mgo.v2/bson by Gustavo Niemeyer
// See THIRD-PARTY-NOTICES for original license terms.

package primitive

import (
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

const (
	MaxDecimal128Exp = 6111
	MinDecimal128Exp = -6176
)

// Decimal128 holds decimal128 BSON values.
type Decimal128 struct {
	h, l uint64
}

// NewDecimal128 creates a Decimal128 using the provide high and low uint64s.
func NewDecimal128(h, l uint64) Decimal128 {
	return Decimal128{h: h, l: l}
}

// GetBytes returns the underlying bytes of the BSON decimal value as two uint64 values. The first
// contains the most first 8 bytes of the value and the second contains the latter.
func (d Decimal128) GetBytes() (uint64, uint64) {
	return d.h, d.l
}

// String returns a string representation of the decimal value.
func (d Decimal128) String() string {
	var pos int     // positive sign
	var e int       // exponent
	var h, l uint64 // significand high/low

	if d.h>>63&1 == 0 {
		pos = 1
	}

	switch d.h >> 58 & (1<<5 - 1) {
	case 0x1F:
		return "NaN"
	case 0x1E:
		return "-Infinity"[pos:]
	}

	l = d.l
	if d.h>>61&3 == 3 {
		// Bits: 1*sign 2*ignored 14*exponent 111*significand.
		// Implicit 0b100 prefix in significand.
		e = int(d.h>>47&(1<<14-1)) + MinDecimal128Exp
		//h = 4<<47 | d.h&(1<<47-1)
		// Spec says all of these values are out of range.
		h, l = 0, 0
	} else {
		// Bits: 1*sign 14*exponent 113*significand
		e = int(d.h>>49&(1<<14-1)) + MinDecimal128Exp
		h = d.h & (1<<49 - 1)
	}

	// Would be handled by the logic below, but that's trivial and common.
	if h == 0 && l == 0 && e == 0 {
		return "-0"[pos:]
	}

	var repr [48]byte // Loop 5 times over 9 digits plus dot, negative sign, and leading zero.
	var last = len(repr)
	var i = len(repr)
	var dot = len(repr) + e
	var rem uint32
Loop:
	for d9 := 0; d9 < 5; d9++ {
		h, l, rem = divmod(h, l, 1e9)
		for d1 := 0; d1 < 9; d1++ {
			// Handle "-0.0", "0.00123400", "-1.00E-6", "1.050E+3", etc.
			if i < len(repr) && (dot == i || l == 0 && h == 0 && rem > 0 && rem < 10 && (dot < i-6 || e > 0)) {
				e += len(repr) - i
				i--
				repr[i] = '.'
				last = i - 1
				dot = len(repr) // Unmark.
			}
			c := '0' + byte(rem%10)
			rem /= 10
			i--
			repr[i] = c
			// Handle "0E+3", "1E+3", etc.
			if l == 0 && h == 0 && rem == 0 && i == len(repr)-1 && (dot < i-5 || e > 0) {
				last = i
				break Loop
			}
			if c != '0' {
				last = i
			}
			// Break early. Works without it, but why.
			if dot > i && l == 0 && h == 0 && rem == 0 {
				break Loop
			}
		}
	}
	repr[last-1] = '-'
	last--

	if e > 0 {
		return string(repr[last+pos:]) + "E+" + strconv.Itoa(e)
	}
	if e < 0 {
		return string(repr[last+pos:]) + "E" + strconv.Itoa(e)
	}
	return string(repr[last+pos:])
}

// BigInt returns significand as big.Int and exponent, bi * 10 ^ exp.
func (d Decimal128) BigInt() (bi *big.Int, exp int, err error) {
	h, l := d.GetBytes()
	var pos int // positive sign

	if h>>63&1 == 0 {
		pos = 1
	}

	switch h >> 58 & (1<<5 - 1) {
	case 0x1F:
		return nil, 0, fmt.Errorf("cannot parse NaN as a *big.Int")
	case 0x1E:
		return nil, 0, fmt.Errorf("cannot parse %s as a *big.Int", "-Infinity"[pos:])
	}

	if h>>61&3 == 3 {
		// Bits: 1*sign 2*ignored 14*exponent 111*significand.
		// Implicit 0b100 prefix in significand.
		exp = int(h>>47&(1<<14-1)) + MinDecimal128Exp
		//h = 4<<47 | d.h&(1<<47-1)
		// Spec says all of these values are out of range.
		h, l = 0, 0
	} else {
		// Bits: 1*sign 14*exponent 113*significand
		exp = int(h>>49&(1<<14-1)) + MinDecimal128Exp
		h = h & (1<<49 - 1)
	}

	// Would be handled by the logic below, but that's trivial and common.
	if h == 0 && l == 0 && exp == 0 {
		if pos == 1 {
			return new(big.Int), 0, nil
		}
		return new(big.Int), 0, nil
	}

	bi = big.NewInt(0)
	const host32bit = ^uint(0)>>32 == 0
	if host32bit {
		bi.SetBits([]big.Word{big.Word(l), big.Word(l >> 32), big.Word(h), big.Word(h >> 32)})
	} else {
		bi.SetBits([]big.Word{big.Word(l), big.Word(h)})
	}

	if pos == 0 {
		return bi.Neg(bi), exp, nil
	}
	return
}

// IsNaN returns whether d is NaN.
func (d Decimal128) IsNaN() bool {
	return d.h>>58&(1<<5-1) == 0x1F
}

// IsInf returns:
//
//   +1 d == Infinity
//    0 other case
//   -1 d == -Infinity
//
func (d Decimal128) IsInf() int {
	if d.h>>58&(1<<5-1) != 0x1E {
		return 0
	}

	if d.h>>63&1 == 0 {
		return 1
	} else {
		return -1
	}
}

func divmod(h, l uint64, div uint32) (qh, ql uint64, rem uint32) {
	div64 := uint64(div)
	a := h >> 32
	aq := a / div64
	ar := a % div64
	b := ar<<32 + h&(1<<32-1)
	bq := b / div64
	br := b % div64
	c := br<<32 + l>>32
	cq := c / div64
	cr := c % div64
	d := cr<<32 + l&(1<<32-1)
	dq := d / div64
	dr := d % div64
	return (aq<<32 | bq), (cq<<32 | dq), uint32(dr)
}

var dNaN = Decimal128{0x1F << 58, 0}
var dPosInf = Decimal128{0x1E << 58, 0}
var dNegInf = Decimal128{0x3E << 58, 0}

func dErr(s string) (Decimal128, error) {
	return dNaN, fmt.Errorf("cannot parse %q as a decimal128", s)
}

var regDecimal128 = regexp.MustCompile(`^(?P<int>[-+]?\d+)(?:\.(?P<dec>\d+))?(?:[Ee](?P<exp>[-+]?\d+))?$`)

// ParseDecimal128 takes the given string and attempts to parse it into a valid
// Decimal128 value.
func ParseDecimal128(s string) (Decimal128, error) {
	matches := regDecimal128.FindStringSubmatch(s)
	if len(matches) == 0 {
		if s == "NaN" || s == "nan" || strings.EqualFold(s, "nan") {
			return dNaN, nil
		}
		if s == "Inf" || s == "inf" || strings.EqualFold(s, "inf") || strings.EqualFold(s, "infinity") {
			return dPosInf, nil
		}
		if s == "-Inf" || s == "-inf" || strings.EqualFold(s, "-inf") || strings.EqualFold(s, "-infinity") {
			return dNegInf, nil
		}
		return dErr(s)
	}

	ip := matches[1]
	dp := matches[2]
	ep := matches[3]

	var err error
	e := 0
	if ep != "" {
		e, err = strconv.Atoi(ep)
		if err != nil {
			return dErr(s)
		}
	}
	if dp != "" {
		e -= len(dp)
	}

	if len(strings.Trim(ip+dp, "-0")) > 35 {
		return dErr(s)
	}

	bi, ok := new(big.Int).SetString(ip+dp, 10)
	if !ok {
		return dErr(s)
	}

	d, ok := ParseDecimal128FromBigInt(bi, e)
	if !ok {
		return dErr(s)
	}
	return d, nil
}

var (
	ten  = big.NewInt(10)
	zero = new(big.Int)
	maxS = new(big.Int).SetBytes([]byte{0x1, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}) // 113 bits
)

// ParseDecimal128FromBigInt attempts to parse the given significand and exponent into a valid Decimal128 value.
func ParseDecimal128FromBigInt(bi *big.Int, exp int) (Decimal128, bool) {
	//copy
	if bi == nil {
		bi = new(big.Int)
	} else {
		bi = new(big.Int).Set(bi)
	}

	q := new(big.Int)
	r := new(big.Int)

	for bi.CmpAbs(maxS) == 1 {
		bi, _ = q.QuoRem(bi, ten, r)
		if r.Cmp(zero) != 0 {
			return Decimal128{}, false
		}
		exp++
		if exp > MaxDecimal128Exp {
			return Decimal128{}, false
		}
	}

	for exp < MinDecimal128Exp {
		// Subnormal.
		bi, _ = q.QuoRem(bi, ten, r)
		if r.Cmp(zero) != 0 {
			return Decimal128{}, false
		}
		exp++
	}
	for exp > MaxDecimal128Exp {
		// Clamped.
		bi.Mul(bi, ten)
		if bi.CmpAbs(maxS) == 1 {
			return Decimal128{}, false
		}
		exp--
	}

	b := bi.Bytes()
	var h, l uint64
	for i := 0; i < len(b); i++ {
		if i < len(b)-8 {
			h = h<<8 | uint64(b[i])
		} else {
			l = l<<8 | uint64(b[i])
		}
	}

	h |= uint64(exp-MinDecimal128Exp) & uint64(1<<14-1) << 49
	if bi.Sign() == -1 {
		h |= 1 << 63
	}

	return Decimal128{h: h, l: l}, true
}
