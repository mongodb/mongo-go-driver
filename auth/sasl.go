package auth

import (
	"gopkg.in/mgo.v2/bson"

	"github.com/10gen/mongo-go-driver/core"
	"github.com/10gen/mongo-go-driver/core/msg"
)

type saslClient interface {
	Start() (string, []byte, error)
	Next(challenge []byte) ([]byte, error)
}

func conductSaslConversation(conn core.Connection, db string, client saslClient) error {
	if db == "" {
		db = defaultAuthDB
	}

	mech, payload, err := client.Start()
	if err != nil {
		return newError(err, mech)
	}

	saslStartRequest := msg.NewCommand(
		msg.NextRequestID(),
		db,
		true,
		bson.D{
			{"saslStart", 1},
			{"mechanism", mech},
			{"payload", payload},
		},
	)

	type saslResponse struct {
		ConversationID int    `bson:"conversationId"`
		Code           int    `bson:"code"`
		Done           bool   `bson:"done"`
		Payload        []byte `bson:"payload"`
	}

	var saslResp saslResponse
	err = core.ExecuteCommand(conn, saslStartRequest, &saslResp)
	if err != nil {
		return newError(err, mech)
	}

	cid := saslResp.ConversationID

	for {
		if saslResp.Code != 0 {
			return newError(err, mech)
		}
		if saslResp.Done {
			return nil
		}

		payload, err = client.Next(saslResp.Payload)
		if err != nil {
			return newError(err, mech)
		}

		saslContinueRequest := msg.NewCommand(
			msg.NextRequestID(),
			db,
			true,
			bson.D{
				{"saslContinue", 1},
				{"conversationId", cid},
				{"payload", payload},
			},
		)

		err = core.ExecuteCommand(conn, saslContinueRequest, &saslResp)
		if err != nil {
			return newError(err, mech)
		}
	}
}
