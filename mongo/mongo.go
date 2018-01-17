package mongo

import (
	"errors"
	"strings"

	oldbson "github.com/10gen/mongo-go-driver/bson"
	"github.com/skriptble/wilson/bson"
	"github.com/skriptble/wilson/bson/builder"
	"github.com/skriptble/wilson/bson/objectid"
)

func transformDocument(document interface{}) (*bson.Document, error) {
	var bd *bson.Document
	var err error
	switch t := document.(type) {
	case nil:
		bd = bson.NewDocument(0)
	case *bson.Document:
		bd = t
	case bson.Reader, []byte:
		bd, err = bson.ReadDocument(t.(bson.Reader))
		if err != nil {
			return nil, err
		}
	case *builder.DocumentBuilder:
		buf := make([]byte, t.RequiredBytes())
		_, err = t.WriteDocument(buf)
		if err != nil {
			return nil, err
		}
		bd, _ = bson.ReadDocument(buf)
	default:
		// TODO(skriptble): Use a decoder (probably from a pool).
		buf, err := oldbson.Marshal(t)
		if err != nil {
			return nil, err
		}
		// NOTE: We just marshaled this into a valid BSON object.
		bd, _ = bson.ReadDocument(buf)
	}

	return bd, err
}

func ensureID(d *bson.Document) (interface{}, error) {
	var id interface{}

	elem, err := d.Lookup("_id")
	switch {
	case err == bson.ErrElementNotFound:
		oid := objectid.New()
		d.Append(bson.C.ObjectID("_id", oid))
		id = oid
	case err != nil:
		return nil, err
	default:
		id = elem
	}
	return id, nil
}

func ensureDollarKey2(doc *bson.Document) error {
	elem, err := doc.ElementAt(0)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(elem.Key(), "$") {
		return errors.New("update document must contain key beginning with '$'")
	}
	return nil
}
