package extjson

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/10gen/mongo-go-driver/bson"
)

// Special values for extended JSON.
var (
	Decimal128NaN, _    = bson.ParseDecimal128("NaN")
	Decimal128Inf, _    = bson.ParseDecimal128("Inf")
	Decimal128NegInf, _ = bson.ParseDecimal128("-Inf")
)

func encodeExtendedToBuffer(value interface{}, enc *json.Encoder, buff *bytes.Buffer) (bool, error) {
	switch x := value.(type) {
	case float64:
		var v string
		if math.IsNaN(x) {
			v = decValueNaN
		} else if math.IsInf(x, 1) {
			v = decValueInfinity
		} else if math.IsInf(x, -1) {
			v = decValueNegInfinity
		} else {
			v = strconv.FormatFloat(x, 'f', -1, 64)
		}
		buff.WriteString(`{"$numberDouble":"`)
		buff.WriteString(v)
		buff.WriteString(`"}`)
	case bson.Binary:
		buff.WriteString(`{"$binary":{"base64":"`)
		buff.WriteString(base64.StdEncoding.EncodeToString(x.Data))
		buff.WriteString(`","subType":"`)
		buff.WriteString(hex.EncodeToString([]byte{x.Kind}))
		buff.WriteString(`"}}`)
	case []byte:
		buff.WriteString(`{"$binary":{"base64":"`)
		buff.WriteString(base64.StdEncoding.EncodeToString(x))
		buff.WriteString(`","subType":"00"}}`)
	case bson.ObjectId:
		encodeOIDToBuffer(x, buff)
	case time.Time:
		// Overflow here is undefiend
		buff.WriteString(`{"$date":{"$numberLong":"`)
		buff.WriteString(fmt.Sprintf("%d", (x.Unix()*1e3)+(int64(x.Nanosecond())/1e6)))
		buff.WriteString(`"}}`)
	case bson.RegEx:
		buff.WriteString(`{"$regularExpression":{"pattern":"`)
		buff.WriteString(x.Pattern)
		buff.WriteString(`","options":"`)
		buff.WriteString(x.Options)
		buff.WriteString(`"}}`)
	case bson.DBPointer:
		buff.WriteString(`{"$dbPointer":{"$ref":"`)
		buff.WriteString(x.Namespace)
		buff.WriteString(`","$id":`)
		encodeOIDToBuffer(x.Id, buff)
		buff.WriteString(`}}`)
	case bson.JavaScript:
		buff.WriteString(`{"$code":"`)
		buff.WriteString(x.Code)
		buff.WriteString(`"`)

		if x.Scope == nil {
			buff.WriteString(`}`)
			break
		}

		buff.WriteString(`,"$scope":`)
		if err := encodeMarshalable(x.Scope, enc, buff, true); err != nil {
			return false, err
		}
		buff.WriteString(`}`)
	case bson.Symbol:
		buff.WriteString(`{"$symbol":"`)
		buff.WriteString(string(x))
		buff.WriteString(`"}`)
	case int:
		if x < -2147483648 || x > 2147483647 {
			encodeInt64ToBuffer(int64(x), buff)
			break
		}
		encodeInt32ToBuffer(int32(x), buff)
	case int32:
		encodeInt32ToBuffer(x, buff)
	case int64:
		encodeInt64ToBuffer(x, buff)
	case bson.MongoTimestamp:
		buff.WriteString(`{"$timestamp":"`)
		buff.WriteString(fmt.Sprintf("%d", x))
		buff.WriteString(`"}`)


	//expected: "{\"a\":{\"$timestamp\":{\"t\":123456789,\"i\":42}}}"
	//received: "{\"a\":{\"$timestamp\":\"530242871224172586\"}}"
	//<t> is the JSON representation of a 32-bit unsigned integer for seconds since epoch.
	//<i> is a 32-bit unsigned integer for the increment.

	case bson.Decimal128:
		var v string
		switch x {
		case Decimal128NaN:
			v = decValueNaN
		case Decimal128Inf:
			v = decValueInfinity
		case Decimal128NegInf:
			v = decValueNegInfinity
		default:
			v = x.String()
		}
		buff.WriteString(`{"$numberDecimal":"`)
		buff.WriteString(v)
		buff.WriteString(`"}`)
	case DBRef:
		buff.WriteString(`{"$ref":"`)
		buff.WriteString(x.Collection)
		buff.WriteString(`","$id":`)
		if err := encodeMarshalable(x.Id, enc, buff, true); err != nil {
			return false, err
		}

		if x.Database == "" {
			buff.WriteString(`}`)
			break
		}

		buff.WriteString(`,"$db":"`)
		buff.WriteString(x.Database)
		buff.WriteString(`"}`)
	default:
		switch value {
		case bson.MinKey:
			buff.WriteString(`{"$minKey":1}`)
		case bson.MaxKey:
			buff.WriteString(`{"$maxKey":1}`)
		case bson.Undefined:
			buff.WriteString(`{"$undefined":true}`)
		default:
			// not extendable
			return false, nil
		}
	}
	return true, nil
}

func encodeOIDToBuffer(id bson.ObjectId, buff *bytes.Buffer) {
	buff.WriteString(`{"$oid":"`)
	buff.WriteString(id.Hex())
	buff.WriteString(`"}`)
}

func encodeInt32ToBuffer(v int32, buff *bytes.Buffer) {
	buff.WriteString(`{"$numberInt":"`)
	buff.WriteString(fmt.Sprintf("%d", v))
	buff.WriteString(`"}`)
}

func encodeInt64ToBuffer(v int64, buff *bytes.Buffer) {
	buff.WriteString(`{"$numberLong":"`)
	buff.WriteString(fmt.Sprintf("%d", v))
	buff.WriteString(`"}`)
}
