package commandx

import (
	"errors"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/network/result"
	"go.mongodb.org/mongo-driver/x/network/wiremessage"
	"go.mongodb.org/mongo-driver/x/network/wiremessagex"
)

type IsMasterResult struct {
	Arbiters                     bsoncore.Value    `bson:"arbiters,omitempty"`
	ArbiterOnly                  bsoncore.Value    `bson:"arbiterOnly,omitempty"`
	ClusterTime                  bsoncore.Document `bson:"$clusterTime,omitempty"`
	Compression                  bsoncore.Value    `bson:"compression,omitempty"`
	ElectionID                   bsoncore.Value    `bson:"electionId,omitempty"`
	Hidden                       bsoncore.Value    `bson:"hidden,omitempty"`
	Hosts                        bsoncore.Value    `bson:"hosts,omitempty"`
	IsMaster                     bsoncore.Value    `bson:"ismaster,omitempty"`
	IsReplicaSet                 bsoncore.Value    `bson:"isreplicaset,omitempty"`
	LastWriteTimestamp           bsoncore.Value    `bson:"lastWriteDate,omitempty"`
	LogicalSessionTimeoutMinutes bsoncore.Value    `bson:"logicalSessionTimeoutMinutes,omitempty"`
	MaxBSONObjectSize            bsoncore.Value    `bson:"maxBsonObjectSize,omitempty"`
	MaxMessageSizeBytes          bsoncore.Value    `bson:"maxMessageSizeBytes,omitempty"`
	MaxWriteBatchSize            bsoncore.Value    `bson:"maxWriteBatchSize,omitempty"`
	Me                           bsoncore.Value    `bson:"me,omitempty"`
	MaxWireVersion               bsoncore.Value    `bson:"maxWireVersion,omitempty"`
	MinWireVersion               bsoncore.Value    `bson:"minWireVersion,omitempty"`
	Msg                          bsoncore.Value    `bson:"msg,omitempty"`
	OK                           bsoncore.Value    `bson:"ok"`
	Passives                     bsoncore.Value    `bson:"passives,omitempty"`
	ReadOnly                     bsoncore.Value    `bson:"readOnly,omitempty"`
	SaslSupportedMechs           bsoncore.Value    `bson:"saslSupportedMechs,omitempty"`
	Secondary                    bsoncore.Value    `bson:"secondary,omitempty"`
	SetName                      bsoncore.Value    `bson:"setName,omitempty"`
	SetVersion                   bsoncore.Value    `bson:"setVersion,omitempty"`
	Tags                         bsoncore.Document `bson:"tags,omitempty"`
}

func (imr IsMasterResult) extractResult(wm []byte) (bsoncore.Document, error) {
	wmLength := len(wm)
	length, _, _, opcode, wm, ok := wiremessagex.ReadHeader(wm)
	if !ok || int(length) > wmLength {
		return nil, errors.New("malformed wire message: insufficient bytes")
	}

	wm = wm[:wmLength-16] // constrain to just this wiremessage, incase there are multiple in the slice

	switch opcode {
	case wiremessage.OpReply:
		_, wm, ok = wiremessagex.ReadReplyFlags(wm)
		if !ok {
			return nil, errors.New("malformed wire message: missing flag bytes")
		}
		_, wm, ok = wiremessagex.ReadReplyCursorID(wm)
		if !ok {
			return nil, errors.New("malformed wire message: missing cursorID bytes")
		}
		_, wm, ok = wiremessagex.ReadReplyStartingFrom(wm)
		if !ok {
			return nil, errors.New("malformed wire message: missing startingFrom bytes")
		}
		var numReturned int32
		numReturned, wm, ok = wiremessagex.ReadReplyNumberReturned(wm)
		if !ok {
			return nil, errors.New("malformed wire message: missing numberReturned bytes")
		}
		if numReturned == 0 {
			return nil, ErrNoDocCommandResponse
		}
		if numReturned > 1 {
			return nil, ErrMultiDocCommandResponse
		}
		var res bsoncore.Document
		res, wm, ok = wiremessagex.ReadReplyDocument(wm)
		if !ok {
			return nil, NewCommandResponseError("malformed wire message: Insufficent bytes to read document", nil)
		}
		if len(wm) > 0 {
			return nil, NewCommandResponseError("malformed wire message: NumberReturned does not match number of documents returned", nil)
		}

		err := extractError(res)
		if err != nil {
			return nil, err
		}
		return res, nil
	default:
		return nil, fmt.Errorf("cannot unmarshal %v into IsMasterResult", opcode)
	}
}

