// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package decimal128

import (
	"strconv"
)

// These constants are the maximum and minimum values for the exponent field in a decimal128 value.
const (
	MaxDecimal128Exp = 6111
	MinDecimal128Exp = -6176
)

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

// String returns a string representation of the decimal value.
func String(h, l uint64) string {
	var posSign int      // positive sign
	var exp int          // exponent
	var high, low uint64 // significand high/low

	if h>>63&1 == 0 {
		posSign = 1
	}

	switch h >> 58 & (1<<5 - 1) {
	case 0x1F:
		return "NaN"
	case 0x1E:
		return "-Infinity"[posSign:]
	}

	low = l
	if h>>61&3 == 3 {
		// Bits: 1*sign 2*ignored 14*exponent 111*significand.
		// Implicit 0b100 prefix in significand.
		exp = int(h >> 47 & (1<<14 - 1))
		// Spec says all of these values are out of range.
		high, low = 0, 0
	} else {
		// Bits: 1*sign 14*exponent 113*significand
		exp = int(h >> 49 & (1<<14 - 1))
		high = h & (1<<49 - 1)
	}
	exp += MinDecimal128Exp

	// Would be handled by the logic below, but that's trivial and common.
	if high == 0 && low == 0 && exp == 0 {
		return "-0"[posSign:]
	}

	var repr [48]byte // Loop 5 times over 9 digits plus dot, negative sign, and leading zero.
	last := len(repr)
	i := len(repr)
	dot := len(repr) + exp
	var rem uint32
Loop:
	for d9 := 0; d9 < 5; d9++ {
		high, low, rem = divmod(high, low, 1e9)
		for d1 := 0; d1 < 9; d1++ {
			// Handle "-0.0", "0.00123400", "-1.00E-6", "1.050E+3", etc.
			if i < len(repr) && (dot == i || low == 0 && high == 0 && rem > 0 && rem < 10 && (dot < i-6 || exp > 0)) {
				exp += len(repr) - i
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
			if low == 0 && high == 0 && rem == 0 && i == len(repr)-1 && (dot < i-5 || exp > 0) {
				last = i
				break Loop
			}
			if c != '0' {
				last = i
			}
			// Break early. Works without it, but why.
			if dot > i && low == 0 && high == 0 && rem == 0 {
				break Loop
			}
		}
	}
	repr[last-1] = '-'
	last--

	if exp > 0 {
		return string(repr[last+posSign:]) + "E+" + strconv.Itoa(exp)
	}
	if exp < 0 {
		return string(repr[last+posSign:]) + "E" + strconv.Itoa(exp)
	}
	return string(repr[last+posSign:])
}
