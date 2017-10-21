package extjson

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/stretchr/testify/require"
)

func TestMarshalD(t *testing.T) {
	type Wrapper struct {
		Val MarshalD `bson:"val"`
	}
	t.Run("Round tripping serializable values should work", func(t *testing.T) {
		ifcPtr := interface{}(true)
		t.Run("JSON", func(t *testing.T) {
			for _, tc := range []struct {
				doc      MarshalD
				expected MarshalD

				str    []byte
				errOut bool
				errIn  bool
			}{
				{
					MarshalD{{"_id", bson.D{{"$oid", "5989e3f2d2b38b22f33411fb"}}}},
					MarshalD{{"_id", bson.ObjectIdHex("5989e3f2d2b38b22f33411fb")}},
					nil, false, false,
				},
				{
					MarshalD{{"_id", bson.D{{"$oid", "xXxBrOkEnOiDxXx"}}}},
					MarshalD{},
					nil, false, true,
				},
				{
					MarshalD{{"value", bson.D{{"$numberDouble", "5"}}}},
					MarshalD{{"value", 5.0}},
					nil, false, false,
				},
				{
					MarshalD{{"$one.two", []interface{}{1.0, 2.0, 3.0}}},
					MarshalD{{"$one.two", []interface{}{1.0, 2.0, 3.0}}},
					nil, false, false,
				},
				{
					MarshalD{{"1", &bson.D{{"one", "two"}}}, {"2", bson.D{{"one", "two"}}}},
					MarshalD{{"1", bson.D{{"one", "two"}}}, {"2", bson.D{{"one", "two"}}}},
					nil, false, false,
				},
				{
					MarshalD{{"1", bson.D{{"one", &ifcPtr}}}, {"2", bson.D{{"one", ifcPtr}}}},
					MarshalD{{"1", bson.D{{"one", ifcPtr}}}, {"2", bson.D{{"one", ifcPtr}}}},
					nil, false, false,
				},
				{
					MarshalD{{"1", func() {}}, {"2", bson.D{{"one", ifcPtr}}}},
					MarshalD{},
					nil, true, false,
				},
				{
					MarshalD{{"1", map[string]interface{}{"a": func() {}}}, {"2", bson.D{{"one", ifcPtr}}}},
					MarshalD{},
					nil, true, false,
				},
				{
					MarshalD{{"1", bson.M{"a": func() {}}}, {"2", bson.D{{"one", ifcPtr}}}},
					MarshalD{},
					nil, true, false,
				},
				{
					MarshalD{},
					MarshalD{},
					nil, false, false,
				},
				{
					MarshalD{},
					MarshalD{},
					[]byte(`{"val": 12}`),
					false,
					true,
				},
				{
					MarshalD{},
					MarshalD{},
					nil,
					false,
					false,
				},
				{
					MarshalD{{"a", []interface{}{func() {}}}},
					MarshalD{},
					nil,
					true,
					false,
				},
			} {
				var md []byte
				var err error

				if tc.str == nil {
					w := Wrapper{tc.doc}
					md, err = json.Marshal(w)
					if tc.errOut {
						require.Error(t, err)
						continue
					}
					require.NoError(t, err)
				} else {
					md = tc.str
				}

				actualDoc := Wrapper{Val: MarshalD{}}
				err = json.Unmarshal(md, &actualDoc)
				if tc.errIn {
					require.Error(t, err)
					continue
				}
				require.NoError(t, err)
				require.Equal(t, actualDoc.Val, tc.expected)
			}
		})

		t.Run("BSON", func(t *testing.T) {
			for _, tc := range []struct {
				doc      MarshalD
				expected MarshalD

				str    []byte
				errOut bool
				errIn  bool
			}{
				{
					MarshalD{{"_id", bson.D{{"$oid", "5989e3f2d2b38b22f33411fb"}}}},
					MarshalD{{"_id", bson.ObjectIdHex("5989e3f2d2b38b22f33411fb")}},
					nil, false, false,
				},
				{
					MarshalD{{"_id", bson.D{{"$oid", "xXxMoArBrOkEnOiDxXx"}}}},
					MarshalD{},
					nil, false, true,
				},
				{
					MarshalD{{"value", bson.D{{"$numberDouble", "5"}}}},
					MarshalD{{"value", 5.0}},
					nil, false, false,
				},
				{
					MarshalD{{"$one.two", []interface{}{1.0, 2.0, 3.0}}},
					MarshalD{{"$one.two", []interface{}{1.0, 2.0, 3.0}}},
					nil, false, false,
				},
				{
					MarshalD{{"1", &bson.D{{"one", "two"}}}, {"2", bson.D{{"one", "two"}}}},
					MarshalD{{"1", bson.D{{"one", "two"}}}, {"2", bson.D{{"one", "two"}}}},
					nil, false, false,
				},
				{
					MarshalD{{"1", bson.D{{"one", &ifcPtr}}}, {"2", bson.D{{"one", ifcPtr}}}},
					MarshalD{{"1", bson.D{{"one", ifcPtr}}}, {"2", bson.D{{"one", ifcPtr}}}},
					nil, false, false,
				},
				{
					MarshalD{{"1", func() {}}, {"2", bson.D{{"one", ifcPtr}}}},
					MarshalD{},
					nil, true, false,
				},
				{
					MarshalD{},
					MarshalD{},
					nil, false, false,
				},
				{
					MarshalD{},
					MarshalD{},
					[]byte(`12`),
					false,
					true,
				},
				{
					MarshalD{},
					MarshalD(nil),
					[]byte(``),
					false,
					false,
				},
			} {

				var md []byte
				var err error

				if tc.str == nil {
					w := Wrapper{tc.doc}
					md, err = bson.Marshal(w)
					if tc.errOut {
						require.Error(t, err)
						continue
					}
					require.NoError(t, err)
				} else {
					md, err = bson.Marshal(struct {
						Val string `bson:"val"`
					}{string(tc.str)})
					require.NoError(t, err)
				}

				var actualDoc Wrapper
				err = bson.Unmarshal(md, &actualDoc)
				if tc.errIn {
					require.Error(t, err)
					continue
				}
				require.NoError(t, err)
				require.Equal(t, actualDoc.Val, tc.expected)
			}
		})
	})
}

