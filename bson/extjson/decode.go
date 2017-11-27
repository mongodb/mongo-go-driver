package extjson

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"

	"github.com/10gen/mongo-go-driver/bson"
)

// DecodeExtended decodes any extended JSON values in the given value.
// DecodeExtended is safe to call multiple times on the same value.
// Information might be lost due to inaccessible struct fields and as
// such should only really be used once on a new value.
func DecodeExtended(value interface{}) (interface{}, error) {
	value, isPtr, isValid := deref(value)
	if value == nil || !isValid {
		return value, nil
	}
	if isPtr {
		decoded, err := DecodeExtended(value)
		if err != nil {
			return nil, err
		}
		return ref(decoded), nil
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
		if isStruct(value) && !canExtend(value) {
			return structValueMapWithErr(value, decodeSpecial)
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
	return ParseSpecialKeys(special)
}

func ParseSpecialKeys(special interface{}) (interface{}, error) {
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
			switch v := value.(type) {
			case bson.D:
				// Timestamp should only have 2 fields
				if len(v) != 2 {
					return nil, errors.New("Expected $timestamp value to contain only two fields")
				}
				var t, i, parsed uint64
				if v[0].Name == "t" {
					t = uint64(v[0].Value.(float64))
					i = uint64(v[1].Value.(float64))
				} else {
					t = uint64(v[1].Value.(float64))
					i = uint64(v[0].Value.(float64))
				}
				parsed = (t << 32)
				parsed = parsed + i
				return bson.MongoTimestamp(int64(parsed)), nil
			default:
				return nil, errors.New("expected $timestamp value to be a bson.D")
			}
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
		case "$binary":
			binary := bson.Binary{}
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
		case "$regularExpression":
			regex := bson.RegEx{}
			value, ok := doc["$regularExpression"]
			if !ok {
				return nil, errors.New("Not ok, could not parse regex")
			}
			valueBsonD := value.(bson.D)
			if len(valueBsonD) != 2 {
				return nil, errors.New("Regex BsonD not of valid length")
			}
			for _, doc := range valueBsonD {
				if doc.Name == "pattern" {
					v, ok := doc.Value.(string)
					if !ok {
						return nil, errors.New("expected $regex field to have string value")
					}
					regex.Pattern = v
				} else if doc.Name == "options" {
					v, ok := doc.Value.(string)
					if !ok {
						return nil, errors.New("expected $options field to have string value")
					}
					regex.Options = v

					// Validate regular expression options
					for i := range regex.Options {
						switch o := regex.Options[i]; o {
						case 'g', 'i', 'm', 's', 'x', 'l', 'u': // allowed
						default:
							return nil, fmt.Errorf("invalid regular expression option '%v'", o)
						}
					}
				}
			}
			return regex, nil
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

// Methods below were obtained from stitch/utils

// deref pulls an underlying object out of a pointer or interface, if applicable.
// It returns the dereferenced value where possible, otherwise just the given value.
// It also returns whether the original value was a pointer, and whether the
// inner value is valid.
func deref(v interface{}) (value interface{}, isPtr bool, isValid bool) {
	val := reflect.ValueOf(v)
	isPtr = val.Kind() == reflect.Ptr
	if isPtr || val.Kind() == reflect.Interface {
		val = val.Elem()
	}
	if !val.IsValid() {
		return v, isPtr, false
	}
	return val.Interface(), isPtr, true
}

// ref returns a pointer to the given concrete value.
func ref(v interface{}) interface{} {
	val := reflect.ValueOf(v)
	out := reflect.New(val.Type())
	out.Elem().Set(val)
	return out.Interface()
}

// isStruct returns whether the argument is a struct.
func isStruct(v interface{}) bool {
	return reflect.ValueOf(v).Kind() == reflect.Struct
}

// structValueMapWithErr applies a mapping to each exported value of the given struct, failing upon error.
// WARNING: use of this is discouraged, explore non-reflection alternatives before use.
func structValueMapWithErr(st interface{}, mapping func(interface{}) (interface{}, error)) (interface{}, error) {
	st, isPtr, _ := deref(st)
	if isPtr {
		out, err := structValueMapWithErr(st, mapping)
		return ref(out), err
	}
	if !isStruct(st) {
		return st, fmt.Errorf("expected struct, given %T", st)
	}
	val := reflect.ValueOf(st)
	vt := val.Type()
	out := reflect.New(vt).Elem()
	for fieldIdx := 0; fieldIdx < vt.NumField(); fieldIdx++ {
		if len(vt.Field(fieldIdx).PkgPath) != 0 {
			continue // unexported
		}
		field := val.Field(fieldIdx).Interface()
		newField, err := mapping(field)
		if err != nil {
			return nil, err
		}
		if newField == nil {
			continue
		}
		out.Field(fieldIdx).Set(reflect.ValueOf(newField))
	}
	return out.Interface(), nil
}