func (imr IsMasterResult) ConvertToResult(wm []byte) (result.IsMaster, error) {
	doc, err := imr.extractResult(wm)
	if err != nil {
		return result.IsMaster{}, err
	}
	var res result.IsMaster
	err = bson.Unmarshal(doc, &res)
	return res, err
}

func (imr *IsMasterResult) UnmarshalWireMessage(wm []byte) error {
	wmLength := len(wm)
	length, _, _, opcode, wm, ok := wiremessagex.ReadHeader(wm)
	if !ok || int(length) > wmLength {
		return errors.New("malformed wire message: insufficient bytes")
	}

	wm = wm[:wmLength-16] // constrain to just this wiremessage, incase there are multiple in the slice

	switch opcode {
	case wiremessage.OpReply:
		_, wm, ok = wiremessagex.ReadReplyFlags(wm)
		if !ok {
			return errors.New("malformed wire message: missing flag bytes")
		}
		_, wm, ok = wiremessagex.ReadReplyCursorID(wm)
		if !ok {
			return errors.New("malformed wire message: missing cursorID bytes")
		}
		_, wm, ok = wiremessagex.ReadReplyStartingFrom(wm)
		if !ok {
			return errors.New("malformed wire message: missing startingFrom bytes")
		}
		var numReturned int32
		numReturned, wm, ok = wiremessagex.ReadReplyNumberReturned(wm)
		if !ok {
			return errors.New("malformed wire message: missing numberReturned bytes")
		}
		if numReturned == 0 {
			return ErrNoDocCommandResponse
		}
		if numReturned > 1 {
			return ErrMultiDocCommandResponse
		}
		var res bsoncore.Document
		res, wm, ok = wiremessagex.ReadReplyDocument(wm)
		if !ok {
			return NewCommandResponseError("malformed wire message: Insufficent bytes to read document", nil)
		}
		if len(wm) > 0 {
			return NewCommandResponseError("malformed wire message: NumberReturned does not match number of documents returned", nil)
		}

		err := extractError(res)
		if err != nil {
			return err
		}

		elems, err := res.Elements()
		if err != nil {
			return err
		}

		for _, elem := range elems {
			switch strings.ToLower(elem.Key()) {
			case "arbiters":
				imr.Arbiters = elem.Value()
			case "arbiteronly":
				imr.ArbiterOnly = elem.Value()
			case "$clustertime":
				imr.ClusterTime, ok = elem.Value().DocumentOK()
				if !ok {
					return errors.New("Invalid response from server, '$clusterTime' field must be a document")
				}
			case "compression":
				imr.Compression = elem.Value()
			case "electionid":
				imr.ElectionID = elem.Value()
			case "hidden":
				imr.Hidden = elem.Value()
			case "hosts":
				imr.Hosts = elem.Value()
			case "ismaster":
				imr.IsMaster = elem.Value()
			case "isreplicaset":
				imr.IsReplicaSet = elem.Value()
			case "lastwritetimestamp":
				imr.LastWriteTimestamp = elem.Value()
			case "logicalsessiontimeoutminutes":
				imr.LogicalSessionTimeoutMinutes = elem.Value()
			case "maxbsonobjectsize":
				imr.MaxBSONObjectSize = elem.Value()
			case "maxmessagesizebytes":
				imr.MaxMessageSizeBytes = elem.Value()
			case "maxwritebatchsize":
				imr.MaxWriteBatchSize = elem.Value()
			case "me":
				imr.Me = elem.Value()
			case "maxwireversion":
				imr.MaxWireVersion = elem.Value()
			case "minwireversion":
				imr.MinWireVersion = elem.Value()
			case "msg":
				imr.Msg = elem.Value()
			case "ok":
				imr.OK = elem.Value()
			case "passives":
				imr.Passives = elem.Value()
			case "readonly":
				imr.ReadOnly = elem.Value()
			case "saslsupportedmechs":
				imr.SaslSupportedMechs = elem.Value()
			case "secondary":
				imr.Secondary = elem.Value()
			case "setname":
				imr.SetName = elem.Value()
			case "setversion":
				imr.SetVersion = elem.Value()
			case "tags":
				imr.Tags, ok = elem.Value().DocumentOK()
				if !ok {
					return errors.New("Invalid response from server, 'tags' field must be a document")
				}
			}
		}
	default:
		return fmt.Errorf("cannot unmarshal %v into IsMasterResult", opcode)
	}

	return nil
}