func TestValue(t *testing.T) {
	type Wrapper struct {
		Val *Value `bson:"val"`
	}
	t.Run("Round tripping serializable values should work", func(t *testing.T) {
		assertBSON := func(input Wrapper) Wrapper {
			out, err := bson.Marshal(input)
			require.NoError(t, err)

			wrapper := Wrapper{}
			require.NoError(t, bson.Unmarshal(out, &wrapper))
			require.IsType(t, wrapper.Val.V, bson.D{})
			return wrapper
		}

		assertJSON := func(input Wrapper) Wrapper {
			out, err := json.Marshal(input)
			require.NoError(t, err)

			wrapper := Wrapper{}
			require.NoError(t, json.Unmarshal(out, &wrapper))
			require.IsType(t, wrapper.Val.V, bson.D{})
			return wrapper
		}

		t.Run("bson.D", func(t *testing.T) {
			expected := NewValueOf(bson.D{{"$one.two", []interface{}{1.0, 2.0, 3.0}}})

			// BSON
			wrapper := assertBSON(Wrapper{expected})
			require.Equal(t, wrapper.Val.V, expected.V)

			// JSON
			wrapper = assertJSON(Wrapper{expected})
			require.Equal(t, wrapper.Val.V, expected.V)
		})

		t.Run("bson.D with extended JSON", func(t *testing.T) {
			hex := "5989e3f2d2b38b22f33411fb"
			input := NewValueOf(bson.D{{"_id", bson.D{{"$oid", hex}}}})
			expected := bson.D{{"_id", bson.ObjectIdHex(hex)}}

			// BSON
			wrapper := assertBSON(Wrapper{input})
			require.Equal(t, wrapper.Val.V, expected)

			// JSON
			wrapper = assertJSON(Wrapper{input})
			require.Equal(t, wrapper.Val.V, expected)
		})

		t.Run("Primitives", func(t *testing.T) {
			for _, tc := range []struct {
				in     interface{}
				out    interface{}
				errOut bool
			}{
				{nil, nil, false},
				{1.0, 1.0, false},
				{false, false, false},
				{true, true, false},
				{"true", "true", false},
				{"", "", false},
				{[]interface{}{1.0, false, "true"}, []interface{}{1.0, false, "true"}, false},
				{[]interface{}{1.0, false, "true", bson.D{{"one", 1.0}}}, []interface{}{1.0, false, "true", bson.D{{"one", 1.0}}}, false},
			} {
				for i := 0; i < 2; i++ {
					safeVal := NewValueOf(tc.in)
					wrapper := Wrapper{safeVal}

					if i == 0 {
						out, err := bson.Marshal(wrapper)

						if tc.errOut {
							require.Error(t, err)
							continue
						}
						require.NoError(t, err)

						wrapper = Wrapper{}
						require.NoError(t, bson.Unmarshal(out, &wrapper))
						require.Equal(t, wrapper.Val.V, tc.out)
					}

					if i == 1 {
						out, err := json.Marshal(wrapper)

						if tc.errOut {
							require.Error(t, err)
							continue
						}
						require.NoError(t, err)

						wrapper = Wrapper{}
						require.NoError(t, json.Unmarshal(out, &wrapper))
						if tc.in == nil {
							require.Nil(t, wrapper.Val)
						} else {
							require.Equal(t, wrapper.Val.V, tc.out)
						}
					}
				}
			}
		})

		t.Run("Empty JSON should return a nil value", func(t *testing.T) {
			// BSON
			out, err := bson.Marshal(bson.D{{"str", ""}})
			require.NoError(t, err)

			wrapper := struct {
				Str bson.Raw `bson:"str"`
			}{}
			require.NoError(t, bson.Unmarshal(out, &wrapper))

			safeVal := &Value{}
			require.NoError(t, safeVal.SetBSON(wrapper.Str))
			require.Nil(t, safeVal.V)

			// JSON
			safeVal = &Value{}
			require.NoError(t, safeVal.UnmarshalJSON([]byte("")))
			require.Nil(t, safeVal.V)
		})
	})

	t.Run("Errors during unmarshaling should be handled", func(t *testing.T) {
		t.Run("BSON", func(t *testing.T) {
			t.Run("Bad JSON", func(t *testing.T) {
				out, err := bson.Marshal(bson.D{{"val", 12}})
				require.NoError(t, err)

				var wrapper Wrapper
				require.Error(t, bson.Unmarshal(out, &wrapper))
			})

			t.Run("Bad JSON", func(t *testing.T) {
				safeVal := &Value{}
				require.Error(t, safeVal.SetBSON(bson.Raw{Kind: 0x02}))

			})

			t.Run("Bad JSON", func(t *testing.T) {
				out, err := bson.Marshal(bson.D{{"val", `{`}})
				require.NoError(t, err)

				var wrapper Wrapper
				require.Error(t, bson.Unmarshal(out, &wrapper))
			})
		})

		t.Run("JSON", func(t *testing.T) {
			t.Run("Bad JSON", func(t *testing.T) {
				safeVal := &Value{}
				require.Error(t, safeVal.UnmarshalJSON([]byte("{")))
			})
		})
	})
}

