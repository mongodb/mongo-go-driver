package extjson_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/10gen/stitch/utils/test"

	gc "github.com/smartystreets/goconvey/convey"
	xjson "github.com/10gen/mongo-go-driver/bson/extjson"
	"github.com/10gen/mongo-go-driver/bson"
)

const docStr = `{
	"_id" : {
	  "quarter" : {
		  "year" : {"$numberInt": "2016"},
	    "quarter" : {"$numberInt": "2"}
    },
	  "user" : "name"
	},
	"email" : "name@company.com",
	"firstName" : "name",
	"lastName" : "namename",
	"title" : "CTO",
	"date":{"$date":{"$numberLong":"1257894000000"}},
	"ts":{"$timestamp": "1475683896" },
	"reg":{"$regex":"^abc", "$options":"g"},
	"binary1": {"$binary":"aGVsbG8gd29ybGQ=", "$type":"00"},
	"objectives" : [
	  {"$oid": "57bb4dfcbae72afa928ae1bb"},
	  {"$oid": "57c03963f11724f0d73e0a7a"}
	],
	"int1" : {"$numberInt": "100"},
	"l1" : {"$numberLong": "100"},
	"code" : {"$code": "function(){}"},
	"cws" : {"$code": "function(){}", "$scope":{"a": {"$numberInt": "1"}}},
	"undef" : {"$undefined":true},
	"min" : {"$minKey":1},
	"max" : {"$maxKey":1},
	"status" : "DRAFTING",
	"preferredName" : "Naming",
	"managers" : [
	  "name.namey"
	],
	"reportsTo" : "name.namey"
}`

func assertBsonD(t *testing.T, jsonString string) (bson.D, bson.D) {
	doc := xjson.MarshalD{} // Need this to get the data
	gc.So(json.Unmarshal([]byte(jsonString), &doc), gc.ShouldBeNil)
	bsonDoc := bson.D(doc)

	dec, err := xjson.DecodeExtended(bsonDoc)
	gc.So(err, gc.ShouldBeNil)

	r := xjson.MarshalD(dec.(bson.D))
	rDocStr, err := r.MarshalJSON()
	gc.So(err, gc.ShouldBeNil)

	rDoc := xjson.MarshalD{}
	gc.So(json.Unmarshal([]byte(rDocStr), &rDoc), gc.ShouldBeNil)
	return bson.D(rDoc), bsonDoc
}


func assertMarshalD(t,jsonString string) (xjson.MarshalD, xjson.MarshalD) {
	doc := xjson.MarshalD{}
	gc.So(json.Unmarshal([]byte(jsonString), &doc), gc.ShouldBeNil)


	dec, err := xjson.DecodeExtended(doc)
	gc.So(err, gc.ShouldBeNil)

	rDocStr, err := json.Marshal(dec)
	gc.So(err, gc.ShouldBeNil)

	rDoc := xjson.MarshalD{}
	gc.So(json.Unmarshal([]byte(rDocStr), &rDoc), gc.ShouldBeNil)

	return rDoc, doc
}

