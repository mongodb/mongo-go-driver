package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/mongodb/mongo-go-driver/core/address"
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
	handler := &connection.Proxy{
		Transformer: wiremessage.RequestTransformerFunc(LogRequestTransformer),
		Pool:        pool,
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