func TestValueWithOptions(t *testing.T) {
	serialize := func(v Value) string {
		serialized, err := json.Marshal(v)
		require.NoError(t, err)

		return string(serialized)
	}

	bsonInput := bson.D{
		{"number", 5},
		{"mapNumber", bson.D{{"value", 9}}},
		{"other", bson.D{
			{"number", 6},
			{"_id", bson.ObjectIdHex("5989e3f2d2b38b22f33411fb")},
		},
		},
	}
	t.Run("When Extended JSON is disabled", func(t *testing.T) {
		isDisabled := true
		expected := `{"number":5,"mapNumber":{"value":9},"other":{"number":6,"_id":"5989e3f2d2b38b22f33411fb"}}`
		t.Run("bson.D values should not be serialized as Extended JSON", func(t *testing.T) {
			input := ValueOf(bsonInput).NewWithOptions(ValueOptions{DisableExtended: isDisabled})
			require.Equal(t, serialize(input), expected)
		})

		t.Run("bson.D pointer values should not be serialized as Extended JSON", func(t *testing.T) {
			input := ValueOf(&bsonInput).NewWithOptions(ValueOptions{DisableExtended: isDisabled})
			require.Equal(t, serialize(input), expected)
		})
	})

	t.Run("When Extended JSON is enabled", func(t *testing.T) {
		isDisabled := false
		expected := `{"number":{"$numberInt":"5"},"mapNumber":{"value":{"$numberInt":"9"}},"other":{"number":{"$numberInt":"6"},"_id":{"$oid":"5989e3f2d2b38b22f33411fb"}}}`
		t.Run("bson.D values should be serialized as Extended JSON", func(t *testing.T) {
			input := ValueOf(bsonInput).NewWithOptions(ValueOptions{DisableExtended: isDisabled})
			require.Equal(t, serialize(input), expected)
		})

		t.Run("bson.D pointer values should be serialized as Extended JSON", func(t *testing.T) {
			v := bsonInput
			input := ValueOf(&v).NewWithOptions(ValueOptions{DisableExtended: isDisabled})
			require.Equal(t, serialize(input), expected)
		})
	})
}

