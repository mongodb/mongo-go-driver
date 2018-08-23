// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package command

import (
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
)

func decodeCommandOpMsg(msg wiremessage.Msg) (bson.Reader, error) {
	var mainDoc bson.Document

	for _, section := range msg.Sections {
		switch converted := section.(type) {
		case wiremessage.SectionBody:
			err := mainDoc.UnmarshalBSON(converted.Document)
			if err != nil {
				return nil, err
			}
		case wiremessage.SectionDocumentSequence:
			arr := bson.NewArray()
			for _, doc := range converted.Documents {
				newDoc := bson.NewDocument()
				err := newDoc.UnmarshalBSON(doc)
				if err != nil {
					return nil, err
				}

				arr.Append(bson.VC.Document(newDoc))
			}

			mainDoc.Append(bson.EC.Array(converted.Identifier, arr))
		}
	}

	byteArray, err := mainDoc.MarshalBSON()
	if err != nil {
		return nil, err
	}

	rdr := bson.Reader(byteArray)
	_, err = rdr.Validate()
	if err != nil {
		return nil, NewCommandResponseError("malformed OP_MSG: invalid document", err)
	}

	err = extractError(rdr)
	if err != nil {
		return nil, err
	}
	return rdr, nil
}
