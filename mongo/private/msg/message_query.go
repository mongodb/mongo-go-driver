// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package msg

import (
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson"
)

// Query is a message sent to the server.
type Query struct {
	ReqID                int32
	Flags                QueryFlags
	FullCollectionName   string
	NumberToSkip         int32
	NumberToReturn       int32
	Query                interface{}
	ReturnFieldsSelector interface{}
}

// RequestID gets the request id of the message.
func (m *Query) RequestID() int32 { return m.ReqID }

// QueryFlags are the flags in a Query.
type QueryFlags int32

// QueryFlags constants.
const (
	_ QueryFlags = 1 << iota
	TailableCursor
	SlaveOK
	OplogReplay
	NoCursorTimeout
	AwaitData
	Exhaust
	Partial
)

// AddMeta wraps the query with meta data.
func AddMeta(r Request, meta map[string]*bson.Document) error {
	if len(meta) > 0 {
		switch typedR := r.(type) {
		case *Query:
			// TODO(skriptble): To change this to the new BSON library we'll
			// need an element encoder and a value encoder. These types can be
			// used to take an interface{} and turn it into a *bson.Element or
			// *bson.Value.
			enc := bson.NewDocumentEncoder()
			query, err := enc.EncodeDocument(typedR.Query)
			if err != nil {
				panic(fmt.Errorf("Could not transform document to *bson.Document: %v", err))
			}
			doc := bson.NewDocument(
				bson.EC.SubDocument("$query", query))

			for k, v := range meta {
				doc.Append(bson.EC.SubDocument(k, v))
			}

			typedR.Query = doc
		default:
			return fmt.Errorf("cannot wrap request(%T) with meta", r)
		}
	}

	return nil
}