func TestJSONDecodeToBSON(t *testing.T) {
	t.Run("decoding json into bson.D", func(t *testing.T) {
		for _, c := range []struct {
			in             string
			expectedDecode interface{}
			expectedRT     interface{}
			expectedErr    bool
			expectedOk     bool
		}{
			{
				`{"blah":[1,2,3], "foo": [4, 5, 6], "string":"test","Null": null, "Number": 1.234, "truebool": true}`,
				bson.D{
					{"blah", []interface{}{1.0, 2.0, 3.0}},
					{"foo", []interface{}{4.0, 5.0, 6.0}},
					{"string", "test"},
					{"Null", nil},
					{"Number", 1.234},
					{"truebool", true},
				},
				bson.D{
					{"blah", []interface{}{bson.D{{"$numberDouble", "1.0"}}, bson.D{{"$numberDouble", "2.0"}}, bson.D{{"$numberDouble", "3.0"}}}},
					{"foo", []interface{}{bson.D{{"$numberDouble", "4.0"}}, bson.D{{"$numberDouble", "5.0"}}, bson.D{{"$numberDouble", "6.0"}}}},
					{"string", "test"},
					{"Null", nil},
					{"Number", bson.D{{"$numberDouble", "1.234"}}},
					{"truebool", true},
				},
				false,
				true,
			},
			{
				`{"blah":[[[1,2,3],4,5,[6,{"x":7}]]]}`,
				bson.D{{"blah",
					[]interface{}{
						[]interface{}{
							[]interface{}{
								1.0,
								2.0,
								3.0,
							},
							4.0, 5.0,
							[]interface{}{
								6.0,
								bson.D{{"x", 7.0}},
							},
						},
					},
				}},
				bson.D{{"blah",
					[]interface{}{
						[]interface{}{
							[]interface{}{
								bson.D{{"$numberDouble", "1.0"}},
								bson.D{{"$numberDouble", "2.0"}},
								bson.D{{"$numberDouble", "3.0"}},
							},
							bson.D{{"$numberDouble", "4.0"}}, bson.D{{"$numberDouble", "5.0"}},
							[]interface{}{
								bson.D{{"$numberDouble", "6.0"}},
								bson.D{{"x", bson.D{{"$numberDouble", "7.0"}}}},
							},
						},
					},
				}},
				false,
				true,
			},
			{
				`{"x":{"y":[{"z":1}]}}`,
				bson.D{{"x", bson.D{{"y", []interface{}{bson.D{{"z", 1.0}}}}}}},
				bson.D{{"x", bson.D{{"y", []interface{}{bson.D{{"z", bson.D{{"$numberDouble", "1.0"}}}}}}}}},
				false,
				true,
			},
			{
				"null",
				nil,
				nil,
				false,
				true,
			},
			{
				"3",
				3.0,
				map[string]interface{}{"$numberDouble": "3"},
				false,
				true,
			},
			{
				"",
				nil,
				nil,
				false,
				false,
			},
			{
				`["foo", "bar"]`,
				[]interface{}{"foo", "bar"},
				[]interface{}{"foo", "bar"},
				false,
				true,
			},
			{
				`["foo", "bar", {"one": 2}]`,
				[]interface{}{"foo", "bar", bson.D{{"one", 2.0}}},
				[]interface{}{"foo", "bar", bson.D{{"one", map[string]interface{}{"$numberDouble": "2"}}}},
				false,
				true,
			},
			//			/* tests to verify that broken json produces errors */
			{`["foo", `, nil, nil, true, false},
			{`{"foo":} `, nil, nil, true, false},
		} {
			t.Run(fmt.Sprintf("checking parse of %v", c.in), func(t *testing.T) {
				out, ok, err := decodeJSONasBSON(strings.NewReader(c.in))
				require.Equal(t, ok, c.expectedOk)

				if c.expectedErr {
					require.Error(t, err)
					require.Nil(t, out)
				} else {
					require.NoError(t, err)
					require.Equal(t, out, c.expectedDecode)
				}

				if !c.expectedErr && c.expectedOk {
					if outBson, ok := out.(bson.D); ok {
						t.Run("checking roundtrip correctness", func(t *testing.T) {
							// Check that decoding/encoding can roundtrip properly.
							encoded, err := EncodeBSONDtoJSON(outBson)
							require.NoError(t, err)

							redecoded, ok, err := decodeJSONasBSON(bytes.NewReader(encoded))
							require.NoError(t, err)
							require.True(t, ok)
							require.Equal(t, redecoded, c.expectedRT)
						})
					}
				}
			})
		}
	})
}

