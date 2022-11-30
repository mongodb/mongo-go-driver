// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package primitive

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/require"
)

type bigIntTestCase struct {
	s string

	h uint64
	l uint64

	bi  *big.Int
	exp int

	remark string
}

func parseBigInt(s string) *big.Int {
	bi, _ := new(big.Int).SetString(s, 10)
	return bi
}

var (
	one = big.NewInt(1)

	biMaxS  = new(big.Int).SetBytes([]byte{0x1, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})
	biNMaxS = new(big.Int).Neg(biMaxS)

	biOverflow  = new(big.Int).Add(biMaxS, one)
	biNOverflow = new(big.Int).Neg(biOverflow)

	bi12345  = parseBigInt("12345")
	biN12345 = parseBigInt("-12345")

	bi9_14  = parseBigInt("90123456789012")
	biN9_14 = parseBigInt("-90123456789012")

	bi9_34  = parseBigInt("9999999999999999999999999999999999")
	biN9_34 = parseBigInt("-9999999999999999999999999999999999")
)

var bigIntTestCases = []bigIntTestCase{
	{s: "12345", h: 0x3040000000000000, l: 12345, bi: bi12345},
	{s: "-12345", h: 0xB040000000000000, l: 12345, bi: biN12345},

	{s: "90123456.789012", h: 0x3034000000000000, l: 90123456789012, bi: bi9_14, exp: -6},
	{s: "-90123456.789012", h: 0xB034000000000000, l: 90123456789012, bi: biN9_14, exp: -6},
	{s: "9.0123456789012E+22", h: 0x3052000000000000, l: 90123456789012, bi: bi9_14, exp: 9},
	{s: "-9.0123456789012E+22", h: 0xB052000000000000, l: 90123456789012, bi: biN9_14, exp: 9},
	{s: "9.0123456789012E-8", h: 0x3016000000000000, l: 90123456789012, bi: bi9_14, exp: -21},
	{s: "-9.0123456789012E-8", h: 0xB016000000000000, l: 90123456789012, bi: biN9_14, exp: -21},

	{s: "9999999999999999999999999999999999", h: 3477321013416265664, l: 4003012203950112767, bi: bi9_34},
	{s: "-9999999999999999999999999999999999", h: 12700693050271041472, l: 4003012203950112767, bi: biN9_34},
	{s: "0.9999999999999999999999999999999999", h: 3458180714999941056, l: 4003012203950112767, bi: bi9_34, exp: -34},
	{s: "-0.9999999999999999999999999999999999", h: 12681552751854716864, l: 4003012203950112767, bi: biN9_34, exp: -34},
	{s: "99999999999999999.99999999999999999", h: 3467750864208103360, l: 4003012203950112767, bi: bi9_34, exp: -17},
	{s: "-99999999999999999.99999999999999999", h: 12691122901062879168, l: 4003012203950112767, bi: biN9_34, exp: -17},
	{s: "9.999999999999999999999999999999999E+35", h: 3478446913323108288, l: 4003012203950112767, bi: bi9_34, exp: 2},
	{s: "-9.999999999999999999999999999999999E+35", h: 12701818950177884096, l: 4003012203950112767, bi: biN9_34, exp: 2},
	{s: "9.999999999999999999999999999999999E+40", h: 3481261663090214848, l: 4003012203950112767, bi: bi9_34, exp: 7},
	{s: "-9.999999999999999999999999999999999E+40", h: 12704633699944990656, l: 4003012203950112767, bi: biN9_34, exp: 7},
	{s: "99999999999999999999999999999.99999", h: 3474506263649159104, l: 4003012203950112767, bi: bi9_34, exp: -5},
	{s: "-99999999999999999999999999999.99999", h: 12697878300503934912, l: 4003012203950112767, bi: biN9_34, exp: -5},

	{s: "1.038459371706965525706099265844019E-6143", remark: "subnormal", h: 0x333333333333, l: 0x3333333333333333, bi: parseBigInt("10384593717069655257060992658440190"), exp: MinDecimal128Exp - 1},
	{s: "-1.038459371706965525706099265844019E-6143", remark: "subnormal", h: 0x8000333333333333, l: 0x3333333333333333, bi: parseBigInt("-10384593717069655257060992658440190"), exp: MinDecimal128Exp - 1},

	{s: "rounding overflow 1", remark: "overflow", bi: parseBigInt("103845937170696552570609926584401910"), exp: MaxDecimal128Exp},
	{s: "rounding overflow 2", remark: "overflow", bi: parseBigInt("103845937170696552570609926584401910"), exp: MaxDecimal128Exp},

	{s: "subnormal overflow 1", remark: "overflow", bi: biMaxS, exp: MinDecimal128Exp - 1},
	{s: "subnormal overflow 2", remark: "overflow", bi: biNMaxS, exp: MinDecimal128Exp - 1},

	{s: "clamped overflow 1", remark: "overflow", bi: biMaxS, exp: MaxDecimal128Exp + 1},
	{s: "clamped overflow 2", remark: "overflow", bi: biNMaxS, exp: MaxDecimal128Exp + 1},

	{s: "biMaxS+1 overflow", remark: "overflow", bi: biOverflow, exp: MaxDecimal128Exp},
	{s: "biNMaxS-1 overflow", remark: "overflow", bi: biNOverflow, exp: MaxDecimal128Exp},

	{s: "NaN", h: 0x7c00000000000000, l: 0, remark: "NaN"},
	{s: "Infinity", h: 0x7800000000000000, l: 0, remark: "Infinity"},
	{s: "-Infinity", h: 0xf800000000000000, l: 0, remark: "-Infinity"},
}

