package bsoncodec

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mongodb/mongo-go-driver/bson/bsonrw"
)

func TestRegistry(t *testing.T) {
	t.Run("Register", func(t *testing.T) {
		fc1, fc2, fc3, fc4 := new(fakeCodec), new(fakeCodec), new(fakeCodec), new(fakeCodec)
		t.Run("interface", func(t *testing.T) {
			var t1f *testInterface1
			var t2f *testInterface2
			var t4f *testInterface4
			ips := []interfaceValueEncoder{
				{i: reflect.TypeOf(t1f).Elem(), ve: fc1},
				{i: reflect.TypeOf(t2f).Elem(), ve: fc2},
				{i: reflect.TypeOf(t1f).Elem(), ve: fc3},
				{i: reflect.TypeOf(t4f).Elem(), ve: fc4},
			}
			want := []interfaceValueEncoder{
				{i: reflect.TypeOf(t1f).Elem(), ve: fc3},
				{i: reflect.TypeOf(t2f).Elem(), ve: fc2},
				{i: reflect.TypeOf(t4f).Elem(), ve: fc4},
			}
			rb := NewRegistryBuilder()
			for _, ip := range ips {
				rb.RegisterEncoder(ip.i, ip.ve)
			}
			got := rb.interfaceEncoders
			if !cmp.Equal(got, want, cmp.AllowUnexported(interfaceValueEncoder{}, fakeCodec{}), cmp.Comparer(typeComparer)) {
				t.Errorf("The registered interfaces are not correct. got %v; want %v", got, want)
			}
		})
		t.Run("type", func(t *testing.T) {
			ft1, ft2, ft4 := fakeType1{}, fakeType2{}, fakeType4{}
			rb := NewRegistryBuilder().
				RegisterEncoder(reflect.TypeOf(ft1), fc1).
				RegisterEncoder(reflect.TypeOf(ft2), fc2).
				RegisterEncoder(reflect.TypeOf(ft1), fc3).
				RegisterEncoder(reflect.TypeOf(ft4), fc4)
			want := []struct {
				t reflect.Type
				c ValueEncoder
			}{
				{reflect.PtrTo(reflect.TypeOf(ft1)), fc3},
				{reflect.PtrTo(reflect.TypeOf(ft2)), fc2},
				{reflect.PtrTo(reflect.TypeOf(ft4)), fc4},
			}
			got := rb.typeEncoders
			for _, s := range want {
				wantT, wantC := s.t, s.c
				gotC, exists := got[wantT]
				if !exists {
					t.Errorf("Did not find type in the type registry: %v", wantT)
				}
				if !cmp.Equal(gotC, wantC, cmp.AllowUnexported(fakeCodec{})) {
					t.Errorf("Codecs did not match. got %#v; want %#v", gotC, wantC)
				}
			}
		})
		t.Run("kind", func(t *testing.T) {
			k1, k2, k4 := reflect.Struct, reflect.Slice, reflect.Map
			rb := NewRegistryBuilder().
				RegisterDefaultEncoder(k1, fc1).
				RegisterDefaultEncoder(k2, fc2).
				RegisterDefaultEncoder(k1, fc3).
				RegisterDefaultEncoder(k4, fc4)
			want := []struct {
				k reflect.Kind
				c ValueEncoder
			}{
				{k1, fc3},
				{k2, fc2},
				{k4, fc4},
			}
			got := rb.kindEncoders
			for _, s := range want {
				wantK, wantC := s.k, s.c
				gotC, exists := got[wantK]
				if !exists {
					t.Errorf("Did not find kind in the kind registry: %v", wantK)
				}
				if !cmp.Equal(gotC, wantC, cmp.AllowUnexported(fakeCodec{})) {
					t.Errorf("Codecs did not match. got %#v; want %#v", gotC, wantC)
				}
			}
		})
		t.Run("RegisterDefault", func(t *testing.T) {
			t.Run("MapCodec", func(t *testing.T) {
				codec := fakeCodec{num: 1}
				codec2 := fakeCodec{num: 2}
				rb := NewRegistryBuilder()
				rb.RegisterDefaultEncoder(reflect.Map, codec)
				if rb.kindEncoders[reflect.Map] != codec {
					t.Errorf("Did not properly set the map codec. got %v; want %v", rb.kindEncoders[reflect.Map], codec)
				}
				rb.RegisterDefaultEncoder(reflect.Map, codec2)
				if rb.kindEncoders[reflect.Map] != codec2 {
					t.Errorf("Did not properly set the map codec. got %v; want %v", rb.kindEncoders[reflect.Map], codec2)
				}
			})
			t.Run("StructCodec", func(t *testing.T) {
				codec := fakeCodec{num: 1}
				codec2 := fakeCodec{num: 2}
				rb := NewRegistryBuilder()
				rb.RegisterDefaultEncoder(reflect.Struct, codec)
				if rb.kindEncoders[reflect.Struct] != codec {
					t.Errorf("Did not properly set the struct codec. got %v; want %v", rb.kindEncoders[reflect.Struct], codec)
				}
				rb.RegisterDefaultEncoder(reflect.Struct, codec2)
				if rb.kindEncoders[reflect.Struct] != codec2 {
					t.Errorf("Did not properly set the struct codec. got %v; want %v", rb.kindEncoders[reflect.Struct], codec2)
				}
			})
			t.Run("SliceCodec", func(t *testing.T) {
				codec := fakeCodec{num: 1}
				codec2 := fakeCodec{num: 2}
				rb := NewRegistryBuilder()
				rb.RegisterDefaultEncoder(reflect.Slice, codec)
				if rb.kindEncoders[reflect.Slice] != codec {
					t.Errorf("Did not properly set the slice codec. got %v; want %v", rb.kindEncoders[reflect.Slice], codec)
				}
				rb.RegisterDefaultEncoder(reflect.Slice, codec2)
				if rb.kindEncoders[reflect.Slice] != codec2 {
					t.Errorf("Did not properly set the slice codec. got %v; want %v", rb.kindEncoders[reflect.Slice], codec2)
				}
			})
			t.Run("ArrayCodec", func(t *testing.T) {
				codec := fakeCodec{num: 1}
				codec2 := fakeCodec{num: 2}
				rb := NewRegistryBuilder()
				rb.RegisterDefaultEncoder(reflect.Array, codec)
				if rb.kindEncoders[reflect.Array] != codec {
					t.Errorf("Did not properly set the slice codec. got %v; want %v", rb.kindEncoders[reflect.Array], codec)
				}
				rb.RegisterDefaultEncoder(reflect.Array, codec2)
				if rb.kindEncoders[reflect.Array] != codec2 {
					t.Errorf("Did not properly set the slice codec. got %v; want %v", rb.kindEncoders[reflect.Array], codec2)
				}
			})
		})
		t.Run("Lookup", func(t *testing.T) {
			type Codec interface {
				ValueEncoder
				ValueDecoder
			}

			var arrinstance [12]int
			arr := reflect.TypeOf(arrinstance)
			slc := reflect.TypeOf(make([]int, 12))
			m := reflect.TypeOf(make(map[string]int))
			strct := reflect.TypeOf(struct{ Foo string }{})
			ft1 := reflect.PtrTo(reflect.TypeOf(fakeType1{}))
			ft2 := reflect.TypeOf(fakeType2{})
			ft3 := reflect.TypeOf(fakeType5(func(string, string) string { return "fakeType5" }))
			ti2 := reflect.TypeOf((*testInterface2)(nil)).Elem()
			fc1, fc2, fc4 := fakeCodec{num: 1}, fakeCodec{num: 2}, fakeCodec{num: 4}
			fsc, fslcc, fmc := new(fakeStructCodec), new(fakeSliceCodec), new(fakeMapCodec)
			reg := NewRegistryBuilder().
				RegisterEncoder(ft1, fc1).
				RegisterEncoder(ft2, fc2).
				RegisterEncoder(ti2, fc4).
				RegisterDefaultEncoder(reflect.Struct, fsc).
				RegisterDefaultEncoder(reflect.Slice, fslcc).
				RegisterDefaultEncoder(reflect.Array, fslcc).
				RegisterDefaultEncoder(reflect.Map, fmc).
				RegisterDecoder(ft1, fc1).
				RegisterDecoder(ft2, fc2).
				RegisterDecoder(ti2, fc4).
				RegisterDefaultDecoder(reflect.Struct, fsc).
				RegisterDefaultDecoder(reflect.Slice, fslcc).
				RegisterDefaultDecoder(reflect.Array, fslcc).
				RegisterDefaultDecoder(reflect.Map, fmc).
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
					"interface registry",
					ti2,
					fc4,
					nil,
					true,
				},
				{
					"default struct codec (pointer)",
					reflect.PtrTo(strct),
					fsc,
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
					nil,
					ErrNoEncoder{Type: reflect.TypeOf(map[int]int{})},
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
			for _, tc := range testCases {
				t.Run("Encoder", func(t *testing.T) {
					t.Run(tc.name, func(t *testing.T) {
						gotcodec, goterr := reg.LookupEncoder(tc.t)
						if !cmp.Equal(goterr, tc.wanterr, cmp.Comparer(compareErrors)) {
							t.Errorf("Errors did not match. got %v; want %v", goterr, tc.wanterr)
						}
						if !cmp.Equal(gotcodec, tc.wantcodec, allowunexported) {
							t.Errorf("Codecs did not match. got %v; want %v", gotcodec, tc.wantcodec)
						}
					})
				})
				t.Run("Decoder", func(t *testing.T) {
					t.Run(tc.name, func(t *testing.T) {
						var wanterr error
						if ene, ok := tc.wanterr.(ErrNoEncoder); ok {
							wanterr = ErrNoDecoder{Type: ene.Type}
						} else {
							wanterr = tc.wanterr
						}
						gotcodec, goterr := reg.LookupDecoder(tc.t)
						if !cmp.Equal(goterr, wanterr, cmp.Comparer(compareErrors)) {
							t.Errorf("Errors did not match. got %v; want %v", goterr, wanterr)
						}
						if !cmp.Equal(gotcodec, tc.wantcodec, allowunexported) {
							t.Errorf("Codecs did not match. got %v; want %v", gotcodec, tc.wantcodec)
						}
					})
				})
			}
		})
	})
}

type fakeType1 struct{ b bool }
type fakeType2 struct{ b bool }
type fakeType3 struct{ b bool }
type fakeType4 struct{ b bool }
type fakeType5 func(string, string) string
type fakeStructCodec struct{ fakeCodec }
type fakeSliceCodec struct{ fakeCodec }
type fakeMapCodec struct{ fakeCodec }

type fakeCodec struct{ num int }

func (fc fakeCodec) EncodeValue(EncodeContext, bsonrw.ValueWriter, interface{}) error { return nil }
func (fc fakeCodec) DecodeValue(DecodeContext, bsonrw.ValueReader, interface{}) error { return nil }

type testInterface1 interface{ test1() }
type testInterface2 interface{ test2() }
type testInterface3 interface{ test3() }
type testInterface4 interface{ test4() }

func typeComparer(i1, i2 reflect.Type) bool { return i1 == i2 }
