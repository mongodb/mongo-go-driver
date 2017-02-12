package conn

import (
	"fmt"

	"github.com/10gen/mongo-go-driver/internal"
	"github.com/10gen/mongo-go-driver/msg"

	"gopkg.in/mgo.v2/bson"
)

// QueryFailureError is an error with a failure response as a document.
type QueryFailureError struct {
	Msg      string
	Response bson.D
}

func (e *QueryFailureError) Error() string {
	return fmt.Sprintf("%s: %v", e.Msg, e.Response)
}

// Message retrieves the message of the error.
func (e *QueryFailureError) Message() string {
	return e.Msg
}

// ExecuteCommand executes the message on the channel.
func ExecuteCommand(c Connection, request msg.Request, out interface{}) error {
	return ExecuteCommands(c, []msg.Request{request}, []interface{}{out})
}

// ExecuteCommands executes the messages on the connection.
func ExecuteCommands(c Connection, requests []msg.Request, out []interface{}) error {
	if len(requests) != len(out) {
		panic("invalid arguments. 'out' length must equal 'msgs' length")
	}

	err := c.Write(requests...)
	if err != nil {
		return internal.WrapErrorf(err, "failed sending commands(%d)", len(requests))
	}

	var errors []error
	for i, req := range requests {
		resp, err := c.Read()
		if err != nil {
			return internal.WrapErrorf(err, "failed receiving command response for %d", req.RequestID())
		}

		if resp.ResponseTo() != req.RequestID() {
			errors = append(errors, fmt.Errorf("received out of order response: expected %d but got %d", req.RequestID(), resp.ResponseTo()))
			continue
		}

		err = readCommandResponse(resp, out[i])
		if err != nil {
			errors = append(errors, internal.WrapErrorf(err, "failed reading command response for %d", req.RequestID()))
			continue
		}
	}

	switch len(errors) {
	case 0:
	case 1:
		return errors[0]
	default:
		return &multiError{
			message: "multiple errors occured",
			errors:  errors,
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
				return internal.WrapError(err, "failed to read command failure document")
			}
			if !ok {
				return fmt.Errorf("unknown command failure")
			}

			return &QueryFailureError{
				Msg:      "command failure",
				Response: doc,
			}
		}

		// read into raw first
		var raw bson.RawD
		ok, err := typedResp.Iter().One(&raw)
		if err != nil {
			return fmt.Errorf("failed to read command response document: %v", err)
		}
		if !ok {
			return fmt.Errorf("no command response document")
		}

		// check the raw command response for ok field.
		ok = false
		for _, rawElem := range raw {
			if rawElem.Name == "ok" {
				var v int32
				err := rawElem.Value.Unmarshal(&v)
				if err == nil && v == 1 {
					ok = true
					break
				}
			}
		}
		if !ok {
			var errmsg string
			for _, rawElem := range raw {
				if rawElem.Name == "errmsg" {
					rawElem.Value.Unmarshal(&errmsg)
					break
				}
			}
			if errmsg == "" {
				return fmt.Errorf("command failed")
			}
			return fmt.Errorf("command failed: %s", errmsg)
		}

		// re-decode the response into the user provided structure...
		ok, err = typedResp.Iter().One(out)
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