func TestDecodeExtended(t *testing.T) {
	gc.Convey("Round tripping a bson.D should work", t, func() {
		actual, expected := assertBsonD(t, docStr)
		gc.So(actual, gc.ShouldResemble, expected)
	})

	// We decode extended JSON `NaN` values to golang `math.NaN()` for bson,
	// which cannot be compared for equality or resemblance without using
	// `math.IsNaN`, hence this additional test.
	gc.Convey("Round tripping a bson.D with NaN values should work", t, func() {
		nanStr := `
			{
				"d1" : {"$numberDouble": "NaN"}
			}
		`
		actual, _ := assertBsonD(t, nanStr)

		d1Value := actual.Map()["d1"]
		isNan := math.IsNaN(d1Value.(float64))
		gc.So(isNan, gc.ShouldBeTrue)
	})

	gc.Convey("Round tripping a MarshalD should work", t, func() {
		actual, expected := assertMarshalD(docStr)
		gc.So(actual, gc.ShouldResemble, expected)
	})

	// We decode extended JSON `NaN` values to golang `math.NaN()` for MarshalD,
	// which cannot be compared for equality or resemblance without using
	// `math.IsNaN`, hence this additional test.
	gc.Convey("Round tripping a MarshalD with NaN values should work", t, func() {
		nanStr := `
			{
				"d1" : {"$numberDouble": "NaN"}
			}
		`
		actual, _ := assertMarshalD(nanStr)
		d1Value := bson.D(actual).Map()["d1"]
		isNan := math.IsNaN(d1Value.(float64))
		gc.So(isNan, gc.ShouldBeTrue)
	})

	gc.Convey("Types known to Stitch but not bsonutil should still be extended", t, func() {
		for _, tc := range []struct {
			Input    interface{}
			Expected interface{}
		}{
			{
				bson.D{
					{"binaryBytes", []byte("HELLO")},
					{"int", 42},
					{"bigInt", 2147483648},
					{"doubleNaN", math.NaN()},
					{"doublePosInf", math.Inf(1)},
					{"doubleNegInf", math.Inf(-1)},
				},
				map[string]interface{}{
					"binaryBytes":  map[string]interface{}{"$binary": "SEVMTE8=", "$type": "00"},
					"int":          map[string]interface{}{"$numberInt": "42"},
					"bigInt":       map[string]interface{}{"$numberLong": "2147483648"},
					"doubleNaN":    map[string]interface{}{"$numberDouble": "NaN"},
					"doublePosInf": map[string]interface{}{"$numberDouble": "Infinity"},
					"doubleNegInf": map[string]interface{}{"$numberDouble": "-Infinity"},
				},
			},
			{
				bson.D{
					{"binaryBytes", []byte("HELLO")},
					{"int", 42},
					{"bigInt", 2147483648},
					{"doubleNaN", math.NaN()},
					{"doublePosInf", math.Inf(1)},
					{"doubleNegInf", math.Inf(-1)},
				},
				map[string]interface{}{
					"binaryBytes":  map[string]interface{}{"$binary": "SEVMTE8=", "$type": "00"},
					"int":          map[string]interface{}{"$numberInt": "42"},
					"bigInt":       map[string]interface{}{"$numberLong": "2147483648"},
					"doubleNaN":    map[string]interface{}{"$numberDouble": "NaN"},
					"doublePosInf": map[string]interface{}{"$numberDouble": "Infinity"},
					"doubleNegInf": map[string]interface{}{"$numberDouble": "-Infinity"},
				},
			},
			{
				[]interface{}{
					bson.D{{"binaryBytes", []byte("HELLO")}},
					bson.D{{"int", 42}},
					bson.D{{"bigInt", 2147483648}},
					bson.D{{"doubleNaN", math.NaN()}},
					bson.D{{"doublePosInf", math.Inf(1)}},
					bson.D{{"doubleNegInf", math.Inf(-1)}},
				},
				[]interface{}{
					map[string]interface{}{"binaryBytes": map[string]interface{}{"$binary": "SEVMTE8=", "$type": "00"}},
					map[string]interface{}{"int": map[string]interface{}{"$numberInt": "42"}},
					map[string]interface{}{"bigInt": map[string]interface{}{"$numberLong": "2147483648"}},
					map[string]interface{}{"doubleNaN": map[string]interface{}{"$numberDouble": "NaN"}},
					map[string]interface{}{"doublePosInf": map[string]interface{}{"$numberDouble": "Infinity"}},
					map[string]interface{}{"doubleNegInf": map[string]interface{}{"$numberDouble": "-Infinity"}},
				},
			},
		} {
			rDocStr, err := xjson.NewValueOf(tc.Input).MarshalJSON()
			gc.So(err, gc.ShouldBeNil)
			var out interface{}
			gc.So(json.Unmarshal([]byte(rDocStr), &out), gc.ShouldBeNil)

			gc.So(out, gc.ShouldResemble, tc.Expected)
		}
	})

	gc.Convey("Error decoding should be returned", t, func() {
		for _, doc := range []interface{}{
			map[string]interface{}{"someLong": map[string]interface{}{"$numberLong": "hello"}},
			bson.M{"dbPointer": bson.M{"$dbPointer": bson.M{}}},
			bson.M{"dbPointer": bson.M{"$dbPointer": bson.M{"$ref": 1}}},
			bson.M{"dbPointer": bson.M{"$dbPointer": 1}},
			bson.M{"dbPointer": bson.M{"$dbPointer": bson.M{"$ref": "reference"}}},
			bson.M{"dbPointer": bson.M{"$dbPointer": bson.M{"$ref": "reference", "$id": "hello"}}},
			bson.M{"dbPointer": bson.M{"$dbPointer": bson.M{"$ref": "reference", "$id": bson.M{}}}},
			bson.M{"dbPointer": bson.M{"$dbPointer": bson.M{"$ref": "reference", "$id": bson.M{"$numberLong": 1}}}},
			bson.M{"date": bson.M{"$date": bson.M{}}},
			bson.M{"date": bson.M{"$date": bson.M{"$numberLong": 1}}},
			bson.M{"dbRef": bson.M{"$ref": "reference", "$id": bson.M{"$numberLong": 1}}},
			bson.M{"dbRef": bson.M{"$ref": 1, "$id": bson.M{"$numberLong": 1}}},
			bson.M{"dbRef": bson.M{"$ref": 1, "$id": bson.M{"$numberLong": "2"}, "$db": "test"}},
			bson.M{"dbRef": bson.M{"$ref": "1", "$id": bson.M{"$numberLong": 1}, "$db": "test"}},
			bson.M{"dbRef": bson.M{"$ref": "1", "$id": bson.M{"$numberLong": "1"}, "$db": 5}},
			bson.M{"binary": bson.M{"$binary": "1", "$type": "00"}},
			bson.M{"binary": bson.M{"$binary": 1, "$type": "00"}},
			bson.M{"binary": bson.M{"$binary": "o0w498Or7cijeBSpkquNtg==", "$other": "00"}},
			bson.M{"binary": bson.M{"$binary": "o0w498Or7cijeBSpkquNtg==", "$type": "0001"}},
			bson.M{"binary": bson.M{"$binary": "o0w498Or7cijeBSpkquNtg==", "$type": "hello"}},
			bson.M{"binary": bson.M{"$binary": "o0w498Or7cijeBSpkquNtg==", "$type": 0x00}},
			bson.M{"regex": bson.M{"$regex": 1, "$options": ""}},
			bson.M{"regex": bson.M{"$regex": "value", "$other": "00"}},
			bson.M{"regex": bson.M{"$regex": "value", "$options": "0001"}},
			bson.M{"regex": bson.M{"$regex": "value", "$options": 5}},
			bson.M{"code": bson.M{"$code": 1, "$options": ""}},
			bson.M{"code": bson.M{"$code": "value", "$other": "00"}},
			bson.M{"code": bson.M{"$code": "value", "$scope": "0001"}},
			bson.M{"code": bson.M{"$code": "value", "$scope": bson.M{"num": bson.M{"$numberLong": 1}}}},
			bson.M{"code": bson.M{"$code": "value", "$scope": 5}},
			bson.M{"symbol": bson.M{"$symbol": 5}},
			bson.M{"double": bson.M{"$numberDouble": 5}},
			bson.M{"timestamp": bson.M{"$timestamp": 5}},
			bson.M{"timestamp": bson.M{"$timestamp": "hello"}},
			bson.M{"int": bson.M{"$numberInt": "hello"}},
			bson.M{"int": bson.M{"$numberInt": 5}},
			bson.M{"oid": bson.M{"$oid": 5}},
			bson.M{"oid": bson.M{"$oid": "notanoid"}},
			bson.M{"code": bson.M{"$code": 01010101010}},
			bson.M{"date": bson.M{"$date": 2017}},
			bson.M{"decimal128": bson.M{"$numberDecimal": 2017}},
		} {
			_, err := xjson.DecodeExtended(doc)
			gc.So(err, gc.ShouldNotBeNil)
		}
	})

	gc.Convey("Decoding a standard pipeline argument instance should work", t, func() {
		type literalArgs struct {
			Items interface{} `json:"items"`
		}
		args := &literalArgs{
			Items: []interface{}{
				bson.D{
					{"_id", bson.D{{"$numberInt", "100"}}},
					{"decimal", bson.D{{"$numberDecimal", "100"}}},
				},
			},
		}
		d100, err := bson.ParseDecimal128("100")
		gc.So(err, gc.ShouldBeNil)
		decoded, err := xjson.DecodeExtended(args)
		gc.So(err, gc.ShouldBeNil)
		gc.So(decoded, gc.ShouldResemble, &literalArgs{
			Items: []interface{}{
				bson.D{
					{"_id", 100},
					{"decimal", d100},
				},
			},
		})
	})

	gc.Convey("Decoding should normally work", t, func() {
		ptrToBadOid := bson.D{{"a", bson.D{{"$oid", "nope"}}}}
		for _, tc := range []struct {
			in    interface{}
			out   interface{}
			errIn bool
		}{
			{nil, nil, false},
			{bson.D{{"a", bson.D{{"$oid", "nope"}}}}, nil, true},
			{[]interface{}{bson.D{{"$oid", "nope"}}}, nil, true},
			{&ptrToBadOid, nil, true},
			{xjson.MarshalD{{"a", xjson.MarshalD{{}}}}, xjson.MarshalD{{"a", xjson.MarshalD{{}}}}, false},
		} {
			decoded, err := xjson.DecodeExtended(tc.in)
			if tc.errIn {
				gc.So(err, gc.ShouldNotBeNil)
				continue
			}
			gc.So(err, gc.ShouldBeNil)
			gc.So(decoded, gc.ShouldResemble, tc.out)
		}
	})

	gc.Convey("Decoding a struct should work", t, func() {
		type someStruct struct {
			SomeVal1 xjson.Value  `json:"someVal1"`
			SomeVal2 *xjson.Value `json:"someVal2"`
			SomeVal3 *xjson.Value `json:"someVal3"`
			Nil      interface{}  `json:"nil"`
			anon     string       // nolint: megacheck
		}
		args := &someStruct{
			SomeVal1: *xjson.NewValueOf([]interface{}{
				bson.D{
					{"_id", bson.D{{"$numberInt", "100"}}},
					{"decimal", bson.D{{"$numberDecimal", "100"}}},
				},
			}),
			SomeVal2: xjson.NewValueOf([]interface{}{
				bson.D{
					{"_id", bson.D{{"$numberInt", "100"}}},
					{"decimal", bson.D{{"$numberDecimal", "100"}}},
				},
			}),
		}
		d100, err := bson.ParseDecimal128("100")
		gc.So(err, gc.ShouldBeNil)
		decoded, err := xjson.DecodeExtended(args)
		gc.So(err, gc.ShouldBeNil)
		gc.So(decoded, testutils.ShouldResembleUnorderedBsonDiffed, &someStruct{
			SomeVal1: *xjson.NewValueOf([]interface{}{
				bson.D{
					{"_id", 100},
					{"decimal", d100},
				},
			}),
			SomeVal2: xjson.NewValueOf([]interface{}{
				bson.D{
					{"_id", 100},
					{"decimal", d100},
				},
			}),
		})

		// Error decoding
		args.SomeVal1 = *xjson.NewValueOf([]interface{}{
			bson.D{
				{"_id", bson.D{{"$oid", "100"}}},
				{"decimal", bson.D{{"$numberDecimal", "100"}}},
			},
		})
		_, err = xjson.DecodeExtended(args)
		gc.So(err, gc.ShouldNotBeNil)
	})
}

