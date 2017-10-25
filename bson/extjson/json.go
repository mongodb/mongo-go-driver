package extjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"runtime"
	"time"

	"github.com/10gen/stitch/common"
	"github.com/10gen/mongo-go-driver/bson"
)

// The DBRef type implements support for the database reference MongoDB
// convention as supported by multiple drivers.  This convention enables
// cross-referencing documents between collections and databases using
// a structure which includes a collection name, a document id, and
// optionally a database name.
//
// See the FindRef methods on Session and on Database.
//
// Relevant documentation:
//
//     http://www.mongodb.org/display/DOCS/Database+References
//
type DBRef struct {
	Collection string      `bson:"$ref"`
	Id         interface{} `bson:"$id"`
	Database   string      `bson:"$db,omitempty"`
}



// decodeJSONasBSON decodes an object from the stream as json, using bson.D to represented maps
// to maintain ordering information.
// Returns the object, a bool indicating if an object was present in the stream,
// and an error if one was encountered.
func decodeJSONasBSON(in io.Reader) (interface{}, bool, error) {
	dec := json.NewDecoder(in)
	depth := new(int)
	object, ok, err := decodeObject(dec, depth)

	if err == io.EOF {
		if *depth != 0 {
			return nil, false, fmt.Errorf("JSON object is incomplete")
		}
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	return object, ok, nil
}

// Value represents a JSON value. It can express a document, array, or any
// extended JSON primitive. It has special behavior what marshaling as BSON.
type Value struct {
	// V effectively conforms to the (recursive) union of the following types:
	// - bson.D
	// - []interface{}
	// - string
	// - float64
	// - bool
	// - nil (interface{})
	// The following types may also be present when unmarshaling bson rather than json.
	// - int
	// - int32
	// - int64
	// - []byte
	// - time.Time
	// - bson.Binary
	// - bson.Undefined
	// - bson.ObjectId
	// - bson.MongoTimestamp
	// - bson.Regex
	// - bson.DBPointer
	// - bson.Javascript
	// - bson.Symbol
	// - bson.Decimal128
	// - bson.MaxKey
	// - bson.MinKey
	V       interface{}
	Options ValueOptions
}

// Encrypt encrypts the underlying value of this value
func (v Value) Encrypt(crypter Crypter) (interface{}, error) {
	encValue, err := crypter.EncryptValue(v)
	if err != nil {
		return nil, err
	}
	v.V = encValue
	return v, nil
}

// Decrypt decrypts the underlying value of this value
func (v Value) Decrypt(crypter Crypter) (interface{}, error) {
	valueEnc, ok := v.V.(string)
	if !ok {
		return nil, errors.New("expected raw value to be encrypted as a string")
	}

	var decValue Value
	if err := crypter.DecryptValue(valueEnc, &decValue); err != nil {
		return nil, err
	}
	v.V = decValue.V
	return v, nil
}

// ValueOptions is a struct of available options for configuring the behavior of Values
type ValueOptions struct {
	DisableExtended bool
}

// NewValueOf turns a valid JSON value into a *Value type.
func NewValueOf(v interface{}) *Value {
	assertValidValue(v)
	return &Value{V: v}
}

// ValueOf turns a valid JSON value into a Value type.
func ValueOf(v interface{}) Value {
	assertValidValue(v)
	return Value{V: v}
}

func assertValidValue(v interface{}) {
	if !common.Debug {
		return
	}
	_, file, line, _ := runtime.Caller(2) // caller of ValueOf
	msg := fmt.Sprintf("%s:%d", file, line)
	assertValidValueInternal(v, msg)
}

func assertValidValueInternal(v interface{}, msg string) {
	switch x := v.(type) {
	case bson.D:
		for _, elem := range x {
			assertValidValueInternal(elem.Value, msg)
		}
	case *bson.D:
		for _, elem := range *x {
			assertValidValueInternal(elem.Value, msg)
		}
	case []interface{}:
		for _, item := range x {
			assertValidValueInternal(item, msg)
		}
	case *[]interface{}:
		for _, item := range *x {
			assertValidValueInternal(item, msg)
		}
	case string, float64, bool, int, int32, int64, []byte, time.Time,
		bson.Binary, bson.ObjectId, bson.MongoTimestamp, bson.RegEx,
		bson.DBPointer, bson.JavaScript, bson.Symbol, bson.Decimal128,
		DBRef:
	case *string, *float64, *bool, *int, *int32, *int64, []*byte, *time.Time,
		*bson.Binary, *bson.ObjectId, *bson.MongoTimestamp, *bson.RegEx,
		*bson.DBPointer, *bson.JavaScript, *bson.Symbol, *bson.Decimal128,
		*DBRef:
	default:
		switch v {
		case bson.Undefined, bson.MaxKey, bson.MinKey, nil:
		default:
			panic(fmt.Sprintf("invalid xjson.Value at %s (saw %T): %#v\n", msg, v, v))
		}
	}
}

// AsOptionalValue converts the value into a non-None OptionalValue.
func (v Value) AsOptionalValue() OptionalValue {
	return OptionalValue{v: v}
}

// GetDocItem tries to fetch a value by key from a supposed document.
func (v Value) GetDocItem(key string) OptionalValue {
	if d, ok := v.V.(bson.D); ok {
		for _, item := range d {
			if item.Name == key {
				return Value{V: item.Value}.AsOptionalValue()
			}
		}
	}
	return OptionalValue{none: true}
}

// NewWithOptions creates a new Value using the provided ValueOptions
// as its Options
func (v Value) NewWithOptions(options ValueOptions) Value {
	return Value{V: v.V, Options: options}
}

// MarshalJSON behaves like a normal JSON marshal on an
// interface except that the ordering of ordered maps' keys
// are preserved. If Extended JSON encoding has been disabled,
// MarshalJSON will serialize to normal, non-extended JSON
func (v Value) MarshalJSON() ([]byte, error) {
	var buff bytes.Buffer
	enc := json.NewEncoder(&buff)
	if err := encodeMarshalable(v.V, enc, &buff, !v.Options.DisableExtended); err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}

// GetBSON prepares the stored value to be stored as BSON.
// The form of a Value in MongoDB should always be a string.
// This is to circumvent rules MongoDB imposes on
// documents such as field naming conventions.
func (v Value) GetBSON() (interface{}, error) {
	md, err := v.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return string(md), nil
}

// SetBSON unmarshals a BSON value from the format described
// in GetBSON into the underlying value.
func (v *Value) SetBSON(raw bson.Raw) error {
	if raw.Kind != 0x02 {
		return fmt.Errorf("expected raw JSON string")
	}
	rawBytes := []byte{}
	if err := raw.Unmarshal(&rawBytes); err != nil {
		return err
	}

	return v.UnmarshalJSON(rawBytes)
}

// UnmarshalJSON behaves like a normal JSON unmarshal on an
// interface except that the ordering of Object keys is preserved
func (v *Value) UnmarshalJSON(in []byte) error {
	object, found, err := decodeJSONasBSON(bytes.NewReader(in))
	if err != nil {
		return err
	}

	if !found {
		*v = ValueOf(nil)
		return nil
	}

	decodedExtended, err := decodeSpecial(object)
	if err != nil {
		return err
	}

	*v = ValueOf(decodedExtended)
	return nil
}

// MarshalD represents a bson.D that preserves JSON Objects
// with keys ordered as they lexically appear
type MarshalD bson.D

// GetBSON prepares the JSON Object to be stored as BSON.
// The form of a MarshalD value in MongoDB should always be a
// JSON string. This is to circumvent rules MongoDB imposes
// on documents such as field naming conventions.
func (md MarshalD) GetBSON() (interface{}, error) {
	ret, err := md.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return string(ret), nil
}

// SetBSON unmarshals a BSON value from the format described
// in GetBSON into the underlying value
func (md *MarshalD) SetBSON(raw bson.Raw) error {
	if raw.Kind != 0x02 {
		return fmt.Errorf("expected raw JSON string")
	}
	var rawJSON string
	if err := raw.Unmarshal(&rawJSON); err != nil {
		return err
	}

	return md.UnmarshalJSON([]byte(rawJSON))
}

// UnmarshalJSON behaves like a normal JSON unmarshal on an
// interface except that the ordering of Object keys is preserved
func (md *MarshalD) UnmarshalJSON(in []byte) error {
	object, found, err := decodeJSONasBSON(bytes.NewReader(in))
	if err != nil {
		return err
	}

	if !found {
		return nil
	}

	decodedExtended, err := DecodeExtended(object)
	if err != nil {
		return err
	}

	if outAsD, ok := decodedExtended.(bson.D); ok {
		*md = MarshalD(outAsD)
		return nil
	}
	return fmt.Errorf("unable to decode '%s' into an 'object'", common.FriendlyTypeName(object))
}

// MarshalJSON behaves like a normal JSON marshal on an
// interface except that the ordering of ordered maps' keys
// are preserved
func (md MarshalD) MarshalJSON() ([]byte, error) {
	var buff bytes.Buffer
	enc := json.NewEncoder(&buff)

	if err := md.encodeJSON(enc, &buff, true); err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}

func (md MarshalD) encodeJSON(enc *json.Encoder, buff *bytes.Buffer, shouldExtend bool) error {
	buff.WriteString("{")
	for i, item := range md {
		if err := enc.Encode(item.Name); err != nil {
			return fmt.Errorf("cannot marshal key %v: %v", item.Name, err)
		}
		buff.Truncate(buff.Len() - 1)
		buff.WriteString(":")

		if err := encodeMarshalable(item.Value, enc, buff, shouldExtend); err != nil {
			return fmt.Errorf("cannot marshal value %v: %v", item.Value, err)
		}
		if i != len(md)-1 {
			buff.WriteString(",")
		}
	}
	buff.WriteString("}")
	return nil
}

// encodeMarshalable handles JSON encoding for types that have potentially already been
// encoded to Extended JSON. This method is initially called by `func (v Value) MarshalJSON()`
// and `func (md MarshalD) MarshalJSON()` before or after Extended JSON encoding has been completed.
// If shouldExtend is true, values will be extended and marshaled inline.
func encodeMarshalable(in interface{}, enc *json.Encoder, buff *bytes.Buffer, shouldExtend bool) error {
	if in == nil {
		buff.WriteString(`null`)
		return nil
	}

	switch typedVal := in.(type) {
	case *interface{}:
		if typedVal == nil {
			buff.WriteString(`null`)
			return nil
		}
		return encodeMarshalable(*typedVal, enc, buff, shouldExtend)
	case bson.D:
		return MarshalD(typedVal).encodeJSON(enc, buff, shouldExtend)
	case *bson.D:
		return MarshalD(*typedVal).encodeJSON(enc, buff, shouldExtend)
	case []interface{}:
		buff.WriteString("[")
		for i, v := range typedVal {
			if err := encodeMarshalable(v, enc, buff, shouldExtend); err != nil {
				return err
			}
			if i != len(typedVal)-1 {
				buff.WriteString(",")
			}
		}
		buff.WriteString("]")
		return nil
	default:
		if !shouldExtend {
			if err := enc.Encode(in); err != nil {
				return err
			}
			buff.Truncate(buff.Len() - 1)
			return nil
		}

		if didExtend, err := encodeExtendedToBuffer(in, enc, buff); err != nil {
			return err
		} else if !didExtend {
			if err := enc.Encode(in); err != nil {
				return err
			}
			buff.Truncate(buff.Len() - 1)
		}
		return nil
	}
}

// EncodeBSONDtoJSON wraps and marshals a bson.D as a MarshalD
func EncodeBSONDtoJSON(in bson.D) ([]byte, error) {
	return json.Marshal(MarshalD(in))
}

func decodeObject(dec *json.Decoder, depth *int) (interface{}, bool, error) {
	t, err := dec.Token()
	if err == io.EOF {
		return nil, false, err
	}
	if err != nil {
		return nil, false, err
	}
	switch typedToken := t.(type) {
	case json.Delim:
		switch tokStr := typedToken.String(); tokStr {
		case "{":
			(*depth)++
			v, err := decodeMap(dec, depth)
			if err != nil {
				return nil, false, err
			}
			return v, true, nil
		case "[":
			(*depth)++
			v, err := decodeArray(dec, depth)
			if err != nil {
				return nil, false, err
			}
			return v, true, nil
		case "]":
			(*depth)--
			// We're inside an array looking for an object, but there are none left.
			return nil, false, nil
		default:
			// Should be unreachable, because the json decoder detects this.
			return nil, false, fmt.Errorf("mismatched delimiters: not expecting a %v", tokStr)
		}
	default:
		return typedToken, true, nil
	}
}

func decodeArray(dec *json.Decoder, depth *int) (interface{}, error) {
	out := []interface{}{}
	for {
		val, ok, err := decodeObject(dec, depth)
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		out = append(out, val)
	}
	return out, nil
}

func decodeMap(dec *json.Decoder, depth *int) (interface{}, error) {
	out := bson.D{}
	for {
		t, err := dec.Token()
		if err != nil {
			return nil, err
		}
		if tDelim, ok := t.(json.Delim); ok {
			if tDelim.String() == "}" {
				(*depth)--
				return out, nil
			}
			// Should be unreachable, because the json decoder is supposed to detect this.
			return nil, fmt.Errorf("mismatched delimiters: wasn't expecting a %v", tDelim.String())
		}
		nextEntry := bson.DocElem{}
		if tStr, ok := t.(string); ok {
			nextEntry.Name = tStr
		} else {
			return nil, fmt.Errorf("expected a string key, but got %T", t)
		}
		val, ok, err := decodeObject(dec, depth)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("expected a value, but got nothing")
		}
		nextEntry.Value = val
		out = append(out, nextEntry)
	}
}
