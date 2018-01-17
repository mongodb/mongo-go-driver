package msg

import (
	"bytes"

	oldbson "github.com/10gen/mongo-go-driver/bson"
	"github.com/skriptble/wilson/bson"
)

func unmarshal(b []byte, v interface{}) error {
	switch t := v.(type) {
	case *bson.Document:
		return bson.NewDecoder(bytes.NewReader(b)).Decode(v)
	case bson.Unmarshaler:
		return t.UnmarshalBSON(b)
	default:
		return oldbson.Unmarshal(b, v)
	}

	return nil
}