type rawWrapper struct {
	Raw rawInner
}

type rawInner struct {
	inner bson.Raw
}

func (ri *rawInner) SetBSON(raw bson.Raw) error {
	ri.inner = raw
	return nil
}

func TestSetBSON(t *testing.T) {
	t.Run("Unmarshaling should only work from a string", func(t *testing.T) {
		t.Run("Malformed BSON should fail", func(t *testing.T) {
			md := MarshalD{}
			require.Error(t, md.SetBSON(bson.Raw{Kind: 0x02}))
		})

		t.Run("Non-string should fail", func(t *testing.T) {
			out, err := bson.Marshal(bson.M{"not": "a string"})
			require.NoError(t, err)

			md := MarshalD{}
			require.Error(t, bson.Unmarshal(out, &md))
		})

		t.Run("String but invalid JSON should fail", func(t *testing.T) {
			out, err := bson.Marshal(bson.M{"raw": "abc"})
			require.NoError(t, err)

			rW := rawWrapper{}
			require.NoError(t, bson.Unmarshal(out, &rW))

			md := MarshalD{}
			require.Error(t, md.SetBSON(rW.Raw.inner))
		})

		t.Run("String and valid JSON", func(t *testing.T) {
			out, err := bson.Marshal(bson.M{"raw": "{\"some\": \"val\"}"})
			require.NoError(t, err)

			rW := rawWrapper{}
			require.NoError(t, bson.Unmarshal(out, &rW))

			md := MarshalD{}
			require.NoError(t, md.SetBSON(rW.Raw.inner))
		})
	})
}

func TestUnmarshaler(t *testing.T) {
	t.Run("When unmarshaling data using the json.Unmarshaler interface", func(t *testing.T) {
		t.Run("an object should be able to be decoded into a field of type MarshalD", func(t *testing.T) {
			in1 := struct {
				X MarshalD `json:"x"`
				Y string   `json:"y"`
			}{}

			err := json.Unmarshal(
				[]byte(`{"x":{"foo":"bar", "xxx":"yyy"}, "y":"hi"}`),
				&in1)
			require.NoError(t, err)
			require.Equal(t, in1.X, MarshalD{{"foo", "bar"}, {"xxx", "yyy"}})
		})
		t.Run("a pointer to MarshalD should be left nil when omitempty used", func(t *testing.T) {
			in1 := struct {
				X *MarshalD `json:"x,omitempty"`
				Y string    `json:"y"`
			}{}

			err := json.Unmarshal(
				[]byte(`{"y":"hi"}`),
				&in1)
			require.NoError(t, err)
			require.Nil(t, in1.X)
		})
		t.Run("an error should be thrown if incoming value is not an object", func(t *testing.T) {
			in1 := struct {
				X MarshalD `json:"x,omitempty"`
				Y string   `json:"y"`
			}{}

			err := json.Unmarshal(
				[]byte(`{"x": "not_an_object", "y":"hi"}`),
				&in1)
			require.Error(t, err)
		})
	})
}

