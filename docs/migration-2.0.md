# Migrating from 1.x to 2.0

The minimum supported version of Go for v2 is 1.18.

To upgrade imports of the Go Driver from v1 to v2, we recommend using [marwan-at-work/mod
](https://github.com/marwan-at-work/mod):

```
mod upgrade --mod-name=go.mongodb.org/mongo-driver
```

## Description Package

The `description` package has been removed in v2.

## Event Package

References to `description.Server` and `description.Topology` have been replaced with `event.ServerDescription` and `event.TopologyDescription`, respectively. Additionally, the following changes have been made to the fields:

- `Kind` has been changed from `uint32` to `string` for ease of use.
- `SessionTimeoutMinutes` has been changed from `uint32` to `*int64` to differentiate between a zero timeout and no timeout.

The following event constants have been renamed to match their string literal value:

- `PoolCreated` to `ConnectionPoolCreated`
- `PoolReady` to `ConnectionPoolRead`
- `PoolCleared` to `ConnectionPoolCleared`
- `PoolClosedEvent` to `ConnectionPoolClosed`
- `GetStarted` to `ConnectionCheckOutStarted`
- `GetFailed` to `ConnectionCheckOutFailed`
- `GetSucceeded` to `ConnectionCheckedOut`
- `ConnectionReturned` to `ConnectionCheckedIn`

### CommandFailedEvent

`CommandFailedEvent.Failure` has been converted from a string type to an error type to convey additional error information and to match the type of other `*.Failure` fields, like `ServerHeartbeatFailedEvent.Failure`.

### CommandFinishedEvent

The type for `ServerConnectionID` has been changed to `*int64` to prevent an `int32` overflow and to differentiate between an ID of `0` and no ID.

### CommandStartedEvent

The type for `ServerConnectionID` has been changed to `*int64` to prevent an `int32` overflow and to differentiate between an ID of `0` and no ID.

### ServerHeartbeatFailedEvent

`DurationNanos` has been removed in favor of `Duration`.

### ServerHeartbeatSucceededEvent

`DurationNanos` has been removed in favor of `Duration`.

## Mongo Package

### Client

#### Connect

`Client.Connect` has been removed in favor of `mongo.Connect()`. See section on `NewClient` for more details.

The `context.Context` parameter has been removed from `mongo.Connect()` since [deployment connector ](https://github.com/mongodb/mongo-go-driver/blob/a76687682f080c9612295b646a00650d00dd16e1/x/mongo/driver/driver.go#L91) doesn’t accept a context, so the context passed to `mongo.Connect` doesn’t actually do anything.

#### UseSession\[WithOptions\]

This example shows how to construct a session object from a context, rather than using a context to perform session operations.

```go
// v1

client.UseSession(context.TODO(), func(sctx mongo.SessionContext) error {
  if err := sctx.StartTransaction(options.Transaction()); err != nil {
    return err
  }

  _, err = coll.InsertOne(context.TODO(), bson.D{{"x", 1}})

  return err
})
```

```go
// v2

client.UseSession(context.TODO(), func(ctx context.Context) error {
  sess := mongo.SessionFromContext(ctx)

  if err := sess.StartTransaction(options.Transaction()); err != nil {
    return err
  }

  _, err = coll.InsertOne(context.TODO(), bson.D{{"x", 1}})

  return err
})
```

### Collection

#### Clone

This example shows how to migrate usage of the `collection.Clone` method, which no longer returns an error.

```go
// v1

clonedColl, err := coll.Clone(options.Collection())
if err != nil {
  log.Fatalf("failed to clone collection: %v", err)
}
```

```go
// v2

clonedColl := coll.Clone(options.Collection())
```

#### Distinct

As of v2, the Distinct collection method will return a struct that can be decoded, similar to `Collection.FindOne`. Instead of iterating through an untyped slice, users can decode same-type data using conventional Go syntax.

If the data returned is not same-type (i.e. `name` is not always a string) a user can iterate through the result directly as a `bson.RawArray` type:

```go
// v1

filter := bson.D{{"age", bson.D{{"$gt", 25}}}}

values, err := coll.Distinct(context.TODO(), "name", filter)
if err != nil {
  log.Fatalf("failed to get distinct values: %v", err)
}

people := make([]any, 0, len(values))
for _, value := range values {
  people = append(people, value)
}

fmt.Printf("car-renting persons: %v\n", people)
```

```go
// v2

filter := bson.D{{"age", bson.D{{"$gt", 25}}}}

res := coll.Distinct(context.TODO(), "name", filter)
if err := res.Err(); err != nil {
  log.Fatalf("failed to get distinct result: %v", err)
}

var people []string
if err := res.Decode(&people); err != nil {
  log.Fatal("failed to decode distint result: %v", err)
}

fmt.Printf("car-renting persons: %v\n", people)
```

If the data returned is not same-type (i.e. `name` is not always a string) a user can iterate through the result directly as a `bson.RawArray` type:

```go
// v2

filter := bson.D{{"age", bson.D{{"$gt", 25}}}}
distinctOpts := options.Distinct().SetMaxTime(2 * time.Second)

res := coll.Distinct(context.TODO(), "name", filter, distinctOpts)
if err := res.Err(); err != nil {
  log.Fatalf("failed to get distinct result: %v", err)
}

rawArr, err := res.Raw()
if err != nil {
  log.Fatalf("failed to get raw data: %v", err)
}

values, err := rawArr.Values()
if err != nil {
  log.Fatalf("failed to get values: %v", err)
}

people := make([]string, 0, len(rawArr))
for _, value := range values {
  people = append(people, value.String())
}

fmt.Printf("car-renting persons: %v\n", people)
```

### InsertMany

The `documents` parameter in the `Collection.InsertMany` function signature has been changed from an `[]interface{}` type to an `any` type. This API no longer requires users to copy existing slice of documents to an `[]interface{}` slice.

```go
// v1

books := []book{
	{
		Name:   "Don Quixote de la Mancha",
		Author: "Miguel de Cervantes",
	},
	{
		Name:   "Cien años de soledad",
		Author: "Gabriel García Márquez",
	},
	{
		Name:   "Crónica de una muerte anunciada",
		Author: "Gabriel García Márquez",
	},
}

booksi := make([]interface{}, len(books))
for i, book := range books {
	booksi[i] = book
}

_, err = collection.InsertMany(ctx, booksi)
if err != nil {
	log.Fatalf("could not insert Spanish authors: %v", err)
}
```

```go
// v2

books := []book{
	{
		Name:   "Don Quixote de la Mancha",
		Author: "Miguel de Cervantes",
	},
	{
		Name:   "Cien años de soledad",
		Author: "Gabriel García Márquez",
	},
	{
		Name:   "Crónica de una muerte anunciada",
		Author: "Gabriel García Márquez",
	},
}


_, err = collection.InsertMany(ctx, books)
if err != nil {
	log.Fatalf("could not insert Spanish authors: %v", err)
}
```

### Database

#### ListCollectionSpecifications

Updated to return a slice of structs instead of a slice of pointers.

```go
// v1

var specs []*mongo.CollectionSpecification
specs, _ = db.ListCollectionSpecifications(context.TODO(), bson.D{})
```

```go
// v2

var specs []mongo.CollectionSpecification
specs, _ = db.ListCollectionSpecifications(context.TODO(), bson.D{})
```

### ErrUnacknowledgedWrite

This sentinel error has been removed from the `mongo` package. Users that need to check if a write operation was unacknowledged can do so by inspecting the `Acknowledged` field on the associated struct:

- `BulkWriteResult`
- `DeleteResult`
- `InsertManyResult`
- `InsertOneResult`
- `RewrapManyDataKeyResult`
- `SingleResult`

```go
// v1

res, err := coll.InsertMany(context.TODO(), books)
if errors.Is(err, mongo.ErrUnacknowledgedWrite) {
	// Do something
}
```

```go
// v2

res, err := coll.InsertMany(context.TODO(), books)
if !res.Acknowledged {
	// Do something
}
```

DDL commands such as dropping a collection will no longer return `ErrUnacknowledgedWrite`, nor will they return a result type that can be used to determine acknowledgement. It is recommended not to perform DDL commands with an unacknowledged write concern.

### Cursor

`Cursor.SetMaxTime` has been renamed to `Cursor.SetMaxAwaitTime`, specifying the maximum time for the server to wait for new documents that match the tailable cursor query on a capped collection.

### GridFS

The `gridfs` package has been merged into the `mongo` package. Additionally, `gridfs.Bucket` has been renamed to `mongo.GridFSBucket`

```go
// v1

var bucket gridfs.Bucket
bucket, _ = gridfs.NewBucket(db, opts)
```

```go
// v2

var bucket mongo.GridFSBucket
bucket, _ = db.GridFSBucket(opts)
```

#### ErrWrongIndex

`ErrWrongIndex` has been renamed to the more intuitive `ErrMissingChunk`, which indicates that the number of chunks read from the server is less than expected.

```go
// v1

n, err := source.Read(buf)
if errors.Is(err, gridfs.ErrWrongIndex) {
	// Do something
}
```

```go
// v2

n, err := source.Read(buf)
if errors.Is(err, mongo.ErrMissingChunk) {
	// Do something
}
```

#### SetWriteDeadline

`SetWriteDeadline` methods have been removed from GridFS operations in favor of extending bucket methods to include a `context.Context` argument.

```go
// v1

uploadStream, _ := bucket.OpenUploadStream("filename", uploadOpts)
uploadStream.SetWriteDeadline(time.Now().Add(2*time.Second))
```

```go
// v2

ctx, cancel := context.WithTimeout(context.TODO(), 2*time.Second)
defer cancel()

uploadStream, _ := bucket.OpenUploadStream(ctx, "filename", uploadOpts)
```

Additionally, `Bucket.DeleteContext()`, `Bucket.FindContext()`, `Bucket.DropContext()`, and `Bucket.RenameContext()` have been removed.

### IndexOptionsBuilder

`mongo.IndexOptionsBuilder` has been removed, use the `IndexOptions` type in the `options` package instead.

### IndexView

#### DropAll

Dropping an index replies with a superset of the following message: `{nIndexesWas: n}`, where n indicates the number of indexes there were prior to removing whichever index(es) were dropped. In the case of DropAll this number is always `m - 1`, where m is the total number of indexes. Thus, we can simplify the DropAll method by removing the server response.

```go
// v1

res, err := coll.Indexes().DropAll(context.TODO())
if err != nil {
	log.Fatalf("failed to drop indexes: %v", err)
}

type dropResult struct {
	NIndexesWas int
}

dres := dropResult{}
if err := bson.Unmarshal(res, &dres); err != nil {
	log.Fatalf("failed to decode: %v", err)
}

numDropped := dres.NIndexWas

// Use numDropped
```

```go
// v2

// List the indexes
cur, err := coll.Indexes().List(context.TODO())
if err != nil {
	log.Fatalf("failed to list indexes: %v", err)
}

numDropped := 0
for cur.Next(context.TODO()) {
numDropped++
}

if err := coll.Indexes().DropAll(context.TODO()); err != nil {
	log.Fatalf("failed to drop indexes: %v", err)
}

// List the indexes
cur, err := coll.Indexes().List(context.TODO())
if err != nil {
	log.Fatalf("failed to list indexes: %v", err)
}

// Use numDropped
```

#### DropOne

Dropping an index replies with a superset of the following message: `{nIndexesWas: n}`, where n indicates the number of indexes there were prior to removing whichever index(es) were dropped. In the case of `DropOne` this number is always 1 in non-error cases. We can simplify the `DropOne` method by removing the server response.

#### ListSpecifications

Updated to return a slice of structs instead of a slice of pointers. See the [database analogue](#ListCollectionSpecifications) for migration guide.

### NewClient

`NewClient` has been removed, use the `Connect` function in the `mongo` package instead.

```go
client, err := mongo.NewClient(options.Client())
if err != nil {
	log.Fatalf("failed to create client: %v", err)
}

if err := client.Connect(context.TODO()); err != nil {
	log.Fatalf("failed to connect to server: %v", err)
}
```

```go
client, err := mongo.Connect(options.Client())
if err != nil {
	log.Fatalf("failed to connect to server: %v", err)
}
```

### Session

Uses of `mongo.Session` through driver constructors (such as `client.StartSession`) have been changed to return a pointer to a `mongo.Session` struct and will need to be updated accordingly.

```go
// v1

var sessions []mongo.Session
for i := 0; i < numSessions; i++ {
  sess, _ := client.StartSession()
  sessions = append(sessions, sess)
}
```

```go
// v2

var sessions []*mongo.Session
for i := 0; i < numSessions; i++ {
  sess, _ := client.StartSession()
  sessions = append(sessions, sess)
}
```

### SingleResult

`SingleResult.DecodeBytes` has been renamed to the more intuitive `SingleResult.Raw`.

### WithSession

This example shows how to update the callback for `mongo.WithSession` to use a `context.Context` implementation, rather than the custom `mongo.SessionContext`.

```go
// v1

mongo.WithSession(context.TODO(),sess,func(sctx mongo.SessionContext) error {
  // Callback
  return nil
})
```

```go
// v2

mongo.WithSession(context.TODO(),sess,func(ctx context.Context) error {
  // Callback
  return nil
})
```

## Options Package

The following fields were marked for internal use only and do not have replacement:

- `ClientOptions.AuthenticateToAnything`
- `FindOptions.OplogReplay`
- `FindOneOptions.OplogReplay`

The following fields were removed because they are no longer supported by the server

- `FindOptions.Snapshot` (4.0)
- `FindOneOptions.Snapshot` (4.0)
- `IndexOptions.Background` (4.2)

### Options

The Go driver offers users the ability to pass multiple options objects to operations in a last-on-wins algorithm, merging data at a field level:

```psuedo
function MergeOptions(target, optionsList):
	for each options in optionsList:
		if options is null or undefined:
			continue

	for each key, value in options:
		if value is not null or undefined:
			target[key] = value

	return target
```

Currently, the driver maintains this logic for every options type, e.g. [MergeClientOptions](https://github.com/mongodb/mongo-go-driver/blob/2e7cb372b05cba29facd58aac7e715c3cec4e377/mongo/options/clientoptions.go#L1065). For v2, we’ve decided to abstract the merge functions by changing the options builder pattern to maintain a slice of setter functions, rather than setting data directly to an options object. Typical usage of options should not change, for example the following is still honored:

```go
opts1 := options.Client().SetAppName("appName")
opts2 := options.Client().SetConnectTimeout(math.MaxInt64)

_, err := mongo.Connect(opts1, opts2)
if err != nil {
	panic(err)
}
```

There are two notable cases that will require a migration: (1) modifying options data after building, and (2) creating a slice of options objects.

#### Modifying Fields After Building

The options builder is now a slice of setters, rather than a single options object. In order to modify the data after building, users will need to create a custom setter function and append the builder’s `Opts` slice:

```go
// v1

opts := options.Client().ApplyURI("mongodb://x:y@localhost:27017")

if opts.Auth.Username == "x" {
  opts.Auth.Password = "z"
}
```

```go

// v2

opts := options.Client().ApplyURI("mongodb://x:y@localhost:27017")

// If the username is "x", use password "z"
pwSetter := func(opts *options.ClientOptions) error {
  if opts.Auth.Username == "x" {
    opts.Auth.Password = "z"
  }

  return nil
}

opts.Opts = append(opts.Opts, pwSetter)
```

#### Creating a Slice of Options

Using options created with the builder pattern as elements in a slice:

```go
// v1

opts1 := options.Client().SetAppName("appName")
opts2 := options.Client().SetConnectTimeout(math.MaxInt64)

opts := []*options.ClientOptions{opts1, opts2}
_, err := mongo.Connect(opts...)
```

```go
// v2

opts1 := options.Client().SetAppName("appName")
opts2 := options.Client().SetConnectTimeout(math.MaxInt64)

// Option objects are "Listers" in v2, objects that hold a list of setters
opts := []options.Lister[options.ClientOptions]{opts1, opts2}
_, err := mongo.Connect(opts...)
```

#### Creating Options from Builder

Since a builder is just a slice of option setters, users can create options directly from a builder:

```go
// v1

opt := &options.ClientOptions{}
opt.ApplyURI(uri)

return clientOptionAdder{option: opt}
```

```go
// v2

var opts options.ClientOptions
for _, set := range options.Client().ApplyURI(uri).Opts {
  _ = set(&opts)
}

return clientOptionAdder{option: &opts}
```

### FindOneOptions

The following types are not valid for a `findOne` operation and have been removed:

- `BatchSize`
- `CursorType`
- `MaxAwaitTime`
- `NoCursorTimeout`

### Merge\*Options

All functions that merge options have been removed in favor of a generic solution. See [GORIVER-2696 ](https://jira.mongodb.org/browse/GORIVER-2696)for more information.

### MaxTime

Users should time out operations using either the client-level operation timeout defined by `ClientOptions.Timeout` or by setting a deadline on the context object passed to an operation. The following fields and methods have been removed:

- `AggregateOptions.MaxTime` and `AggregateOptions.SetMaxTime`
- `ClientOptions.SocketTimeout` and `ClientOptions.SetSocketTimeout`
- `CountOptions.MaxTime` and `CountOptions.SetMaxTime`
- `DistinctOptions.MaxTime` and `DistinctOptions.SetMaxTime`
- `EstimatedDocumentCountOptions.MaxTime` and `EstimatedDocumentCountOptions.SetMaxTime`
- `FindOptions.MaxTime` and `FindOptions.SetMaxTime`
- `FindOneOptions.MaxTime` and `FindOneOptions.SetMaxTime`
- `FindOneAndReplaceOptions.MaxTime` and `FindOneAndReplaceOptions.SetMaxTime`
- `FindOneAndUpdateOptions.MaxTime` and `FindOneAndUpdateOptions.SetMaxTime`
- `GridFSFindOptions.MaxTime` and `GridFSFindOptions.SetMaxTime`
- `CreateIndexesOptions.MaxTime` and `CreateIndexesOptions.SetMaxTime`
- `DropIndexesOptions.MaxTime` and `DropIndexesOptions.SetMaxTime`
- `ListIndexesOptions.MaxTime` and `ListIndexesOptions.SetMaxTime`
- `SessionOptions.DefaultMaxCommitTime` and `SessionOptions.SetDefaultMaxCommitTime`
- `TransactionOptions.MaxCommitTime` and `TransactionOptions.SetMaxCommitTime`

This example illustrates how to define an operation-level timeout using v2, without loss of generality.

### SessionOptions

`DefaultReadConcern`, `DefaultReadPreference`, and `DefaultWriteConcern` are all specific to transactions started by the session. Rather than maintain three fields on the Session struct, v2 has combined these options into `DefaultTransactionOptions` which specifies a `TransactionOptions` object.

```go
// v1

sessOpts := options.Session().SetDefaultReadPreference(readpref.Primary())
```

```go
// v2

txnOpts := options.Transaction().SetReadPreference(readpref.Primary())
sessOpts := options.Session().SetDefaultTransactionOptions(txnOpts)
```

## Read Concern Package

The `Option` type and associated builder functions have been removed in v2 in favor of a `ReadConcern` literal declaration.

### Options Builder

This example shows how to update usage of `New()` and `Level()` options builder with a `ReadConcern` literal declaration.

```go
// v1

localRC := readconcern.New(readconcern.Level("local"))
```

```go
// v2

localRC := &readconcern.ReadConcern{Level: "local"}
```

### ReadConcern.GetLevel()

The `ReadConcernGetLevel()` method has been removed. Use the `ReadConcern.Level` field to get the level instead.

## Write Concern Package

### WTimeout

The `WTimeout` field has been removed from the `WriteConcern` struct. Instead, users should define a timeout at the operation-level using a context object.
