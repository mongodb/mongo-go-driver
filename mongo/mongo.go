package mongo

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/bson/objectid"
)

// TranformDocument handles transforming a document of an allowable type into
// a *bson.Document. This method is called directly after most methods that
// have one or more parameters that are documents.
//
// The supported types for document are:
//
//  bson.Marshaler
//  bson.DocumentMarshaler
//  bson.Reader
//  []byte (must be a valid BSON document)
//  io.Reader (only 1 BSON document will be read)
//  A custom struct type
//
func TransformDocument(document interface{}) (*bson.Document, error) {
	switch d := document.(type) {
	case nil:
		return bson.NewDocument(), nil
	case *bson.Document:
		return d, nil
	case bson.Marshaler, bson.Reader, []byte, io.Reader:
		return bson.NewDocumentEncoder().EncodeDocument(document)
	case bson.DocumentMarshaler:
		return d.MarshalBSONDocument()
	default:
		var kind reflect.Kind
		if t := reflect.TypeOf(document); t.Kind() == reflect.Ptr {
			kind = t.Elem().Kind()
		}
		if reflect.ValueOf(document).Kind() == reflect.Struct || kind == reflect.Struct {
			return bson.NewDocumentEncoder().EncodeDocument(document)
		}
		return nil, fmt.Errorf("cannot transform type %s to a *bson.Document", reflect.TypeOf(document))
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