func getMarshalDEncodeBenchDoc(b *testing.B) MarshalD {
	uUID, err := base64.StdEncoding.DecodeString("o0w498Or7cijeBSpkquNtg==")
	if err != nil {
		b.Error(err)
	}
	dec128, err := bson.ParseDecimal128("-1.869E5")
	if err != nil {
		b.Error(err)
	}
	return MarshalD{
		{"_id", bson.ObjectIdHex("57e193d7a9cc81b4027498b5")},
		{"Symbol", bson.Symbol("symbol")},
		{"String", "string"},
		{"Int32", 42},
		{"Int64", int64(42)},
		{"Double", 42.42},
		{"DoubleNaN", math.NaN()},
		{"DoubleInf", math.Inf(1)},
		{"DoubleNegInf", math.Inf(-1)},
		{"Binary", bson.Binary{Kind: 0x03, Data: uUID}},
		{"BinaryUserDefined", bson.Binary{Kind: 0x80, Data: []byte{1, 2, 3, 4, 5}}},
		{"Code", bson.JavaScript{Code: "function() {}"}},
		{"CodeWithScope", bson.JavaScript{Code: "function() {}", Scope: bson.M{}}},
		{"Subdocument", bson.M{"foo": "bar"}},
		{"Array", []interface{}{1, 2, 3, 4, 5}},
		{"Timestamp", bson.MongoTimestamp((42 << 32) | 1)},
		{"Regex", bson.RegEx{Pattern: "pattern"}},
		{"DatetimeEpoch", time.Unix(0, 0)},
		{"DatetimePositive", time.Unix(int64(math.MaxInt64)/1e3, int64(math.MaxInt64)%1e3*1e6)},
		{"DatetimeNegative", time.Unix(int64(math.MinInt64)/1e3, int64(math.MinInt64)%1e3*1e6)},
		{"True", true},
		{"False", false},
		{"DBPointer", bson.DBPointer{
			Namespace: "db.collection",
			Id:        bson.ObjectIdHex("57e193d7a9cc81b4027498b1"),
		}},
		{"DBRef", DBRef{
			Collection: "collection",
			ID:         bson.ObjectIdHex("57fd71e96e32ab4225b723fb"),
			Database:   "database",
		}},
		{"DBRef2", DBRef{
			Collection: "collection",
			ID:         "test",
			Database:   "database",
		}},
		{"Minkey", bson.MinKey},
		{"Maxkey", bson.MaxKey},
		{"Null", nil},
		{"NullInDoc", bson.M{
			"n": nil,
		}},
		{"Undefined", bson.Undefined},
		{"Decimal", dec128},
		{"DecimalNaN", Decimal128NaN},
		{"DecimalInf", Decimal128Inf},
		{"DecimalNegInf", Decimal128NegInf},
	}
}

func BenchmarkMarshalDEncode(b *testing.B) {
	doc := getMarshalDEncodeBenchDoc(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := doc.MarshalJSON(); err != nil {
			b.Error(err)
		}
	}
}

func TestValueEncryption(t *testing.T) {
	crypter := NewTestCrypter()
	t.Run("Round tripping values should work", func(t *testing.T) {
		for _, tc := range []Value{
			{V: 1.0},
			{V: "secret"},
			{V: nil},
			{V: 1.0},
			{V: "secret"},
			{
				V: bson.D{{"1", false}}},
		} {
			encrypted, err := tc.Encrypt(crypter)
			require.NoError(t, err)
			require.NotEqual(t, encrypted, tc)

			decrypted, err := crypter.Decrypt(encrypted)
			require.NoError(t, err)
			require.Equal(t, decrypted.(Value), tc)
		}
	})
}
