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

	ok := false
	var errmsg, codeName string
	var code int32
	itr, err := rdr.Iterator()
	if err != nil {
		return nil, NewCommandResponseError("malformed OP_MSG: cannot iterate document", err)
	}

	for itr.Next() {
		elem := itr.Element()
		switch elem.Key() {
		case "ok":
			switch elem.Value().Type() {
			case bson.TypeInt32:
				if elem.Value().Int32() == 1 {
					ok = true
				}
			case bson.TypeInt64:
				if elem.Value().Int64() == 1 {
					ok = true
				}
			case bson.TypeDouble:
				if elem.Value().Double() == 1 {
					ok = true
				}
			}
		case "errmsg":
			if str, okay := elem.Value().StringValueOK(); okay {
				errmsg = str
			}
		case "codeName":
			if str, okay := elem.Value().StringValueOK(); okay {
				codeName = str
			}
		case "code":
			if c, okay := elem.Value().Int32OK(); okay {
				code = c
			}
		}
	}

	if !ok {
		if errmsg == "" {
			errmsg = "command failed"
		}

		return nil, Error{
			Code:    code,
			Message: errmsg,
			Name:    codeName,
		}
	}

	return rdr, nil
}