func TestDecimal128_BigInt(t *testing.T) {
	for _, c := range bigIntTestCases {
		t.Run(c.s, func(t *testing.T) {
			switch c.remark {
			case "NaN", "Infinity", "-Infinity":
				d128 := NewDecimal128(c.h, c.l)
				_, _, err := d128.BigInt()
				require.Error(t, err, "case %s", c.s)
			case "":
				d128 := NewDecimal128(c.h, c.l)
				bi, e, err := d128.BigInt()
				require.NoError(t, err, "case %s", c.s)
				require.Equal(t, 0, c.bi.Cmp(bi), "case %s e:%s a:%s", c.s, c.bi.String(), bi.String())
				require.Equal(t, c.exp, e, "case %s", c.s, d128.String())
			}
		})
	}
}

func TestParseDecimal128FromBigInt(t *testing.T) {
	for _, c := range bigIntTestCases {
		switch c.remark {
		case "overflow":
			d128, ok := ParseDecimal128FromBigInt(c.bi, c.exp)
			require.Equal(t, false, ok, "case %s %s", c.s, d128.String(), c.remark)
		case "", "rounding", "subnormal", "clamped":
			d128, ok := ParseDecimal128FromBigInt(c.bi, c.exp)
			require.Equal(t, true, ok, "case %s", c.s)
			require.Equal(t, c.s, d128.String(), "case %s", c.s)

			require.Equal(t, c.h, d128.h, "case %s", c.s, d128.l)
			require.Equal(t, c.l, d128.l, "case %s", c.s, d128.h)
		}
	}
}

func TestParseDecimal128(t *testing.T) {
	cases := make([]bigIntTestCase, 0, len(bigIntTestCases))
	cases = append(cases, bigIntTestCases...)
	cases = append(cases,
		bigIntTestCase{s: "-0001231.453454000000565600000000E-21", h: 0xafe6000003faa269, l: 0x81cfeceaabdb1800},
		bigIntTestCase{s: "12345E+21", h: 0x306a000000000000, l: 12345},
		bigIntTestCase{s: "0.10000000000000000000000000000000000000000001", remark: "parse fail"},
		bigIntTestCase{s: ".125e1", h: 0x303c000000000000, l: 125},
		bigIntTestCase{s: ".125", h: 0x303a000000000000, l: 125},
		bigIntTestCase{s: "", remark: "parse fail"})

	for _, c := range cases {
		t.Run(c.s, func(t *testing.T) {
			switch c.remark {
			case "overflow", "parse fail":
				_, err := ParseDecimal128(c.s)
				require.Error(t, err)
			case "", "rounding", "subnormal", "clamped", "NaN", "Infinity", "-Infinity":
				d128, err := ParseDecimal128(c.s)
				require.NoError(t, err)

				require.Equal(t, c.h, d128.h, "case %s", c.s, d128.l)
				require.Equal(t, c.l, d128.l, "case %s", c.s, d128.h)
			}
		})
	}
}

