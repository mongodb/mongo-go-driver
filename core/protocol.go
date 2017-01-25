package core

import (
	"fmt"

	"github.com/craiggwilson/mongo-go-driver/core/msg"

	"gopkg.in/mgo.v2/bson"
)

// ExecuteCommand executes the message on the channel.
func ExecuteCommand(c Connection, request msg.Request, out interface{}) error {
	return ExecuteCommands(c, []msg.Request{request}, []interface{}{out})
}

// ExecuteCommands executes the messages on the connection.
func ExecuteCommands(c Connection, requests []msg.Request, out []interface{}) error {
	if len(requests) != len(out) {
		return fmt.Errorf("invalid arguments. 'out' length must equal 'msgs' length")
	}

	err := c.Write(requests...)
	if err != nil {
		return fmt.Errorf("failed sending commands(%d): %v", len(requests), err)
	}

	for i, req := range requests {
		resp, err := c.Read()
		if err != nil {
			return fmt.Errorf("failed receiving command response for %d: %v", req.RequestID(), err)
		}

		if resp.ResponseTo() != req.RequestID() {
			return fmt.Errorf("received out of order response. expected %d but got %d", req.RequestID(), resp.ResponseTo())
		}

		err = readCommandResponse(resp, out[i])
		if err != nil {
			return fmt.Errorf("failed reading command response for %d: %v", req.RequestID(), err)
		}
	}

	return nil
}

func readCommandResponse(resp msg.Response, out interface{}) error {
	switch typedResp := resp.(type) {
	case *msg.Reply:
		if typedResp.NumberReturned == 0 {
			return fmt.Errorf("command returned no documents")
		}
		if typedResp.NumberReturned > 1 {
			return fmt.Errorf("command returned multiple documents")
		}

		if typedResp.ResponseFlags&msg.QueryFailure != 0 {
			// read first document as error
			var doc bson.D
			ok, err := typedResp.Iter().One(&doc)
			if err != nil {
				return fmt.Errorf("failed to read command failure document: %v", err)
			}
			if !ok {
				return fmt.Errorf("unknown command failure")
			}

			return fmt.Errorf("command failure: %v", doc)
		}

		ok, err := typedResp.Iter().One(out)
		if err != nil {
			return fmt.Errorf("failed to read command response document: %v", err)
		}
		if !ok {
			return fmt.Errorf("no command response document")
		}
	default:
		return fmt.Errorf("unsupported response message type: %T", typedResp)
	}

	return nil
}
