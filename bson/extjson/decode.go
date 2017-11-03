package extjson

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"
	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/stitch/utils"
)

// DecodeExtended decodes any extended JSON values in the given value.
// DecodeExtended is safe to call multiple times on the same value.
// Information might be lost due to inaccessible struct fields and as
// such should only really be used once on a new value.
func DecodeExtended(value interface{}) (interface{}, error) {
	value, isPtr, isValid := utils.Deref(value)
	if value == nil || !isValid {
		return value, nil
	}
	if isPtr {
		decoded, err := DecodeExtended(value)
		if err != nil {
			return nil, err
		}
		return utils.Ref(decoded), nil
	}

	switch x := value.(type) {
	case bson.D:
		return decodeExtendedBsonD(x)
	case MarshalD:
		doc, err := decodeExtendedBsonD(bson.D(x))
		return MarshalD(doc), err
	case bson.M:
		doc, err := decodeExtendedBsonD(MapAsBsonD(map[string]interface{}(x)))
		return doc.Map(), err
	case map[string]interface{}:
		doc, err := decodeExtendedBsonD(MapAsBsonD(x))
		return map[string]interface{}(doc.Map()), err
	case []interface{}:
		arr := make([]interface{}, len(x))
		for i, elem := range x {
			decoded, err := decodeSpecial(elem)
			if err != nil {
				return nil, err
			}
			arr[i] = decoded
		}
		return arr, nil
	case Value:
		decoded, err := decodeSpecial(x.V)
		if err != nil {
			return nil, err
		}
		return ValueOf(decoded), nil
	default:
		if utils.IsStruct(value) && !canExtend(value) {
			return utils.StructValueMapWithErr(value, decodeSpecial)
		}
		return value, nil
	}
}

func decodeExtendedBsonD(doc bson.D) (bson.D, error) {
	out := make(bson.D, len(doc))
	for i, docElem := range doc {
		decoded, err := decodeSpecial(docElem.Value)
		if err != nil {
			return nil, err
		}
		out[i] = bson.DocElem{docElem.Name, decoded}
	}
	return out, nil
}

// Decodes the provided object as a single extended JSON value
func decodeSpecial(value interface{}) (interface{}, error) {
	if value == nil {
		return value, nil
	}

	switch value.(type) {
	case bson.D, MarshalD, bson.M, map[string]interface{}:
		return parseSpecialKeys(value)
	}
	return DecodeExtended(value)
}

const (
	decValueInfinity    = "Infinity"
	decValueNegInfinity = "-Infinity"
	decValueNaN         = "NaN"
)