func TestEncodeExtended(t *testing.T) {
	gc.Convey("Applicable values should be extended", t, func() {
		str := "hello"

		oid := bson.NewObjectId()

		for _, tc := range []struct {
			in     interface{}
			out    interface{}
			outStr string
		}{
			{"hello", "hello", `"hello"`},
			{
				bson.D{{"1", 1}, {"2", 2}, {"3", 3}},
				bson.D{
					{"1", bson.D{{"$numberInt", "1"}}},
					{"2", bson.D{{"$numberInt", "2"}}},
					{"3", bson.D{{"$numberInt", "3"}}},
				},
				`{"1":{"$numberInt":"1"},"2":{"$numberInt":"2"},"3":{"$numberInt":"3"}}`,
			},
			{nil, nil, `null`},
			{[]interface{}{"1", nil, "2"}, []interface{}{"1", nil, "2"}, `["1",null,"2"]`},
			{[]interface{}{"1", &str}, []interface{}{"1", str}, `["1","hello"]`},
			{1, map[string]interface{}{"$numberInt": "1"}, `{"$numberInt":"1"}`},
			{oid, map[string]interface{}{"$oid": oid.Hex()}, fmt.Sprintf(`{"$oid":"%s"}`, oid.Hex())},
			{
				bson.D{{"time", time.Unix(0, 20)}},
				bson.D{{"time", bson.D{{"$date", bson.D{{"$numberLong", "0"}}}}}},
				`{"time":{"$date":{"$numberLong":"0"}}}`,
			},
			{
				bson.D{{"ptr", bson.DBPointer{Namespace: "one.two", Id: oid}}},
				bson.D{{"ptr", bson.D{{"$dbPointer", bson.D{{"$ref", "one.two"}, {"$id", bson.D{{"$oid", oid.Hex()}}}}}}}},
				fmt.Sprintf(`{"ptr":{"$dbPointer":{"$ref":"one.two","$id":{"$oid":"%s"}}}}`, oid.Hex()),
			},
		} {
			md, err := json.Marshal(xjson.NewValueOf(tc.in))
			gc.So(err, gc.ShouldBeNil)
			gc.So(string(md), gc.ShouldEqual, tc.outStr)
		}
	})
}

