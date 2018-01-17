package msg

import (
	"bytes"

	"github.com/skriptble/wilson/bson"
)

func unmarshal(b []byte, v interface{}) error {
	switch t := v.(type) {
	case *bson.Document:
		return bson.NewDecoder(bytes.NewReader(b)).Decode(v)
	case bson.Unmarshaler:
		return t.UnmarshalBSON(b)
	default:
		return bson.NewDecoder(bytes.NewReader(b)).Decode(v)
	}

	return nil
}
