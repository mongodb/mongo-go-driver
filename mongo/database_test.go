package mongo

import (
	"context"
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
)

// Retrieve a database from a client by using the Database method.
func ExampleDatabase(client *Client) error {
	db := client.Database("foobarbaz")
}

// Individual commands can be sent to the server and response retrieved via run command.
func ExampleDatabase_RunCommand(db *Database) (bson.Raw, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := db.RunCommand(ctx, bson.D{{"ping", 1}})
	if err != nil {
		return nil, err
	}
	return result, nil
}
