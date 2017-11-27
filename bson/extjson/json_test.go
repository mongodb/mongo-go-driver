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
	gc "github.com/smartystreets/goconvey/convey"
)

func TestMarshalD(t *testing.T) {
	type Wrapper struct {
		Val MarshalD `bson:"val"`
	}
	gc.Convey("Round tripping serializable values should work", t, func() {

		ifcPtr := interface{}(true)

		gc.Convey("JSON", func() {
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
						gc.So(err, gc.ShouldNotBeNil)
						continue
					}
					gc.So(err, gc.ShouldBeNil)
				} else {
					md = tc.str
				}

				actualDoc := Wrapper{Val: MarshalD{}}
				err = json.Unmarshal(md, &actualDoc)
				if tc.errIn {
					gc.So(err, gc.ShouldNotBeNil)
					continue
				}
				gc.So(err, gc.ShouldBeNil)

				gc.So(actualDoc.Val, gc.ShouldResemble, tc.expected)
			}
		})

		gc.Convey("BSON", func() {
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
						gc.So(err, gc.ShouldNotBeNil)
						continue
					}
					gc.So(err, gc.ShouldBeNil)
				} else {
					md, err = bson.Marshal(struct {
						Val string `bson:"val"`
					}{string(tc.str)})
					gc.So(err, gc.ShouldBeNil)
				}

				var actualDoc Wrapper
				err = bson.Unmarshal(md, &actualDoc)
				if tc.errIn {
					gc.So(err, gc.ShouldNotBeNil)
					continue
				}
				gc.So(err, gc.ShouldBeNil)

				gc.So(actualDoc.Val, gc.ShouldResemble, tc.expected)
			}
		})
	})
}

func TestValue(t *testing.T) {
	type Wrapper struct {
		Val *Value `bson:"val"`
	}
	gc.Convey("Round tripping serializable values should work", t, func() {
		assertBSON := func(input Wrapper) Wrapper {
			out, err := bson.Marshal(input)
			gc.So(err, gc.ShouldBeNil)

			wrapper := Wrapper{}
			gc.So(bson.Unmarshal(out, &wrapper), gc.ShouldBeNil)
			gc.So(wrapper.Val.V, gc.ShouldHaveSameTypeAs, bson.D{})

			return wrapper
		}

		assertJSON := func(input Wrapper) Wrapper {
			out, err := json.Marshal(input)
			gc.So(err, gc.ShouldBeNil)

			wrapper := Wrapper{}
			gc.So(json.Unmarshal(out, &wrapper), gc.ShouldBeNil)
			gc.So(wrapper.Val.V, gc.ShouldHaveSameTypeAs, bson.D{})

			return wrapper
		}

		gc.Convey("bson.D", func() {
			expected := NewValueOf(bson.D{{"$one.two", []interface{}{1.0, 2.0, 3.0}}})

			// BSON
			wrapper := assertBSON(Wrapper{expected})
			gc.So(wrapper.Val.V, gc.ShouldResemble, expected.V)

			// JSON
			wrapper = assertJSON(Wrapper{expected})
			gc.So(wrapper.Val.V, gc.ShouldResemble, expected.V)
		})

		gc.Convey("bson.D with extended JSON", func() {
			hex := "5989e3f2d2b38b22f33411fb"
			input := NewValueOf(bson.D{{"_id", bson.D{{"$oid", hex}}}})
			expected := bson.D{{"_id", bson.ObjectIdHex(hex)}}

			// BSON
			wrapper := assertBSON(Wrapper{input})
			gc.So(wrapper.Val.V, gc.ShouldResemble, expected)

			// JSON
			wrapper = assertJSON(Wrapper{input})
			gc.So(wrapper.Val.V, gc.ShouldResemble, expected)
		})

		gc.Convey("Primitives", func() {
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
							gc.So(err, gc.ShouldNotBeNil)
							continue
						}
						gc.So(err, gc.ShouldBeNil)

						wrapper = Wrapper{}
						gc.So(bson.Unmarshal(out, &wrapper), gc.ShouldBeNil)
						gc.So(wrapper.Val.V, gc.ShouldResemble, tc.out)
					}

					if i == 1 {
						out, err := json.Marshal(wrapper)

						if tc.errOut {
							gc.So(err, gc.ShouldNotBeNil)
							continue
						}
						gc.So(err, gc.ShouldBeNil)

						wrapper = Wrapper{}
						gc.So(json.Unmarshal(out, &wrapper), gc.ShouldBeNil)

						if tc.in == nil {
							gc.So(wrapper.Val, gc.ShouldBeNil)
						} else {
							gc.So(wrapper.Val.V, gc.ShouldResemble, tc.out)
						}
					}
				}
			}
		})

		gc.Convey("Empty JSON should return a nil value", func() {
			// BSON
			out, err := bson.Marshal(bson.D{{"str", ""}})
			gc.So(err, gc.ShouldBeNil)

			wrapper := struct {
				Str bson.Raw `bson:"str"`
			}{}
			gc.So(bson.Unmarshal(out, &wrapper), gc.ShouldBeNil)

			safeVal := &Value{}
			gc.So(safeVal.SetBSON(wrapper.Str), gc.ShouldBeNil)
			gc.So(safeVal.V, gc.ShouldBeNil)

			// JSON
			safeVal = &Value{}
			gc.So(safeVal.UnmarshalJSON([]byte("")), gc.ShouldBeNil)
			gc.So(safeVal.V, gc.ShouldBeNil)
		})
	})

	gc.Convey("Errors during unmarshaling should be handled", t, func() {
		gc.Convey("BSON", func() {
			gc.Convey("Bad type", func() {
				out, err := bson.Marshal(bson.D{{"val", 12}})
				gc.So(err, gc.ShouldBeNil)

				var wrapper Wrapper
				gc.So(bson.Unmarshal(out, &wrapper), gc.ShouldNotBeNil)
			})

			gc.Convey("Bad BSON", func() {
				safeVal := &Value{}
				gc.So(safeVal.SetBSON(bson.Raw{Kind: 0x02}), gc.ShouldNotBeNil)
			})

			gc.Convey("Bad JSON", func() {
				out, err := bson.Marshal(bson.D{{"val", `{`}})
				gc.So(err, gc.ShouldBeNil)

				var wrapper Wrapper
				gc.So(bson.Unmarshal(out, &wrapper), gc.ShouldNotBeNil)
			})
		})

		gc.Convey("JSON", func() {
			gc.Convey("Bad JSON", func() {
				safeVal := &Value{}
				gc.So(safeVal.UnmarshalJSON([]byte("{")), gc.ShouldNotBeNil)
			})
		})
	})
}

