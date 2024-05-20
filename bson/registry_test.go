// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"errors"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.mongodb.org/mongo-driver/internal/assert"
)

// newTestRegistryBuilder creates a new empty RegistryBuilder.
func newTestRegistryBuilder() *RegistryBuilder {
	return &RegistryBuilder{
		typeEncoders:      make(map[reflect.Type]EncoderFactory),
		typeDecoders:      make(map[reflect.Type]DecoderFactory),
		interfaceEncoders: make(map[reflect.Type]EncoderFactory),
		interfaceDecoders: make(map[reflect.Type]DecoderFactory),
		typeMap:           make(map[Type]reflect.Type),
	}
}

func TestRegistryBuilder(t *testing.T) {
	t.Parallel()

	t.Run("Register", func(t *testing.T) {
		t.Parallel()

		fc1, fc2, fc3, fc4 := new(fakeCodec), new(fakeCodec), new(fakeCodec), new(fakeCodec)
		t.Run("interface", func(t *testing.T) {
			t.Parallel()

			t1f, t2f, t3f, t4f :=
				reflect.TypeOf((*testInterface1)(nil)).Elem(),
				reflect.TypeOf((*testInterface2)(nil)).Elem(),
				reflect.TypeOf((*testInterface3)(nil)).Elem(),
				reflect.TypeOf((*testInterface4)(nil)).Elem()

			var c1, c2, c3, c4 int
			ef1 := func() ValueEncoder {
				c1++
				return fc1
			}
			ef2 := func() ValueEncoder {
				c2++
				return fc2
			}
			ef3 := func() ValueEncoder {
				c3++
				return fc3
			}
			ef4 := func() ValueEncoder {
				c4++
				return fc4
			}

			ips := []struct {
				i  reflect.Type
				ef EncoderFactory
			}{
				{i: t1f, ef: ef1},
				{i: t2f, ef: ef2},
				{i: t1f, ef: ef3},
				{i: t3f, ef: ef2},
				{i: t4f, ef: ef4},
			}
			want := []interfaceValueEncoder{
				{i: t1f, ve: fc3}, {i: t2f, ve: fc2},
				{i: t3f, ve: fc2}, {i: t4f, ve: fc4},
			}

			rb := newTestRegistryBuilder()
			for _, ip := range ips {
				rb.RegisterInterfaceEncoder(ip.i, ip.ef)
			}
			reg := rb.Build()

			if !cmp.Equal(c1, 0) {
				t.Errorf("ef1 is called %d time(s); expected 0", c1)
			}
			if !cmp.Equal(c2, 1) {
				t.Errorf("ef2 is called %d time(s); expected 1", c2)
			}
			if !cmp.Equal(c3, 1) {
				t.Errorf("ef3 is called %d time(s); expected 1", c3)
			}
			if !cmp.Equal(c4, 1) {
				t.Errorf("ef4 is called %d time(s); expected 1", c4)
			}
			codecs, ok := reg.codecTypeMap[reflect.TypeOf((*fakeCodec)(nil))]
			if !cmp.Equal(len(reg.codecTypeMap), 1) || !cmp.Equal(ok, true) || len(codecs) != 3 {
				t.Errorf("codecs were not cached correctly")
			}
			got := make(map[reflect.Type]ValueEncoder)
			for _, e := range reg.interfaceEncoders {
				got[e.i] = e.ve
			}
			for _, s := range want {
				wantI, wantVe := s.i, s.ve
				gotVe, exists := got[wantI]
				if !exists {
					t.Errorf("Did not find type in the type registry: %v", wantI)
				}
				if !cmp.Equal(gotVe, wantVe, cmp.AllowUnexported(fakeCodec{})) {
					t.Errorf("codecs did not match: got %#v; want %#v", gotVe, wantVe)
				}
			}
		})
		t.Run("type", func(t *testing.T) {
			t.Parallel()

			ft1, ft2, ft3, ft4 :=
				reflect.TypeOf(fakeType1{}),
				reflect.TypeOf(fakeType2{}),
				reflect.TypeOf(fakeType3{}),
				reflect.TypeOf(fakeType4{})

			var c1, c2, c3, c4 int
			ef1 := func() ValueEncoder {
				c1++
				return fc1
			}
			ef2 := func() ValueEncoder {
				c2++
				return fc2
			}
			ef3 := func() ValueEncoder {
				c3++
				return fc3
			}
			ef4 := func() ValueEncoder {
				c4++
				return fc4
			}

			ips := []struct {
				i  reflect.Type
				ef EncoderFactory
			}{
				{i: ft1, ef: ef1},
				{i: ft2, ef: ef2},
				{i: ft1, ef: ef3},
				{i: ft3, ef: ef2},
				{i: ft4, ef: ef4},
			}
			want := []interfaceValueEncoder{
				{i: ft1, ve: fc3}, {i: ft2, ve: fc2},
				{i: ft3, ve: fc2}, {i: ft4, ve: fc4},
			}

			rb := newTestRegistryBuilder()
			for _, ip := range ips {
				rb.RegisterTypeEncoder(ip.i, ip.ef)
			}
			reg := rb.Build()

			if !cmp.Equal(c1, 0) {
				t.Errorf("ef1 is called %d time(s); expected 0", c1)
			}
			if !cmp.Equal(c2, 1) {
				t.Errorf("ef2 is called %d time(s); expected 1", c2)
			}
			if !cmp.Equal(c3, 1) {
				t.Errorf("ef3 is called %d time(s); expected 1", c3)
			}
			if !cmp.Equal(c4, 1) {
				t.Errorf("ef4 is called %d time(s); expected 1", c4)
			}
			codecs, ok := reg.codecTypeMap[reflect.TypeOf((*fakeCodec)(nil))]
			if !cmp.Equal(len(reg.codecTypeMap), 1) || !cmp.Equal(ok, true) || len(codecs) != 3 {
				t.Errorf("codecs were not cached correctly")
			}
			got := reg.typeEncoders
			for _, s := range want {
				wantI, wantVe := s.i, s.ve
				gotVe, exists := got.Load(wantI)
				if !exists {
					t.Errorf("type missing in registry: %v", wantI)
				}
				if !cmp.Equal(gotVe, wantVe, cmp.AllowUnexported(fakeCodec{})) {
					t.Errorf("codecs did not match: got %#v; want %#v", gotVe, wantVe)
				}
			}
		})
		t.Run("kind", func(t *testing.T) {
			t.Parallel()

			k1, k2, k3, k4 := reflect.Struct, reflect.Slice, reflect.Int, reflect.Map

			var c1, c2, c3, c4 int
			ef1 := func() ValueEncoder {
				c1++
				return fc1
			}
			ef2 := func() ValueEncoder {
				c2++
				return fc2
			}
			ef3 := func() ValueEncoder {
				c3++
				return fc3
			}
			ef4 := func() ValueEncoder {
				c4++
				return fc4
			}

			ips := []struct {
				k  reflect.Kind
				ef EncoderFactory
			}{
				{k: k1, ef: ef1},
				{k: k2, ef: ef2},
				{k: k1, ef: ef3},
				{k: k3, ef: ef2},
				{k: k4, ef: ef4},
			}
			want := []struct {
				k reflect.Kind
				c ValueEncoder
			}{
				{k1, fc3}, {k2, fc2}, {k4, fc4},
			}

			rb := newTestRegistryBuilder()
			for _, ip := range ips {
				rb.RegisterKindEncoder(ip.k, ip.ef)
			}
			reg := rb.Build()

			if !cmp.Equal(c1, 0) {
				t.Errorf("ef1 is called %d time(s); expected 0", c1)
			}
			if !cmp.Equal(c2, 1) {
				t.Errorf("ef2 is called %d time(s); expected 1", c2)
			}
			if !cmp.Equal(c3, 1) {
				t.Errorf("ef3 is called %d time(s); expected 1", c3)
			}
			if !cmp.Equal(c4, 1) {
				t.Errorf("ef4 is called %d time(s); expected 1", c4)
			}
			codecs, ok := reg.codecTypeMap[reflect.TypeOf((*fakeCodec)(nil))]
			if !cmp.Equal(len(reg.codecTypeMap), 1) || !cmp.Equal(ok, true) || len(codecs) != 3 {
				t.Errorf("codecs were not cached correctly")
			}
			got := reg.kindEncoders
			for _, s := range want {
				wantI, wantVe := s.k, s.c
				gotC := got[wantI]
				if !cmp.Equal(gotC, wantVe, cmp.AllowUnexported(fakeCodec{})) {
					t.Errorf("codecs did not match: got %#v, want %#v", gotC, wantVe)
				}
			}
		})
		t.Run("RegisterDefault", func(t *testing.T) {
			t.Parallel()

			t.Run("MapCodec", func(t *testing.T) {
				t.Parallel()

				codec := &fakeCodec{num: 1}
				codec2 := &fakeCodec{num: 2}
				rb := newTestRegistryBuilder()

				rb.RegisterKindEncoder(reflect.Map, func() ValueEncoder { return codec })
				reg := rb.Build()
				if got := reg.kindEncoders[reflect.Map]; got != codec {
					t.Errorf("map codec not properly set: got %#v, want %#v", got, codec)
				}

				rb.RegisterKindEncoder(reflect.Map, func() ValueEncoder { return codec2 })
				reg = rb.Build()
				if got := reg.kindEncoders[reflect.Map]; got != codec2 {
					t.Errorf("map codec not properly set: got %#v, want %#v", got, codec2)
				}
			})
			t.Run("StructCodec", func(t *testing.T) {
				t.Parallel()

				codec := &fakeCodec{num: 1}
				codec2 := &fakeCodec{num: 2}
				rb := newTestRegistryBuilder()

				rb.RegisterKindEncoder(reflect.Struct, func() ValueEncoder { return codec })
				reg := rb.Build()
				if got := reg.kindEncoders[reflect.Struct]; got != codec {
					t.Errorf("struct codec not properly set: got %#v, want %#v", got, codec)
				}

				rb.RegisterKindEncoder(reflect.Struct, func() ValueEncoder { return codec2 })
				reg = rb.Build()
				if got := reg.kindEncoders[reflect.Struct]; got != codec2 {
					t.Errorf("struct codec not properly set: got %#v, want %#v", got, codec2)
				}
			})
			t.Run("SliceCodec", func(t *testing.T) {
				t.Parallel()

				codec := &fakeCodec{num: 1}
				codec2 := &fakeCodec{num: 2}
				rb := newTestRegistryBuilder()

				rb.RegisterKindEncoder(reflect.Slice, func() ValueEncoder { return codec })
				reg := rb.Build()
				if got := reg.kindEncoders[reflect.Slice]; got != codec {
					t.Errorf("slice codec not properly set: got %#v, want %#v", got, codec)
				}

				rb.RegisterKindEncoder(reflect.Slice, func() ValueEncoder { return codec2 })
				reg = rb.Build()
				if got := reg.kindEncoders[reflect.Slice]; got != codec2 {
					t.Errorf("slice codec not properly set: got %#v, want %#v", got, codec2)
				}
			})
			t.Run("ArrayCodec", func(t *testing.T) {
				t.Parallel()

				codec := &fakeCodec{num: 1}
				codec2 := &fakeCodec{num: 2}
				rb := newTestRegistryBuilder()

				rb.RegisterKindEncoder(reflect.Array, func() ValueEncoder { return codec })
				reg := rb.Build()
				if got := reg.kindEncoders[reflect.Array]; got != codec {
					t.Errorf("slice codec not properly set: got %#v, want %#v", got, codec)
				}

				rb.RegisterKindEncoder(reflect.Array, func() ValueEncoder { return codec2 })
				reg = rb.Build()
				if got := reg.kindEncoders[reflect.Array]; got != codec2 {
					t.Errorf("slice codec not properly set: got %#v, want %#v", got, codec2)
				}
			})
		})
		t.Run("Lookup", func(t *testing.T) {
			t.Parallel()

			type Codec interface {
				ValueEncoder
				ValueDecoder
			}

			var (
				arrinstance     [12]int
				arr             = reflect.TypeOf(arrinstance)
				slc             = reflect.TypeOf(make([]int, 12))
				m               = reflect.TypeOf(make(map[string]int))
				strct           = reflect.TypeOf(struct{ Foo string }{})
				ft1             = reflect.PtrTo(reflect.TypeOf(fakeType1{}))
				ft2             = reflect.TypeOf(fakeType2{})
				ft3             = reflect.TypeOf(fakeType5(func(string, string) string { return "fakeType5" }))
				ti1             = reflect.TypeOf((*testInterface1)(nil)).Elem()
				ti2             = reflect.TypeOf((*testInterface2)(nil)).Elem()
				ti1Impl         = reflect.TypeOf(testInterface1Impl{})
				ti2Impl         = reflect.TypeOf(testInterface2Impl{})
				ti3             = reflect.TypeOf((*testInterface3)(nil)).Elem()
				ti3Impl         = reflect.TypeOf(testInterface3Impl{})
				ti3ImplPtr      = reflect.TypeOf((*testInterface3Impl)(nil))
				fc1, fc2        = &fakeCodec{num: 1}, &fakeCodec{num: 2}
				fsc, fslcc, fmc = new(fakeStructCodec), new(fakeSliceCodec), new(fakeMapCodec)
				pc              = &pointerCodec{}
			)

			fc1EncFac := func() ValueEncoder { return fc1 }
			fc2EncFac := func() ValueEncoder { return fc2 }
			fc3EncFac := func() ValueEncoder { return fc3 }
			fscEncFac := func() ValueEncoder { return fsc }
			fslccEncFac := func() ValueEncoder { return fslcc }
			fmcEncFac := func() ValueEncoder { return fmc }
			pcEncFac := func() ValueEncoder { return pc }

			fc1DecFac := func() ValueDecoder { return fc1 }
			fc2DecFac := func() ValueDecoder { return fc2 }
			fc3DecFac := func() ValueDecoder { return fc3 }
			fscDecFac := func() ValueDecoder { return fsc }
			fslccDecFac := func() ValueDecoder { return fslcc }
			fmcDecFac := func() ValueDecoder { return fmc }
			pcDecFac := func() ValueDecoder { return pc }

			reg := newTestRegistryBuilder().
				RegisterTypeEncoder(ft1, fc1EncFac).
				RegisterTypeEncoder(ft2, fc2EncFac).
				RegisterTypeEncoder(ti1, fc1EncFac).
				RegisterKindEncoder(reflect.Struct, fscEncFac).
				RegisterKindEncoder(reflect.Slice, fslccEncFac).
				RegisterKindEncoder(reflect.Array, fslccEncFac).
				RegisterKindEncoder(reflect.Map, fmcEncFac).
				RegisterKindEncoder(reflect.Ptr, pcEncFac).
				RegisterTypeDecoder(ft1, fc1DecFac).
				RegisterTypeDecoder(ft2, fc2DecFac).
				RegisterTypeDecoder(ti1, fc1DecFac). // values whose exact type is testInterface1 will use fc1 encoder
				RegisterKindDecoder(reflect.Struct, fscDecFac).
				RegisterKindDecoder(reflect.Slice, fslccDecFac).
				RegisterKindDecoder(reflect.Array, fslccDecFac).
				RegisterKindDecoder(reflect.Map, fmcDecFac).
				RegisterKindDecoder(reflect.Ptr, pcDecFac).
				RegisterInterfaceEncoder(ti2, fc2EncFac).
				RegisterInterfaceEncoder(ti3, fc3EncFac).
				RegisterInterfaceDecoder(ti2, fc2DecFac).
				RegisterInterfaceDecoder(ti3, fc3DecFac).
				Build()

			testCases := []struct {
				name      string
				t         reflect.Type
				wantcodec Codec
				wanterr   error
				testcache bool
			}{
				{
					"type registry (pointer)",
					ft1,
					fc1,
					nil,
					false,
				},
				{
					"type registry (non-pointer)",
					ft2,
					fc2,
					nil,
					false,
				},
				{
					// lookup an interface type and expect that the registered encoder is returned
					"interface with type encoder",
					ti1,
					fc1,
					nil,
					true,
				},
				{
					// lookup a type that implements an interface and expect that the default struct codec is returned
					"interface implementation with type encoder",
					ti1Impl,
					fsc,
					nil,
					false,
				},
				{
					// lookup an interface type and expect that the registered hook is returned
					"interface with hook",
					ti2,
					fc2,
					nil,
					false,
				},
				{
					// lookup a type that implements an interface and expect that the registered hook is returned
					"interface implementation with hook",
					ti2Impl,
					fc2,
					nil,
					false,
				},
				{
					// lookup a pointer to a type where the pointer implements an interface and expect that the
					// registered hook is returned
					"interface pointer to implementation with hook (pointer)",
					ti3ImplPtr,
					fc3,
					nil,
					false,
				},
				{
					"default struct codec (pointer)",
					reflect.PtrTo(strct),
					pc,
					nil,
					false,
				},
				{
					"default struct codec (non-pointer)",
					strct,
					fsc,
					nil,
					false,
				},
				{
					"default array codec",
					arr,
					fslcc,
					nil,
					false,
				},
				{
					"default slice codec",
					slc,
					fslcc,
					nil,
					false,
				},
				{
					"default map",
					m,
					fmc,
					nil,
					false,
				},
				{
					"map non-string key",
					reflect.TypeOf(map[int]int{}),
					fmc,
					nil,
					false,
				},
				{
					"No Codec Registered",
					ft3,
					nil,
					ErrNoEncoder{Type: ft3},
					false,
				},
			}

			allowunexported := cmp.AllowUnexported(fakeCodec{}, fakeStructCodec{}, fakeSliceCodec{}, fakeMapCodec{})
			comparepc := func(pc1, pc2 *pointerCodec) bool { return true }
			for _, tc := range testCases {
				tc := tc

				t.Run(tc.name, func(t *testing.T) {
					t.Parallel()

					t.Run("Encoder", func(t *testing.T) {
						t.Parallel()

						gotcodec, goterr := reg.LookupEncoder(tc.t)
						if !cmp.Equal(goterr, tc.wanterr, cmp.Comparer(assert.CompareErrors)) {
							t.Errorf("errors did not match: got %#v, want %#v", goterr, tc.wanterr)
						}
						if !cmp.Equal(gotcodec, tc.wantcodec, allowunexported, cmp.Comparer(comparepc)) {
							t.Errorf("codecs did not match: got %#v, want %#v", gotcodec, tc.wantcodec)
						}
					})
					t.Run("Decoder", func(t *testing.T) {
						t.Parallel()

						wanterr := tc.wanterr
						if ene, ok := tc.wanterr.(ErrNoEncoder); ok {
							wanterr = ErrNoDecoder(ene)
						}

						gotcodec, goterr := reg.LookupDecoder(tc.t)
						if !cmp.Equal(goterr, wanterr, cmp.Comparer(assert.CompareErrors)) {
							t.Errorf("errors did not match: got %#v, want %#v", goterr, wanterr)
						}
						if !cmp.Equal(gotcodec, tc.wantcodec, allowunexported, cmp.Comparer(comparepc)) {
							t.Errorf("codecs did not match: got %v: want %v", gotcodec, tc.wantcodec)
						}
					})
				})
			}
			t.Run("nil type", func(t *testing.T) {
				t.Parallel()

				t.Run("Encoder", func(t *testing.T) {
					t.Parallel()

					wanterr := ErrNoEncoder{Type: reflect.TypeOf(nil)}

					gotcodec, goterr := reg.LookupEncoder(nil)
					if !cmp.Equal(goterr, wanterr, cmp.Comparer(assert.CompareErrors)) {
						t.Errorf("errors did not match: got %#v, want %#v", goterr, wanterr)
					}
					if !cmp.Equal(gotcodec, nil, allowunexported, cmp.Comparer(comparepc)) {
						t.Errorf("codecs did not match: got %#v, want nil", gotcodec)
					}
				})
				t.Run("Decoder", func(t *testing.T) {
					t.Parallel()

					wanterr := ErrNoDecoder{Type: nil}

					gotcodec, goterr := reg.LookupDecoder(nil)
					if !cmp.Equal(goterr, wanterr, cmp.Comparer(assert.CompareErrors)) {
						t.Errorf("errors did not match: got %#v, want %#v", goterr, wanterr)
					}
					if !cmp.Equal(gotcodec, nil, allowunexported, cmp.Comparer(comparepc)) {
						t.Errorf("codecs did not match: got %v: want nil", gotcodec)
					}
				})
			})
			// lookup a type whose pointer implements an interface and expect that the registered hook is
			// returned
			t.Run("interface implementation with hook (pointer)", func(t *testing.T) {
				t.Parallel()

				t.Run("Encoder", func(t *testing.T) {
					t.Parallel()
					gotEnc, err := reg.LookupEncoder(ti3Impl)
					assert.Nil(t, err, "LookupEncoder error: %v", err)

					cae, ok := gotEnc.(*condAddrEncoder)
					assert.True(t, ok, "Expected CondAddrEncoder, got %T", gotEnc)
					if !cmp.Equal(cae.canAddrEnc, fc3, allowunexported, cmp.Comparer(comparepc)) {
						t.Errorf("expected canAddrEnc %#v, got %#v", cae.canAddrEnc, fc3)
					}
					if !cmp.Equal(cae.elseEnc, fsc, allowunexported, cmp.Comparer(comparepc)) {
						t.Errorf("expected elseEnc %#v, got %#v", cae.elseEnc, fsc)
					}
				})
				t.Run("Decoder", func(t *testing.T) {
					t.Parallel()

					gotDec, err := reg.LookupDecoder(ti3Impl)
					assert.Nil(t, err, "LookupDecoder error: %v", err)

					cad, ok := gotDec.(*condAddrDecoder)
					assert.True(t, ok, "Expected CondAddrDecoder, got %T", gotDec)
					if !cmp.Equal(cad.canAddrDec, fc3, allowunexported, cmp.Comparer(comparepc)) {
						t.Errorf("expected canAddrDec %#v, got %#v", cad.canAddrDec, fc3)
					}
					if !cmp.Equal(cad.elseDec, fsc, allowunexported, cmp.Comparer(comparepc)) {
						t.Errorf("expected elseDec %#v, got %#v", cad.elseDec, fsc)
					}
				})
			})
		})
	})
	t.Run("Type Map", func(t *testing.T) {
		t.Parallel()
		reg := newTestRegistryBuilder().
			RegisterTypeMapEntry(TypeString, reflect.TypeOf("")).
			RegisterTypeMapEntry(TypeInt32, reflect.TypeOf(int(0))).
			Build()

		var got, want reflect.Type

		want = reflect.TypeOf("")
		got, err := reg.LookupTypeMapEntry(TypeString)
		noerr(t, err)
		if got != want {
			t.Errorf("unexpected type: got %#v, want %#v", got, want)
		}

		want = reflect.TypeOf(int(0))
		got, err = reg.LookupTypeMapEntry(TypeInt32)
		noerr(t, err)
		if got != want {
			t.Errorf("unexpected type: got %#v, want %#v", got, want)
		}

		want = nil
		wanterr := ErrNoTypeMapEntry{Type: TypeObjectID}
		got, err = reg.LookupTypeMapEntry(TypeObjectID)
		if !errors.Is(err, wanterr) {
			t.Errorf("unexpected error: got %#v, want %#v", err, wanterr)
		}
		if got != want {
			t.Errorf("unexpected error: got %#v, want %#v", got, want)
		}
	})
}

