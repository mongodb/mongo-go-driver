package extjson

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/bson/internal/json"
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
		v := json.EncodeFloatProperlyFormatted(x)
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
		// Overflow here is undefined
		buff.WriteString(`{"$date":{"$numberLong":"`)
		buff.WriteString(fmt.Sprintf("%d", (x.Unix()*1e3)+(int64(x.Nanosecond())/1e6)))
		buff.WriteString(`"}}`)
	case bson.RegEx:
		buff.WriteString(`{"$regularExpression":{"pattern":"`)
		buff.WriteString(escapeJSONMetaCharacters(x.Pattern))
		buff.WriteString(`","options":"`)

		// x.Options should be alphabetically ordered
		var optionsArr []string
		for _, ch := range x.Options {
			optionsArr = append(optionsArr, string(ch))
		}
		sort.Strings(optionsArr)
		sortedOptions := strings.Join(optionsArr, "")

		buff.WriteString(sortedOptions)
		buff.WriteString(`"}}`)
	case bson.DBPointer:
		buff.WriteString(`{"$dbPointer":{"$ref":"`)
		buff.WriteString(x.Namespace)
		buff.WriteString(`","$id":`)
		encodeOIDToBuffer(x.Id, buff)
		buff.WriteString(`}}`)
	case bson.JavaScript:
		buff.WriteString(`{"$code":"`)

		// Remove all null characters within x.Code and replace them with unicode representation of \u0000
		// Also get any non utf-8 values and convert them to their escaped unicode representation.
		for _, char := range x.Code {
			rn, size := utf8.DecodeLastRuneInString(string(char))
			if size > 1 || rn == 0 {
				quoted := strconv.QuoteRuneToASCII(rn) // quoted = "'\u554a'"
				unquoted := quoted[1 : len(quoted)-1]  // unquoted = "\u554a"
				if unquoted == "\\x00" {
					buff.WriteString("\\u0000")
				} else {
					buff.WriteString(unquoted)
				}
			} else {
				buff.WriteString(string(char))
			}
		}

		buff.WriteString(`"`)
		if x.Scope == nil {
			buff.WriteString(`}`)
			break
		}

		// Need this switch as the contents of x.Scope should be marshalled into extJSON as well. However its currently
		// being representated as a map, which becomes serialized as is. Thus we convert it into a bson.D which will then
		// be serialized correctly.
		switch x.Scope.(type) {
		case bson.M:
			d := bson.D{}
			d.AppendMap(x.Scope.(bson.M))
			x.Scope = d
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
		if x < math.MinInt32 || x > math.MaxInt32 {
			encodeInt64ToBuffer(int64(x), buff)
			break
		}
		encodeInt32ToBuffer(int32(x), buff)
	case int32:
		encodeInt32ToBuffer(x, buff)
	case int64:
		encodeInt64ToBuffer(x, buff)
	case bson.MongoTimestamp:
		buff.WriteString(`{"$timestamp":{"t":`)
		buff.WriteString(fmt.Sprintf("%d", uint32(x>>32)))
		buff.WriteString(`,"i":`)
		buff.WriteString(fmt.Sprintf("%d", uint32(x)))
		buff.WriteString(`}}`)
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

func escapeJSONMetaCharacters(s string) string {
	metaChars := [8]string{"\\", "\"", "\b", "\t", "\f", "\n", "\r", "\t"}
	for _, ch := range metaChars {
		esc := "\\" + ch
		s = strings.Replace(s, ch, esc, -1)
	}
	return s
}
