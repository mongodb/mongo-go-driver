package mongo

import (
	"errors"
	"strings"

	"github.com/skriptble/wilson/bson"
	"github.com/skriptble/wilson/bson/objectid"
)

func transformDocument(document interface{}) (*bson.Document, error) {
	switch document.(type) {
	case nil:
		return bson.NewDocument(), nil
	default:
		return bson.NewDocumentEncoder().EncodeDocument(document)
	}
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

func ensureDollarKey(doc *bson.Document) error {
	elem, err := doc.ElementAt(0)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(elem.Key(), "$") {
		return errors.New("update document must contain key beginning with '$'")
	}
	return nil
}