func TestDecimal128_JSON(t *testing.T) {
	t.Run("roundTrip", func(t *testing.T) {
		decimal := NewDecimal128(0x3040000000000000, 12345)
		bytes, err := json.Marshal(decimal)
		assert.Nil(t, err, "json.Marshal error: %v", err)
		got := NewDecimal128(0, 0)
		err = json.Unmarshal(bytes, &got)
		assert.Nil(t, err, "json.Unmarshal error: %v", err)
		assert.Equal(t, decimal.h, got.h, "expected h: %v got: %v", decimal.h, got.h)
		assert.Equal(t, decimal.l, got.l, "expected l: %v got: %v", decimal.l, got.l)
	})
	t.Run("unmarshal extendedJSON", func(t *testing.T) {
		want := NewDecimal128(0x3040000000000000, 12345)
		extJSON := fmt.Sprintf(`{"$numberDecimal": %q}`, want.String())

		got := NewDecimal128(0, 0)
		err := json.Unmarshal([]byte(extJSON), &got)
		assert.Nil(t, err, "json.Unmarshal error: %v", err)
		assert.Equal(t, want.h, got.h, "expected h: %v got: %v", want.h, got.h)
		assert.Equal(t, want.l, got.l, "expected l: %v got: %v", want.l, got.l)
	})
	t.Run("unmarshal null", func(t *testing.T) {
		want := NewDecimal128(0, 0)
		extJSON := `null`

		got := NewDecimal128(0, 0)
		err := json.Unmarshal([]byte(extJSON), &got)
		assert.Nil(t, err, "json.Unmarshal error: %v", err)
		assert.Equal(t, want.h, got.h, "expected h: %v got: %v", want.h, got.h)
		assert.Equal(t, want.l, got.l, "expected l: %v got: %v", want.l, got.l)
	})
	t.Run("unmarshal", func(t *testing.T) {
		cases := make([]bigIntTestCase, 0, len(bigIntTestCases))
		cases = append(cases, bigIntTestCases...)
		cases = append(cases,
			bigIntTestCase{s: "-0001231.453454000000565600000000E-21", h: 0xafe6000003faa269, l: 0x81cfeceaabdb1800},
			bigIntTestCase{s: "12345E+21", h: 0x306a000000000000, l: 12345},
			bigIntTestCase{s: "0.10000000000000000000000000000000000000000001", remark: "parse fail"},
			bigIntTestCase{s: ".125e1", h: 0x303c000000000000, l: 125},
			bigIntTestCase{s: ".125", h: 0x303a000000000000, l: 125})

		for _, c := range cases {
			t.Run(c.s, func(t *testing.T) {
				input := fmt.Sprintf(`{"foo": %q}`, c.s)
				var got map[string]Decimal128
				err := json.Unmarshal([]byte(input), &got)

				switch c.remark {
				case "overflow", "parse fail":
					assert.NotNil(t, err, "expected Unmarshal error, got nil")
				default:
					assert.Nil(t, err, "Unmarshal error: %v", err)
					gotDecimal := got["foo"]
					assert.Equal(t, c.h, gotDecimal.h, "expected h: %v got: %v", c.h, gotDecimal.l)
					assert.Equal(t, c.l, gotDecimal.l, "expected l: %v got: %v", c.l, gotDecimal.h)
				}
			})
		}
	})
}
