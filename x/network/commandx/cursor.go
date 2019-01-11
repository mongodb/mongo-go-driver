package commandx

import (
	"errors"
	"fmt"
	"strconv"

	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/network/wiremessage"
	"go.mongodb.org/mongo-driver/x/network/wiremessagex"
)

type Cursor struct {
	// Cursor Response parameters
	FirstBatch bsoncore.Value
	NextBatch  bsoncore.Value
	NS         bsoncore.Value
	ID         bsoncore.Value

	OperationTime bsoncore.Value
	ClusterTime   bsoncore.Document `bson:"$clusterTime"`
}

func (c *Cursor) UnmarshalWireMessage(wm []byte) error {
	wmLength := len(wm)
	length, _, _, opcode, wm, ok := wiremessagex.ReadHeader(wm)
	if !ok || int(length) > wmLength {
		return errors.New("malformed wire message: insufficient bytes")
	}

	wm = wm[:wmLength-16] // constrain to just this wiremessage, incase there are multiple in the slice

	switch opcode {
	case wiremessage.OpMsg:
		_, wm, ok = wiremessagex.ReadMsgFlags(wm)
		if !ok {
			return errors.New("malformed wire message: missing OP_MSG flags")
		}

		var fbSet, nbSet, nsSet, idSet, otSet, ctSet bool
		var singleDocument bsoncore.Document
		for len(wm) > 0 {
			var stype wiremessage.SectionType
			stype, wm, ok = wiremessagex.ReadMsgSectionType(wm)
			if !ok {
				return errors.New("malformed wire message: insuffienct bytes to read section type")
			}

			switch stype {
			case wiremessage.SingleDocument:
				singleDocument, wm, ok = wiremessagex.ReadMsgSectionSingleDocument(wm)
				if !ok {
					return errors.New("malformed wire message: insufficient bytes to read single document")
				}

				elements, err := singleDocument.Elements()
				if err != nil {
					return err
				}

				for _, elem := range elements {
					// TODO(GODRIVER-617): We can use arrays and slices of bytes here to reduce string
					// allocations.
					switch elem.Key() {
					case "cursor":
						cursorElems, err := elem.Value().Document().Elements()
						if err != nil {
							return err
						}

						for _, cursorElem := range cursorElems {
							switch cursorElem.Key() {
							case "firstBatch":
								if fbSet {
									return errors.New("Invalid response from server, multiple 'firstBatch' fields")
								}
								fbSet = true
								c.FirstBatch = cursorElem.Value()
							case "nextBatch":
								if nbSet {
									return errors.New("Invalid response from server, multiple 'nextBatch' fields")
								}
								nbSet = true
								c.NextBatch = cursorElem.Value()
							case "ns":
								if nsSet {
									return errors.New("Invalid response from server, multiple 'ns' fields")
								}
								nsSet = true
								c.NS = cursorElem.Value()
							case "id":
								if idSet {
									return errors.New("Invalid response from server, multiple 'id' fields")
								}
								idSet = true
								c.ID = cursorElem.Value()
							}
						}
					case "operationTime":
						if otSet {
							return errors.New("Invalid response from server, multiple 'operationTime' fields")
						}
						otSet = true
						c.OperationTime = elem.Value()
					case "$clusterTime":
						if ctSet {
							return errors.New("Invalid response from server, multiple '$clusterTime' fields")
						}
						ctSet = true
						c.ClusterTime, ok = elem.Value().DocumentOK()
						if !ok {
							return errors.New("Invalid response from server, '$clusterTime' field must be a document")
						}
					}
				}
			case wiremessage.DocumentSequence:
				var identifier string
				var docs []bsoncore.Document
				identifier, docs, wm, ok = wiremessagex.ReadMsgSectionDocumentSequence(wm)
				if !ok {
					return errors.New("malformed wire message: insufficient bytes to read document sequence")
				}

				switch identifier {
				case "firstBatch":
					if fbSet {
						return errors.New("Invalid response from server, multiple 'firstBatch' fields")
					}
					fbSet = true
					idx, doc := bsoncore.AppendArrayStart(nil)
					for i, doc := range docs {
						doc = bsoncore.AppendDocumentElement(doc, strconv.Itoa(i), doc)
					}
					doc, _ = bsoncore.AppendArrayEnd(doc, idx)
					c.FirstBatch = bsoncore.Value{Type: bsontype.Array, Data: doc}
				case "nextBatch":
					if nbSet {
						return errors.New("Invalid response from server, multiple 'nextBatch' fields")
					}
					nbSet = true
					idx, doc := bsoncore.AppendArrayStart(nil)
					for i, doc := range docs {
						doc = bsoncore.AppendDocumentElement(doc, strconv.Itoa(i), doc)
					}
					doc, _ = bsoncore.AppendArrayEnd(doc, idx)
					c.FirstBatch = bsoncore.Value{Type: bsontype.Array, Data: doc}
				}
			default:
				return fmt.Errorf("malformed wire message: uknown section type %v", stype)
			}
		}
		return nil
	default:
		return fmt.Errorf("cannot unmarshal %v into Cursor", opcode)
	}
}