func TestExtendedJSONSpecEncode(t *testing.T) {
	uUID, err := base64.StdEncoding.DecodeString("o0w498Or7cijeBSpkquNtg==")
	if err != nil {
		t.Error(err)
		return
	}
	dec128, err := bson.ParseDecimal128("-1.869E5")
	if err != nil {
		t.Error(err)
		return
	}

	inputDoc := bson.D{
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
		{"CodeWithScope", bson.JavaScript{Code: "function() {}", Scope: bson.D{}}},
		{"Subdocument", bson.D{{"foo", "bar"}}},
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
		{"DBRef", xjson.DBRef{
			Collection: "collection",
			Id:         bson.ObjectIdHex("57fd71e96e32ab4225b723fb"),
			Database:   "database",
		}},
		{"DBRef2", xjson.DBRef{
			Collection: "collection",
			Id:         "test",
			Database:   "database",
		}},
		{"Minkey", bson.MinKey},
		{"Maxkey", bson.MaxKey},
		{"Null", nil},
		{"NullInDoc", bson.D{
			{"n", nil},
		}},
		{"Undefined", bson.Undefined},
		{"Decimal", dec128},
		{"DecimalNaN", xjson.Decimal128NaN},
		{"DecimalInf", xjson.Decimal128Inf},
		{"DecimalNegInf", xjson.Decimal128NegInf},
	}

	expectedDecodedDoc := bson.D{
		{"_id", bson.ObjectIdHex("57e193d7a9cc81b4027498b5")},
		{"Symbol", bson.Symbol("symbol")},
		{"String", "string"},
		{"Int32", 42},
		{"Int64", int64(42)},
		{"Double", 42.42},
		{"DoubleInf", math.Inf(1)},
		{"DoubleNegInf", math.Inf(-1)},
		{"Binary", bson.Binary{Kind: 0x03, Data: uUID}},
		{"BinaryUserDefined", bson.Binary{Kind: 0x80, Data: []byte{1, 2, 3, 4, 5}}},
		{"Code", bson.JavaScript{Code: "function() {}"}},
		{"CodeWithScope", bson.JavaScript{Code: "function() {}", Scope: bson.D{}}},
		{"Subdocument", bson.D{{"foo", "bar"}}},
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
		{"DBRef", bson.D{
			{"$ref", "collection"},
			{"$id", bson.D{
				{"$oid", "57fd71e96e32ab4225b723fb"},
			}},
			{"$db", "database"},
		}},
		{"DBRef2", bson.D{
			{"$ref", "collection"},
			{"$id", "test"},
			{"$db", "database"},
		}},
		{"Minkey", bson.MinKey},
		{"Maxkey", bson.MaxKey},
		{"Null", nil},
		{"NullInDoc", bson.D{
			{"n", nil},
		}},
		{"Undefined", bson.Undefined},
		{"Decimal", dec128},
		{"DecimalNaN", xjson.Decimal128NaN},
		{"DecimalInf", xjson.Decimal128Inf},
		{"DecimalNegInf", xjson.Decimal128NegInf},
	}

	const expectedEncodedStr = `{"_id":{"$oid":"57e193d7a9cc81b4027498b5"},"Symbol":{"$symbol":"symbol"},"String":"string","Int32":{"$numberInt":"42"},"Int64":{"$numberLong":"42"},"Double":{"$numberDouble":"42.42"},"DoubleInf":{"$numberDouble":"Infinity"},"DoubleNegInf":{"$numberDouble":"-Infinity"},"Binary":{"$binary":"o0w498Or7cijeBSpkquNtg==","$type":"03"},"BinaryUserDefined":{"$binary":"AQIDBAU=","$type":"80"},"Code":{"$code":"function() {}"},"CodeWithScope":{"$code":"function() {}","$scope":{}},"Subdocument":{"foo":"bar"},"Array":[{"$numberInt":"1"},{"$numberInt":"2"},{"$numberInt":"3"},{"$numberInt":"4"},{"$numberInt":"5"}],"Timestamp":{"$timestamp":"180388626433"},"Regex":{"$regex":"pattern","$options":""},"DatetimeEpoch":{"$date":{"$numberLong":"0"}},"DatetimePositive":{"$date":{"$numberLong":"9223372036854775807"}},"DatetimeNegative":{"$date":{"$numberLong":"-9223372036854775808"}},"True":true,"False":false,"DBPointer":{"$dbPointer":{"$ref":"db.collection","$id":{"$oid":"57e193d7a9cc81b4027498b1"}}},"DBRef":{"$ref":"collection","$id":{"$oid":"57fd71e96e32ab4225b723fb"},"$db":"database"},"DBRef2":{"$ref":"collection","$id":"test","$db":"database"},"Minkey":{"$minKey":1},"Maxkey":{"$maxKey":1},"Null":null,"NullInDoc":{"n":null},"Undefined":{"$undefined":true},"Decimal":{"$numberDecimal":"-1.869E+5"},"DecimalNaN":{"$numberDecimal":"NaN"},"DecimalInf":{"$numberDecimal":"Infinity"},"DecimalNegInf":{"$numberDecimal":"-Infinity"}}`

	gc.Convey("Encoding input should match output", t, func() {
		// Round trip; exclude NaN
		inputWithoutNan := append(bson.D{}, inputDoc[0:6]...)
		inputWithoutNan = append(inputWithoutNan, inputDoc[7:]...)

		md, err := json.Marshal(xjson.NewValueOf(inputWithoutNan))
		gc.So(err, gc.ShouldBeNil)
		gc.So(string(md), gc.ShouldEqual, expectedEncodedStr)

		var rt xjson.MarshalD
		gc.So(json.Unmarshal(md, &rt), gc.ShouldBeNil)
		gc.So(bson.D(rt), gc.ShouldResemble, expectedDecodedDoc)
	})
}

