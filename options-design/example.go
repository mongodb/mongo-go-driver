package main

import (
	"context"
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/skriptble/giant/findoptsx"
	"github.com/skriptble/giant/mongox"
)

func main() {
	fmt.Println("options examples")
}

func find(ctx context.Context, filter interface{}, collection *mongox.Collection) error {
	var err error

	// The basic use case, no options
	_, err = collection.Find(ctx, filter)

	// A bundle can be created so that we can use it to discover which options are valid.
	_, err = collection.Find(ctx, filter, findoptsx.BundleFind().AllowPartialResults(true))

	// If we know what options are valid for which methods, we can just use the functions directly.
	_, err = collection.Find(ctx, filter, findoptsx.AllowPartialResults(true))

	// an empty bundle can be used as a namespace, the nil value is useful.
	var bundle *findoptsx.FindBundle

	// We can use it without making an actual instance.
	_, err = collection.Find(ctx, filter, bundle.BatchSize(5))

	bundle = findoptsx.BundleFind(findoptsx.AllowPartialResults(true))
	_, err = collection.Find(ctx, filter, bundle)

	bundle = findoptsx.BundleFind(findoptsx.AllowPartialResults(true)).Limit(10)
	_, err = collection.Find(ctx, filter, bundle, findoptsx.Limit(5))

	bundle = bundle.Skip(10)
	_, err = collection.Find(ctx, filter, bundle.Sort(bson.NewDocument(bson.EC.Int32("foo", -1))))
	return err
}