// parseSpecialKeys takes a document and inspects it for any extended JSON
// type (e.g $numberLong) and replaces any such values with the corresponding
// BSON type.
func parseSpecialKeys(special interface{}) (interface{}, error) {
	// first ensure we are using a correct document type
	var doc map[string]interface{}
	switch v := special.(type) {
	case bson.D:
		doc = v.Map()
	case MarshalD:
		doc = bson.D(v).Map()
	case bson.M:
		doc = map[string]interface{}(v)
	case map[string]interface{}:
		doc = v
	default:
		return nil, fmt.Errorf("%v (type %T) is not valid input to parseSpecialKeys", special, special)
	}
	// check document to see if it is special
	switch len(doc) {
	case 1: // document has a single field
		var key string
		var value interface{}
		for key, value = range doc {
		}
		switch key {
		case "$date":
			switch v := value.(type) {
			case bson.D:
				return decodeDate(v.Map())
			case MarshalD:
				return decodeDate(bson.D(v).Map())
			case map[string]interface{}:
				return decodeDate(v)
			case bson.M:
				return decodeDate(map[string]interface{}(v))
			default:
				return nil, errors.New("invalid type for $date field")
			}
		case "$code":
			if v, ok := value.(string); ok {
				return bson.JavaScript{Code: v}, nil
			}
			return nil, errors.New("expected $code field to have string value")
		case "$oid":
			if v, ok := value.(string); ok {
				if !bson.IsObjectIdHex(v) {
					return nil, errors.New("expected $oid field to contain 24 hexadecimal character")
				}
				return bson.ObjectIdHex(v), nil
			}
			return nil, errors.New("expected $oid field to have string value")
		case "$numberLong":
			return parseNumberLongField(value)
		case "$numberInt":
			if v, ok := value.(string); ok {
				// all of decimal, hex, and octal are supported here
				n, err := strconv.ParseInt(v, 0, 32)
				return int(n), err
			}
			return nil, errors.New("expected $numberInt field to have string value")
		case "$timestamp":
			v, ok := value.(string)
			if !ok || v == "" {
				return nil, errors.New("expected $timestamp key to be a string")
			}
			parsed, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				return nil, errors.New("expected $timestamp key to be a string representing a 64-bit unsigned timestamp")
			}
			return bson.MongoTimestamp(parsed), nil
		case "$numberDouble":
			if v, ok := value.(string); ok {
				switch v {
				case decValueNaN:
					return math.NaN(), nil
				case decValueInfinity:
					return math.Inf(1), nil
				case decValueNegInfinity:
					return math.Inf(-1), nil
				default:
					return strconv.ParseFloat(v, 64)
				}
			}
			return nil, errors.New("expected $numberDouble field to have string value")
		case "$numberDecimal":
			if v, ok := value.(string); ok {
				switch v {
				case decValueNaN:
					return Decimal128NaN, nil
				case decValueInfinity:
					return Decimal128Inf, nil
				case decValueNegInfinity:
					return Decimal128NegInf, nil
				default:
					return bson.ParseDecimal128(v)
				}
			}
			return nil, errors.New("expected $numberDecimal field to have string value")
		case "$dbPointer":
			switch v := value.(type) {
			case bson.D:
				return decodeDBPointer(v.Map())
			case MarshalD:
				return decodeDBPointer(bson.D(v).Map())
			case map[string]interface{}:
				return decodeDBPointer(v)
			case bson.M:
				return decodeDBPointer(map[string]interface{}(v))
			default:
				return nil, errors.New("expected $dbPointer field to have document value")
			}


		// TODO Steven: Clean up
		case "$binary":
			binary := bson.Binary{}
			// binary.Kind => subType field
			// binary.Data => whatever goes in base64

			value, ok := doc["$binary"]
			if !ok {
				return nil, errors.New("Not ok, could not parse")
			}
			valueBsonD := value.(bson.D)

			if len(valueBsonD) != 2 {
				return nil, errors.New("Binary BsonD not of valid length")
			}

			for _, doc := range valueBsonD {
				if doc.Name == "subType" {
					kind, _ := hex.DecodeString(doc.Value.(string))
					binary.Kind = kind[0]
				} else if doc.Name == "base64" {
					bytes, _ := base64.StdEncoding.DecodeString(doc.Value.(string))
					binary.Data = bytes
				}
			}
			return binary, nil

			//if value, ok := doc["$binary"]; ok {
			//	binary := bson.Binary{}
			//	v, ok := value.(string)
			//	if !ok {
			//		return nil, errors.New("expected $binary field to have string value")
			//	}
			//	bytes, err := base64.StdEncoding.DecodeString(v)
			//	if err != nil {
			//		return nil, err
			//	}
			//	binary.Data = bytes
			//
			//	if value, ok = doc["$type"]; ok {
			//		v, ok := value.(string)
			//		if !ok {
			//			return nil, errors.New("expected $type field to have string value")
			//		}
			//		kind, err := hex.DecodeString(v)
			//		if err != nil {
			//			return nil, err
			//		}
			//		if len(kind) != 1 {
			//			return nil, errors.New("expected single byte (as hexadecimal string) for $type field")
			//		}
			//		binary.Kind = kind[0]
			//		return binary, nil
			//	}
			//	return nil, errors.New("expected $type field with $binary field")
			//}

			//


		case "$symbol":
			v, ok := value.(string)
			if !ok {
				return nil, errors.New("expected $symbol field to have string value")
			}
			return bson.Symbol(v), nil
		case "$undefined":
			return bson.Undefined, nil
		case "$maxKey":
			return bson.MaxKey, nil
		case "$minKey":
			return bson.MinKey, nil
		}

	case 2: // document has two fields
		if value, ok := doc["$code"]; ok {
			code := bson.JavaScript{}
			v, ok := value.(string)
			if !ok {
				return nil, errors.New("expected $code field to have string value")
			}
			code.Code = v

			if value, ok = doc["$scope"]; ok {
				switch v := value.(type) {
				case map[string]interface{}, bson.D, bson.M, MarshalD:
					x, err := DecodeExtended(v)
					if err != nil {
						return nil, err
					}
					code.Scope = x
					return code, nil
				}
				return nil, errors.New("expected $scope field to contain map")
			}
			return nil, errors.New("expected $scope field with $code field")
		}

		if value, ok := doc["$regex"]; ok {
			regex := bson.RegEx{}
			v, ok := value.(string)
			if !ok {
				return nil, errors.New("expected $regex field to have string value")
			}
			regex.Pattern = v

			if value, ok = doc["$options"]; ok {
				v, ok = value.(string)
				if !ok {
					return nil, errors.New("expected $options field to have string value")
				}
				regex.Options = v

				// Validate regular expression options
				for i := range regex.Options {
					switch o := regex.Options[i]; o {
					case 'g', 'i', 'm', 's': // allowed
					default:
						return nil, fmt.Errorf("invalid regular expression option '%v'", o)
					}
				}
				return regex, nil
			}
			return nil, errors.New("expected $options field with $regex field")
		}


		//
		// No longer need this as DBRefs are obsolete
		//
		//if value, ok := doc["$ref"]; ok {
		//	if _, ok = value.(string); !ok {
		//		return nil, errors.New("expected string for $ref field")
		//	}
		//	if value, ok = doc["$id"]; ok {
		//		switch v := value.(type) {
		//		case map[string]interface{}, bson.D, bson.M, MarshalD:
		//			if _, err := parseSpecialKeys(v); err != nil {
		//				return nil, fmt.Errorf("error parsing $id field: %v", err)
		//			}
		//		}
		//
		//		// We do not care for dbRef as a typed object as it's not real BSON
		//		return special, nil
		//	}
		//}
	case 3:
		if value, ok := doc["$ref"]; ok {
			if _, ok = value.(string); !ok {
				return nil, errors.New("expected string for $ref field")
			}
			if value, ok = doc["$id"]; ok {
				switch v := value.(type) {
				case map[string]interface{}, bson.D, bson.M, MarshalD:
					if _, err := parseSpecialKeys(v); err != nil {
						return nil, fmt.Errorf("error parsing $id field: %v", err)
					}
				}
				if value, ok = doc["$db"]; ok {
					if _, ok = value.(string); !ok {
						return nil, errors.New("expected string for $db field")
					}

					// We do not care for dbRef as a typed object as it's not real BSON
					return special, nil
				}
			}
		}
	}

	// nothing matched, so we recurse deeper
	return DecodeExtended(special)
}