func TestValueWithOptions(t *testing.T) {
	serialize := func(v Value) string {
		serialized, err := json.Marshal(v)
		gc.So(err, gc.ShouldBeNil)

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

	gc.Convey("When Extended JSON is disabled", t, func() {
		isDisabled := true
		expected := `{"number":5,"mapNumber":{"value":9},"other":{"number":6,"_id":"5989e3f2d2b38b22f33411fb"}}`

		gc.Convey("bson.D values should not be serialized as Extended JSON", func() {
			input := ValueOf(bsonInput).NewWithOptions(ValueOptions{DisableExtended: isDisabled})

			gc.So(serialize(input), gc.ShouldEqual, expected)
		})

		gc.Convey("bson.D pointer values should not be serialized as Extended JSON", func() {
			input := ValueOf(&bsonInput).NewWithOptions(ValueOptions{DisableExtended: isDisabled})

			gc.So(serialize(input), gc.ShouldEqual, expected)
		})
	})

	gc.Convey("When Extended JSON is enabled", t, func() {
		isDisabled := false
		expected := `{"number":{"$numberInt":"5"},"mapNumber":{"value":{"$numberInt":"9"}},"other":{"number":{"$numberInt":"6"},"_id":{"$oid":"5989e3f2d2b38b22f33411fb"}}}`

		gc.Convey("bson.D values should be serialized as Extended JSON", func() {
			input := ValueOf(bsonInput).NewWithOptions(ValueOptions{DisableExtended: isDisabled})

			gc.So(serialize(input), gc.ShouldEqual, expected)
		})

		gc.Convey("bson.D pointer values should be serialized as Extended JSON", func() {
			v := bsonInput
			input := ValueOf(&v).NewWithOptions(ValueOptions{DisableExtended: isDisabled})

			gc.So(serialize(input), gc.ShouldEqual, expected)
		})
	})
}

func TestJSONDecodeToBSON(t *testing.T) {
	gc.Convey("decoding json into bson.D", t, func() {
		for _, c := range []struct {
			in             string
			expectedDecode interface{}
			expectedRT     interface{}
			expectedErr    bool
			expectedOk     bool
		}{
			{
				`{"blah":[1,2,3], "foo": [4, 5, 6], "string":"poop","Null": null, "Number": 1.234, "truebool": true}`,
				bson.D{
					{"blah", []interface{}{1.0, 2.0, 3.0}},
					{"foo", []interface{}{4.0, 5.0, 6.0}},
					{"string", "poop"},
					{"Null", nil},
					{"Number", 1.234},
					{"truebool", true},
				},
				bson.D{
					{"blah", []interface{}{bson.D{{"$numberDouble", "1"}}, bson.D{{"$numberDouble", "2"}}, bson.D{{"$numberDouble", "3"}}}},
					{"foo", []interface{}{bson.D{{"$numberDouble", "4"}}, bson.D{{"$numberDouble", "5"}}, bson.D{{"$numberDouble", "6"}}}},
					{"string", "poop"},
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
								bson.D{{"$numberDouble", "1"}},
								bson.D{{"$numberDouble", "2"}},
								bson.D{{"$numberDouble", "3"}},
							},
							bson.D{{"$numberDouble", "4"}}, bson.D{{"$numberDouble", "5"}},
							[]interface{}{
								bson.D{{"$numberDouble", "6"}},
								bson.D{{"x", bson.D{{"$numberDouble", "7"}}}},
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
				bson.D{{"x", bson.D{{"y", []interface{}{bson.D{{"z", bson.D{{"$numberDouble", "1"}}}}}}}}},
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
			gc.Convey(fmt.Sprintf("checking parse of %v", c.in), func() {
				out, ok, err := decodeJSONasBSON(strings.NewReader(c.in))
				gc.So(ok, gc.ShouldEqual, c.expectedOk)
				if c.expectedErr {
					gc.So(err, gc.ShouldNotBeNil)
					gc.So(out, gc.ShouldBeNil)
				} else {
					gc.So(err, gc.ShouldBeNil)
					gc.So(out, gc.ShouldResemble, c.expectedDecode)
				}

				if !c.expectedErr && c.expectedOk {
					if outBson, ok := out.(bson.D); ok {
						gc.Convey("checking roundtrip correctness", func() {
							// Check that decoding/encoding can roundtrip properly.
							encoded, err := EncodeBSONDtoJSON(outBson)
							gc.So(err, gc.ShouldBeNil)

							redecoded, ok, err := decodeJSONasBSON(bytes.NewReader(encoded))
							gc.So(err, gc.ShouldBeNil)
							gc.So(ok, gc.ShouldBeTrue)
							gc.So(redecoded, gc.ShouldResemble, c.expectedRT)
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
	gc.Convey("Unmarshaling should only work from a string", t, func() {
		gc.Convey("Malformed BSON should fail", func() {
			md := MarshalD{}
			gc.So(md.SetBSON(bson.Raw{Kind: 0x02}), gc.ShouldNotBeNil)
		})

		gc.Convey("Non-string should fail", func() {
			out, err := bson.Marshal(bson.M{"not": "a string"})
			gc.So(err, gc.ShouldBeNil)

			md := MarshalD{}
			gc.So(bson.Unmarshal(out, &md), gc.ShouldNotBeNil)
		})

		gc.Convey("String but invalid JSON should fail", func() {
			out, err := bson.Marshal(bson.M{"raw": "abc"})
			gc.So(err, gc.ShouldBeNil)

			rW := rawWrapper{}
			gc.So(bson.Unmarshal(out, &rW), gc.ShouldBeNil)

			md := MarshalD{}
			gc.So(md.SetBSON(rW.Raw.inner), gc.ShouldNotBeNil)
		})

		gc.Convey("String and valid JSON", func() {
			out, err := bson.Marshal(bson.M{"raw": "{\"some\": \"val\"}"})
			gc.So(err, gc.ShouldBeNil)

			rW := rawWrapper{}
			gc.So(bson.Unmarshal(out, &rW), gc.ShouldBeNil)

			md := MarshalD{}
			gc.So(md.SetBSON(rW.Raw.inner), gc.ShouldBeNil)
		})
	})
}

func TestUnmarshaler(t *testing.T) {
	gc.Convey("When unmarshaling data using the json.Unmarshaler interface", t, func() {
		gc.Convey("an object should be able to be decoded into a field of type MarshalD", func() {
			in1 := struct {
				X MarshalD `json:"x"`
				Y string   `json:"y"`
			}{}

			err := json.Unmarshal(
				[]byte(`{"x":{"foo":"bar", "xxx":"yyy"}, "y":"hi"}`),
				&in1)
			gc.So(err, gc.ShouldBeNil)
			gc.So(in1.X, gc.ShouldResemble, MarshalD{{"foo", "bar"}, {"xxx", "yyy"}})
		})
		gc.Convey("a pointer to MarshalD should be left nil when omitempty used", func() {
			in1 := struct {
				X *MarshalD `json:"x,omitempty"`
				Y string    `json:"y"`
			}{}

			err := json.Unmarshal(
				[]byte(`{"y":"hi"}`),
				&in1)
			gc.So(err, gc.ShouldBeNil)
			gc.So(in1.X, gc.ShouldBeNil)
		})
		gc.Convey("an error should be thrown if incoming value is not an object", func() {
			in1 := struct {
				X MarshalD `json:"x,omitempty"`
				Y string   `json:"y"`
			}{}

			err := json.Unmarshal(
				[]byte(`{"x": "not_an_object", "y":"hi"}`),
				&in1)
			gc.So(err, gc.ShouldNotBeNil)
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
			Id:         bson.ObjectIdHex("57fd71e96e32ab4225b723fb"),
			Database:   "database",
		}},
		{"DBRef2", DBRef{
			Collection: "collection",
			Id:         "test",
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
	gc.Convey("Round tripping values should work", t, func() {
		for _, tc := range []Value{
			{V: 1.0},
			{V: "secret"},
			{V: nil},
			{V: 1.0},
			{V: "secret"},
			{
				V: bson.D{{"1", false}}},
		} {
			// Incorrect method being called
			//encrypted, err := crypter.Encrypt(tc)
			encrypted, err := tc.Encrypt(crypter)
			gc.So(err, gc.ShouldBeNil)
			gc.So(encrypted, gc.ShouldNotResemble, tc)

			decrypted, err := crypter.Decrypt(encrypted)
			gc.So(err, gc.ShouldBeNil)
			gc.So(decrypted.(Value), gc.ShouldResemble, tc)
		}
	})
}
