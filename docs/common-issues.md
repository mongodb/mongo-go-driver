# Frequency Encountered Issues

## `WriteXXX` can only write while positioned on a Element or Value but is positioned on a TopLevel

The [`bson.Marshal`](https://pkg.go.dev/go.mongodb.org/mongo-driver/bson#Marshal) function requires a parameter that can be decoded into a BSON Document, i.e. a [`primitive.D`](https://github.com/mongodb/mongo-go-driver/blob/master/bson/bson.go#L31). Therefore the error message

> `WriteXXX` can only write while positioned on a Element or Value but is positioned on a TopLevel

occurs when the input to `bson.Marshal` is something *other* than a BSON Document. Examples of this occurance include

- `WriteString`: the input into `bson.Marshal` is a string
- `WriteNull`: the input into `bson.Marshal` is null
- `WriteInt32`: the input into `bson.Marshal` is an integer

Here is a more concrete example which is fixed by initialize `sort` via `sort := bson.D{}`:

```go
package main

import (
	"context"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	ctx := context.Background()
	uri := "mongodb://localhost:27017"
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	coll := client.Database("foo").Collection("bar")
	var sort bson.D // this is nil and will result in a WriteNull error
	update := bson.D{{"$inc", bson.D{{"x", 1}}}}
	sr := coll.FindOneAndUpdate(ctx, bson.D{}, update, options.FindOneAndUpdate().SetSort(sort))

	if err := sr.Err(); err != nil {
		log.Fatalf("error getting single result: %v", err)
	}
}
```

