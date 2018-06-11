package main

import (
	"context"
	"fmt"

	"github.com/mongodb/mongo-go-driver/options-design/mongo"
	"github.com/mongodb/mongo-go-driver/options-design/mongo/findopt"
	"github.com/mongodb/mongo-go-driver/options-design/mongo/countopt"
	"github.com/mongodb/mongo-go-driver/options-design/mongo/distinctopt"
	"github.com/mongodb/mongo-go-driver/options-design/mongo/mongoopt"
	"github.com/mongodb/mongo-go-driver/options-design/mongo/aggregateopt"
	"github.com/mongodb/mongo-go-driver/options-design/mongo/insertopt"
)

func main() {
	fmt.Println("options examples")
}

func find(ctx context.Context, filter interface{}, collection *mongo.Collection) error {
	var err error

	// The basic use case, no options
	_, err = collection.Find(ctx, filter)

	// A bundle can be created so that we can use it to discover which options are valid.
	_, err = collection.Find(ctx, filter, findopt.BundleFind().AllowPartialResults(true))

	// If we know what options are valid for which methods, we can just use the functions directly.
	_, err = collection.Find(ctx, filter, findopt.AllowPartialResults(true))

	// an empty bundle can be used as a namespace, the nil value is useful.
	var bundle *findopt.FindBundle

	// We can use it without making an actual instance.
	_, err = collection.Find(ctx, filter, bundle.BatchSize(5))

	bundle = findopt.BundleFind(findopt.AllowPartialResults(true))
	_, err = collection.Find(ctx, filter, bundle)

	bundle = findopt.BundleFind(findopt.AllowPartialResults(true)).Limit(10)
	_, err = collection.Find(ctx, filter, bundle, findopt.Limit(5))

	bundle = bundle.Skip(10)
	_, err = collection.Find(ctx, filter, bundle.Sort(map[string]interface{}{"foo": -1}))
	return err
}

func count(ctx context.Context, filter interface{}, collection *mongo.Collection) error {
	var err error

	// no options
	_, err = collection.Count(ctx, filter)

	// bundle
	_, err = collection.Count(ctx, filter, countopt.BundleCount().Limit(5))

	// use without bundle
	_, err = collection.Count(ctx, filter, countopt.Limit(5))

	// empty bundle
	var bundle *countopt.CountBundle

	// use without instance
	_, err = collection.Count(ctx, filter, bundle.Limit(5))

	_, err = collection.Count(ctx, filter, bundle)

	bundle = countopt.BundleCount(countopt.Limit(10)).Skip(20)
	_, err = collection.Count(ctx, filter, bundle, countopt.OptMaxTimeMs(50))

	return err
}

func distinct(ctx context.Context, fieldName string, filter interface{}, collection *mongo.Collection) error {
	var err error

	_, err = collection.Distinct(ctx, fieldName, filter)

	_, err = collection.Distinct(ctx, fieldName, filter, distinctopt.BundleDistinct().Collation(&mongoopt.Collation{}))

	_, err = collection.Distinct(ctx, fieldName, filter, distinctopt.Collation(&mongoopt.Collation{}))

	var bundle *distinctopt.DistinctBundle

	_, err = collection.Distinct(ctx, fieldName, filter, bundle.Collation(&mongoopt.Collation{}))

	_, err = collection.Distinct(ctx, fieldName, filter, bundle)

	bundle = distinctopt.BundleDistinct(distinctopt.Collation(&mongoopt.Collation{}))

	_, err = collection.Distinct(ctx, fieldName, filter, bundle, distinctopt.OptMaxTime(50))

	return err
}

func aggregate(ctx context.Context, filter interface{}, collection *mongo.Collection) error {
	var err error

	// The basic use case, no options
	_, err = collection.Aggregate(ctx, filter)

	// A bundle can be created so that we can use it to discover which options are valid.
	_, err = collection.Aggregate(ctx, filter, aggregateopt.BundleAggregate().AllowDiskUse(true))

	// If we know what options are valid for which methods, we can just use the functions directly.
	_, err = collection.Aggregate(ctx, filter, aggregateopt.AllowDiskUse(true))

	// an empty bundle can be used as a namespace, the nil value is useful.
	var bundle *aggregateopt.AggregateBundle

	// We can use it without making an actual instance.
	_, err = collection.Aggregate(ctx, filter, bundle.BatchSize(5))

	bundle = aggregateopt.BundleAggregate(aggregateopt.AllowDiskUse(true))
	_, err = collection.Aggregate(ctx, filter, bundle)

	bundle = aggregateopt.BundleAggregate(aggregateopt.AllowDiskUse(true)).BatchSize(10)
	_, err = collection.Aggregate(ctx, filter, bundle, aggregateopt.BatchSize(5))

	bundle = bundle.MaxTime(10)
	_, err = collection.Aggregate(ctx, filter, bundle.Comment("foo bar"))
	return err
}

func insert(ctx context.Context, doc interface{}, collection *mongo.Collection) error {
	var err error

	_, err = collection.InsertOne(ctx, doc)

	_, err = collection.InsertOne(ctx, doc, insertopt.BundleOne().BypassDocumentValidation(true))

	_, err = collection.InsertOne(ctx, doc, insertopt.BypassDocumentValidation(true))

	var bundle *insertopt.OneBundle

	_, err = collection.InsertOne(ctx, doc, bundle.BypassDocumentValidation(true))

	return err
}
