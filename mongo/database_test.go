package mongo

import (
	"context"
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
)

// Individual commands can be sent to the server and response retrieved via run command.
func ExampleDatabase_RunCommand() {
	var db *Database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := db.RunCommand(ctx, bson.D{{"ping", 1}}).Err()
	if err != nil {
		return
	}
	return
}