func TestExtendedJSONSpecDecode(t *testing.T) {
	inputDoc := bson.M{
		"_id": bson.M{
			"$oid": "57e193d7a9cc81b4027498b5",
		},
		"Symbol": bson.M{
			"$symbol": "symbol",
		},
		"String": "string",
		"Int32": bson.M{
			"$numberInt": "42",
		},
		"Int64": bson.M{
			"$numberLong": "42",
		},
		"Double": bson.M{
			"$numberDouble": "42.42",
		},
		"DoubleInf": bson.M{
			"$numberDouble": "Infinity",
		},
		"DoubleNegInf": bson.M{
			"$numberDouble": "-Infinity",
		},
		"Binary": bson.M{
			"$binary": "o0w498Or7cijeBSpkquNtg==",
			"$type":   "03",
		},
		"BinaryUserDefined": bson.M{
			"$binary": "AQIDBAU=",
			"$type":   "80",
		},
		"Code": bson.M{
			"$code": "function() {}",
		},
		"CodeWithScope": bson.M{
			"$code":  "function() {}",
			"$scope": bson.M{},
		},
		"Subdocument": bson.M{
			"foo": "bar",
		},
		"Array": []interface{}{
			bson.M{"$numberInt": "1"},
			bson.M{"$numberInt": "2"},
			bson.M{"$numberInt": "3"},
			bson.M{"$numberInt": "4"},
			bson.M{"$numberInt": "5"},
		},
		"Timestamp": bson.M{
			"$timestamp": "180388626433",
		},
		"Regex": bson.M{
			"$regex":   "pattern",
			"$options": "",
		},
		"DatetimeEpoch": bson.M{
			"$date": bson.M{
				"$numberLong": "0",
			},
		},
		"DatetimePositive": bson.M{
			"$date": bson.D{{
				"$numberLong", "9223372036854775807",
			}},
		},
		"DatetimeNegative": bson.M{
			"$date": xjson.MarshalD{{
				"$numberLong", "-9223372036854775808",
			}},
		},
		"True":  true,
		"False": false,
		"DBPointer": bson.M{
			"$dbPointer": bson.M{
				"$ref": "db.collection",
				"$id": bson.M{
					"$oid": "57e193d7a9cc81b4027498b1",
				},
			},
		},
		"DBRef": bson.M{
			"$ref": "collection",
			"$id": bson.M{
				"$oid": "57fd71e96e32ab4225b723fb",
			},
			"$db": "database",
		},
		"DBRef2": bson.M{
			"$ref": "collection",
			"$id":  "test",
			"$db":  "database",
		},
		"DBRef3": bson.M{
			"$ref": "collection",
			"$id":  "test",
		},
		"Minkey": bson.M{
			"$minKey": 1,
		},
		"Maxkey": bson.M{
			"$maxKey": 1,
		},
		"Null": nil,
		"NullInDoc": bson.M{
			"n": nil,
		},
		"Undefined": bson.M{
			"$undefined": true,
		},
		"Decimal": bson.M{
			"$numberDecimal": "-1.869E+5",
		},
		"DecimalInf": bson.M{
			"$numberDecimal": "Infinity",
		},
		"DecimalNegInf": bson.M{
			"$numberDecimal": "-Infinity",
		},
		"DecimalNaN": bson.M{
			"$numberDecimal": "NaN",
		},
	}

	uUID, err := base64.StdEncoding.DecodeString("o0w498Or7cijeBSpkquNtg==")
	if err != nil {
		t.Error(err)
		return
	}
	dec128, err := bson.ParseDecimal128("-1.869E5")
	if err != nil {
		t.Error(err)
		return
	}

	expectedDoc := bson.M{
		"_id":               bson.ObjectIdHex("57e193d7a9cc81b4027498b5"),
		"Symbol":            bson.Symbol("symbol"),
		"String":            "string",
		"Int32":             42,
		"Int64":             int64(42),
		"Double":            42.42,
		"DoubleInf":         math.Inf(1),
		"DoubleNegInf":      math.Inf(-1),
		"Binary":            bson.Binary{Kind: 0x03, Data: uUID},
		"BinaryUserDefined": bson.Binary{Kind: 0x80, Data: []byte{1, 2, 3, 4, 5}},
		"Code":              bson.JavaScript{Code: "function() {}"},
		"CodeWithScope":     bson.JavaScript{Code: "function() {}", Scope: bson.M{}},
		"Subdocument":       bson.M{"foo": "bar"},
		"Array":             []interface{}{1, 2, 3, 4, 5},
		"Timestamp":         bson.MongoTimestamp((42 << 32) | 1),
		"Regex":             bson.RegEx{Pattern: "pattern"},
		"DatetimeEpoch":     time.Unix(0, 0),
		"DatetimePositive":  time.Unix(int64(math.MaxInt64)/1e3, int64(math.MaxInt64)%1e3*1e6),
		"DatetimeNegative":  time.Unix(int64(math.MinInt64)/1e3, int64(math.MinInt64)%1e3*1e6),
		"True":              true,
		"False":             false,
		"DBPointer": bson.DBPointer{
			Namespace: "db.collection",
			Id:        bson.ObjectIdHex("57e193d7a9cc81b4027498b1"),
		},
		"DBRef": bson.M{
			"$ref": "collection",
			"$id": bson.M{
				"$oid": "57fd71e96e32ab4225b723fb",
			},
			"$db": "database",
		},
		"DBRef2": bson.M{
			"$ref": "collection",
			"$id":  "test",
			"$db":  "database",
		},
		"DBRef3": bson.M{
			"$ref": "collection",
			"$id":  "test",
		},
		"Minkey":        bson.MinKey,
		"Maxkey":        bson.MaxKey,
		"Null":          nil,
		"NullInDoc":     bson.M{"n": nil},
		"Undefined":     bson.Undefined,
		"Decimal":       dec128,
		"DecimalInf":    xjson.Decimal128Inf,
		"DecimalNegInf": xjson.Decimal128NegInf,
		"DecimalNaN":    xjson.Decimal128NaN,
	}

	gc.Convey("Encoding input should match output", t, func() {
		decoded, err := xjson.DecodeExtended(inputDoc)
		gc.So(err, gc.ShouldBeNil)
		gc.So(decoded, gc.ShouldResemble, expectedDoc)

		// Decoding twice should have no effect
		decoded, err = xjson.DecodeExtended(decoded)
		gc.So(err, gc.ShouldBeNil)
		gc.So(decoded, gc.ShouldResemble, expectedDoc)
	})
}