func decodeDate(v map[string]interface{}) (time.Time, error) {
	if jsonValue, ok := v["$numberLong"]; ok {
		n, err := parseNumberLongField(jsonValue)
		if err != nil {
			return time.Time{}, err
		}
		return time.Unix(n/int64(1e3), (n%int64(1e3))*int64(1e6)).UTC(), err
	}
	return time.Time{}, errors.New("expected $numberLong field in $date")
}

func parseNumberLongField(value interface{}) (int64, error) {
	if v, ok := value.(string); ok {
		return strconv.ParseInt(v, 10, 64)
	}
	return 0, errors.New("expected $numberLong field to have string value")
}

func decodeDBPointer(v map[string]interface{}) (bson.DBPointer, error) {
	pointer := bson.DBPointer{}

	ref, ok := v["$ref"].(string)
	if !ok || ref == "" {
		return pointer, errors.New("expected $ref field as string in $dbPointer")
	}
	pointer.Namespace = ref

	id, ok := v["$id"]
	if !ok {
		return pointer, errors.New("expected $id field as ObjectId in $dbPointer")
	}

	switch v := id.(type) {
	case map[string]interface{}, bson.D, bson.M, MarshalD:
		parsedID, err := parseSpecialKeys(v)
		if err != nil {
			return pointer, fmt.Errorf("error parsing $id field: %v", err)
		}
		actualID, ok := parsedID.(bson.ObjectId)
		if !ok {
			return pointer, errors.New("expected $id field as ObjectId in $dbPointer")
		}
		pointer.Id = actualID
	default:
		return pointer, errors.New("expected $id field as ObjectId in $dbPointer")
	}

	return pointer, nil
}

func canExtend(v interface{}) bool {
	if v == nil {
		return false
	}
	switch v.(type) {
	case int, int32, int64, float64, []byte, bson.Binary, bson.ObjectId,
		time.Time, bson.MongoTimestamp, bson.RegEx, bson.JavaScript,
		bson.Decimal128, bson.Symbol, bson.DBPointer, DBRef:
		return true
	}
	switch v {
	case bson.Undefined, bson.MaxKey, bson.MinKey:
		return true
	}
	return false
}
