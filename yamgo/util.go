package yamgo

import (
	"github.com/10gen/mongo-go-driver/bson"
)

func getOrInsertID(document interface{}) (bson.M, interface{}, error) {
	bytes, err := bson.Marshal(document)
	if err != nil {
		return nil, nil, err
	}

	// TODO GODRIVER-77: Roundtrip is inefficient, and order of user-provided document isn't preserved.
	var doc bson.M
	err = bson.Unmarshal(bytes, &doc)
	if err != nil {
		return nil, nil, err
	}

	var id interface{}
	if docID, ok := doc["_id"]; ok {
		id = docID
	} else {
		id = bson.NewObjectId()
		doc["_id"] = id
	}

	return doc, id, nil
}