func BenchmarkLookupEncoder(b *testing.B) {
	type childStruct struct {
		V1, V2, V3, V4 int
	}
	type nestedStruct struct {
		childStruct
		A struct{ C1, C2, C3, C4 childStruct }
		B struct{ C1, C2, C3, C4 childStruct }
		C struct{ M1, M2, M3, M4 map[int]int }
	}
	types := [...]reflect.Type{
		reflect.TypeOf(int64(1)),
		reflect.TypeOf(&fakeCodec{}),
		reflect.TypeOf(&testInterface1Impl{}),
		reflect.TypeOf(&nestedStruct{}),
	}
	rb := NewRegistryBuilder()
	for _, typ := range types {
		rb.RegisterTypeEncoder(typ, func() ValueEncoder { return &fakeCodec{} })
	}
	r := rb.Build()
	b.Run("Serial", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := r.LookupEncoder(types[i%len(types)])
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("Parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for i := 0; pb.Next(); i++ {
				_, err := r.LookupEncoder(types[i%len(types)])
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})
}

type fakeType1 struct{}
type fakeType2 struct{}
type fakeType3 struct{}
type fakeType4 struct{}
type fakeType5 func(string, string) string
type fakeStructCodec struct{ *fakeCodec }
type fakeSliceCodec struct{ *fakeCodec }
type fakeMapCodec struct{ *fakeCodec }

type fakeCodec struct {
	// num is used to differentiate fakeCodec instances and to force Go to allocate a new value in
	// memory for every fakeCodec. If fakeCodec were an empty struct, Go may use the same pointer
	// for every instance of fakeCodec, making comparisons between pointers to instances of
	// fakeCodec sometimes meaningless.
	num int
}

func (*fakeCodec) EncodeValue(EncoderRegistry, ValueWriter, reflect.Value) error {
	return nil
}
func (*fakeCodec) DecodeValue(DecoderRegistry, ValueReader, reflect.Value) error {
	return nil
}

type testInterface1 interface{ test1() }
type testInterface2 interface{ test2() }
type testInterface3 interface{ test3() }
type testInterface4 interface{ test4() }

type testInterface1Impl struct{}

var _ testInterface1 = testInterface1Impl{}

func (testInterface1Impl) test1() {}

type testInterface2Impl struct{}

var _ testInterface2 = testInterface2Impl{}

func (testInterface2Impl) test2() {}

type testInterface3Impl struct{}

var _ testInterface3 = (*testInterface3Impl)(nil)

func (*testInterface3Impl) test3() {}
