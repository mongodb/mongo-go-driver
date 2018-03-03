package command

import (
	"errors"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/result"
)

// unmarshalFindAndModifyResult turns the provided bson.Reader into a findAndModify result.
func unmarshalFindAndModifyResult(rdr bson.Reader) (result.FindAndModify, error) {
	var res result.FindAndModify
	val, err := rdr.Lookup("value")
	switch {
	case err == bson.ErrElementNotFound:
		return result.FindAndModify{}, errors.New("invalid response from server, no value field")
	case err != nil:
		return result.FindAndModify{}, err
	}

	switch val.Value().Type() {
	case bson.TypeNull:
	case bson.TypeEmbeddedDocument:
		res.Value = val.Value().ReaderDocument()
	default:
		return result.FindAndModify{}, errors.New("invalid response from server, 'value' field is not a document")
	}

	if val, err := rdr.Lookup("lastErrorObject", "updatedExisting"); err == nil {
		b, ok := val.Value().BooleanOK()
		if ok {
			res.LastErrorObject.UpdatedExisting = b
		}
	}

	if val, err := rdr.Lookup("lastErrorObject", "upserted"); err == nil {
		oid, ok := val.Value().ObjectIDOK()
		if ok {
			res.LastErrorObject.Upserted = oid
		}
	}
	return res, nil
}
