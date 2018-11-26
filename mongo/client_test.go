package mongo

import (
	"context"
	"time"
)

func ExampleClient_Connect() error {
	client, err := NewClient("mongodb://foo:bar@localhost:27017")
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		return err
	}

	return nil
}
