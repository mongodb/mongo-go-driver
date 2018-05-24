package main

import (
	"context"
	"fmt"

	"github.com/mongodb/mongo-go-driver/options-design/mongo"
	"github.com/mongodb/mongo-go-driver/options-design/mongo/findopts"
)

func main() {
	fmt.Println("options examples")
}

func find(ctx context.Context, filter interface{}, collection *mongo.Collection) error {
	var err error

	// The basic use case, no options
	_, err = collection.Find(ctx, filter)

	// A bundle can be created so that we can use it to discover which options are valid.
	_, err = collection.Find(ctx, filter, findopts.BundleFind().AllowPartialResults(true))

	// If we know what options are valid for which methods, we can just use the functions directly.
	_, err = collection.Find(ctx, filter, findopts.AllowPartialResults(true))

	// an empty bundle can be used as a namespace, the nil value is useful.
	var bundle *findopts.FindBundle

	// We can use it without making an actual instance.
	_, err = collection.Find(ctx, filter, bundle.BatchSize(5))

	bundle = findopts.BundleFind(findopts.AllowPartialResults(true))
	_, err = collection.Find(ctx, filter, bundle)

	bundle = findopts.BundleFind(findopts.AllowPartialResults(true)).Limit(10)
	_, err = collection.Find(ctx, filter, bundle, findopts.Limit(5))

	bundle = bundle.Skip(10)
	_, err = collection.Find(ctx, filter, bundle.Sort(map[string]interface{}{"foo": -1}))
	return err
}
