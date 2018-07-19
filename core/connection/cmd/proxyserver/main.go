package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/address"
	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/connection"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
)

func main() {
	pool, err := connection.NewPool(address.Address("localhost:27017"), 10, 40)
	if err != nil {
		log.Fatal(err)
	}
	err = pool.Connect(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	handler := &connection.ProxyServer{
		Retriever: connection.TransformerRetrieverFunc(TransformerRetriever),
		Pool:      pool,
	}
	svr := &connection.Server{
		Addr:    address.Address("localhost:27016"),
		Handler: handler,
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		err := svr.ListenAndServe()
		if err != nil && err != connection.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-c
	err = svr.Shutdown(context.Background())
	if err != nil {
		log.Fatal(err)
	}
}

// LogRequestTransformer is a request transformer.
func LogRequestTransformer(wm wiremessage.WireMessage) (wiremessage.WireMessage, wiremessage.ResponseTransformer, error) {
	log.Println(wm)
	return wm, wiremessage.ResponseTransformerFunc(LogResponseTransformer), nil
}

// LogResponseTransformer is a response transformer.
func LogResponseTransformer(wm wiremessage.WireMessage) (wiremessage.WireMessage, error) {
	log.Println(wm)
	return wm, nil
}

// TransformerRetriever retrieves transformers.
func TransformerRetriever(wm wiremessage.WireMessage) (interface{}, error) {
	name, err := command.Name(wm)
	if err != nil {
		return nil, nil // This isn't a command, just pass it through the proxy
	}

	switch name {
	case "isProxy":
		return connection.HijackerFunc(Hijacker), nil
	}

	return wiremessage.RequestTransformerFunc(LogRequestTransformer), nil
}

// Hijacker is a connection.Hijacker.
func Hijacker(wm wiremessage.WireMessage, c connection.Connection) error {
	// We are handling commands here
	name, err := command.Name(wm)
	if err != nil {
		// Unknown command
		// TODO: We need to send the command back to the user, sketching stuff
		// out for now. Eventually, we'll need to send this back to the user.
		// This error will only happen if the wm is malformed, so we should be
		// sending a different error here anyway.
		err := command.Error{
			Code:    59,
			Message: fmt.Sprintf(`no such command: '', bad cmd: '%s'`, wm),
		}
		return err
	}

	switch name {
	case "isProxy":
		return handleIsProxy(wm, c)
	default:
		cmderr := noSuchCommand(name, wm) // Send this command back to the user
		return cmderr
	}
}

func handleIsProxy(wm wiremessage.WireMessage, c connection.Connection) error {
	log.Println("Handling 'isProxy'", wm)
	var isProxy command.IsProxy
	err := isProxy.Decode(wm).Err()
	if err != nil {
		return err
	}
	isProxyResponse := command.IsProxyResponse{Mode: "proxy server"}
	wm, err = isProxyResponse.Encode(&isProxy)
	if err != nil {
		return err
	}

	log.Println("Sending response to 'isProxy'", wm)
	return c.WriteWireMessage(context.TODO(), wm)
}

func noSuchCommand(name string, wm wiremessage.WireMessage) command.Error {
	var cmd bson.Reader
	switch t := wm.(type) {
	case wiremessage.Query:
		cmd = t.Query
	}
	cmderr := command.Error{
		Code:    59,
		Message: fmt.Sprintf(`no such command: '%s', bad cmd: '%s'`, name, cmd),
	}
	return cmderr
}
