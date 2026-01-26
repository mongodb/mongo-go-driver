# Migrating from 1.x to 2.0

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

`Client.Connect()` has been removed in favor of `mongo.Connect()`. See the section on `NewClient` for more details.

The `context.Context` parameter has been removed from `mongo.Connect()` because the [deployment connector](https://github.com/mongodb/mongo-go-driver/blob/a76687682f080c9612295b646a00650d00dd16e1/x/mongo/driver/driver.go#L91) doesn’t accept a context, meaning that the context passed to `mongo.Connect()` in previous versions didn't serve a purpose.

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

The `Distinct()` collection method returns a struct that can be decoded, similar to `Collection.FindOne`. Instead of iterating through an untyped slice, users can decode same-type data using conventional Go syntax.

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
  log.Fatal("failed to decode distinct result: %v", err)
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

#### InsertMany

The `documents` parameter in the `Collection.InsertMany` function signature has been changed from an `[]any` type to an `any` type. This API no longer requires users to copy existing slice of documents to an `[]any` slice.

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

booksi := make([]any, len(books))
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
```

### Database

#### ListCollectionSpecifications

`ListCollectionSpecifications()` returns a slice of structs instead of a slice of pointers.

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

Dropping an index replies with a superset of the following message: `{nIndexesWas: n}`, where n indicates the number of indexes there were prior to removing whichever index(es) were dropped. In the case of `DropAll` this number is always `m - 1`, where m is the total number of indexes as you cannot delete the index on the `_id` field. Thus, we can simplify the `DropAll` method by removing the server response.

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

<!-- markdown-link-check-disable -->

Updated to return a slice of structs instead of a slice of pointers. See the [database analogue](#ListCollectionSpecifications) for migration guide.

<!-- markdown-link-check-enable -->

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

`ClientOptions.AuthenticateToAnything` was removed in v2 (it was marked for internal use in v1).

The following fields were removed because they are no longer supported by the server

- `FindOptions.Snapshot` (4.0)
- `FindOneOptions.Snapshot` (4.0)
- `IndexOptions.Background` (4.2)

### Options

The Go Driver offers users the ability to pass multiple options objects to operations in a last-on-wins algorithm, merging data at a field level:

```pseudo
function MergeOptions(target, optionsList):
  for each options in optionsList:
    if options is null or undefined:
      continue

  for each key, value in options:
    if value is not null or undefined:
      target[key] = value

  return target
```

Currently, the driver maintains this logic for every options type, e.g. [MergeFindOptions](https://github.com/mongodb/mongo-go-driver/blob/2e7cb372b05cba29facd58aac7e715c3cec4e377/mongo/options/findoptions.go#L257). For v2, we’ve decided to abstract the merge functions by changing the options builder pattern to maintain a slice of setter functions, rather than setting data directly to an options object. Typical usage of options should not change, for example the following is still honored:

```go
opts1 := options.Find().SetBatchSize(1)
opts2 := options.Find().SetComment("foo")

_, err := coll.Find(context.TODO(), bson.D{{"x", 1"}}, opts1, opts2)
if err != nil {
	panic(err)
}
```

There are two notable cases that will require a migration: (1) modifying options data after building, and (2) creating a slice of options objects.

#### Modifying Fields After Building

The options builder is now a slice of setters, rather than a single options object. In order to modify the data after building, users will need to create a custom setter function and append the builder’s `Opts` slice:

```go
// v1

opts := options.Find().SetBatchSize(1)

if opts.MaxAwaitTime == nil {
  opts.MaxAwaitTime = &defaultMaxAwaitTime
}
```

```go
// v2

opts := options.Find().SetBatchSize(1)

maxAwaitTimeSetter := func(opts *options.FindOptions) error {
  if opts.MaxAwaitTime == nil {
    opts.MaxAwaitTime = &defaultMaxAwaitTime
  }

  return nil
}

opts.Opts = append(opts.Opts, maxAwaitTimeSetter)
```

#### Creating a Slice of Options

Using options created with the builder pattern as elements in a slice:

```go
// v1

opts1 := options.Find().SetBatchSize(1)
opts2 := options.Find().SetComment("foo")

opts := []*options.FindOptions{opts1, opts2}
_, err := coll.Find(context.TODO(), bson.D{{"x", 1"}}, opts...)
```

```go
// v2

opts1 := options.Find().SetBatchSize(1)
opts2 := options.Find().SetComment("foo")

opts := []options.Lister[options.FindOptions]{opts1, opts2}
_, err := coll.Find(context.TODO(), bson.D{{"x", 1"}}, opts...)
```

#### Creating Options from Builder

Since a builder is just a slice of option setters, users can create options directly from a builder:

```go
// v1

opt := &options.FindOptions{}
opt.SetBatchSize(1)

return findOptionAdder{option: opt}
```

```go
// v2

var opts options.FindOptions
for _, set := range options.Find().SetBatchSize(1).Opts {
  _ = set(&opts)
}

return findOptionAdder{option: &opts}
```

### DeleteManyOptions / DeleteOneOptions

The `DeleteOptions` has been separated into `DeleteManyOptions` and `DeleteOneOptions` to configure the corresponding `DeleteMany` and `DeleteOne` operations.

### FindOneOptions

The following types are not valid for a `findOne` operation and have been removed:

- `BatchSize`
- `CursorType`
- `MaxAwaitTime`
- `NoCursorTimeout`

### FindOneAndUpdateOptions

The `ArrayFilters` struct type has been removed in v2. As a result, the `ArrayFilters` field in the `FindOneAndUpdateOptions` struct now uses the `[]any` type. The `ArrayFilters` field (now of type `[]any`) serves the same purpose as the `Filters` field in the original `ArrayFilters` struct.

### UpdateManyOptions / UpdateOneOptions

The `UpdateOptions` has been separated into `UpdateManyOptions` and `UpdateOneOptions` to configure the corresponding `UpdateMany` and `UpdateOne` operations.

The `ArrayFilters` struct type has been removed in v2. As a result, the `ArrayFilters` fields in the new `UpdateManyOptions` and `UpdateOneOptions` structs (which replace the old `UpdateOptions` struct) now use the `[]any` type. The `ArrayFilters` field (now of type `[]any`) serves the same purpose as the `Filters` field in the original `ArrayFilters` struct.

### Merge\*Options

With the exception of `MergeClientOptions`, all functions that merge options have been removed in favor of a generic solution. `MergeClientOptions` is retained to allow combining `*ClientOptions` in a "last-one-wins" fashion. See [GODRIVER-2696](https://jira.mongodb.org/browse/GODRIVER-2696) for more information.

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

`DefaultReadConcern`, `DefaultReadPreference`, and `DefaultWriteConcern` are all specific to transactions started by the session. Rather than maintain three fields on the `Session` struct, v2 has combined these options into `DefaultTransactionOptions` which specifies a `TransactionOptions` object.

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

## Bsoncodecs / Bsonoptions Package

`*Codec` structs and `New*Codec` methods have been removed. Additionally, the correlated `bson/bsonoptions` package has been removed, so codecs are not directly configurable using `*CodecOptions` structs in Go Driver 2.0. To configure the encode and decode behavior, use the configuration methods on a `bson.Encoder` or `bson.Decoder`. To configure the encode and decode behavior for a `mongo.Client`, use `options.ClientOptions.SetBSONOptions` with `BSONOptions`.

This example shows how to set `ObjectIDAsHex`.

```go
// v1

var res struct {
	ID string
}

codecOpt := bsonoptions.StringCodec().SetDecodeObjectIDAsHex(true)
strCodec := bsoncodec.NewStringCodec(codecOpt)
reg := bson.NewRegistryBuilder().RegisterDefaultDecoder(reflect.String, strCodec).Build()
dc := bsoncodec.DecodeContext{Registry: reg}
dec, err := bson.NewDecoderWithContext(dc, bsonrw.NewBSONDocumentReader(DOCUMENT))
if err != nil {
	panic(err)
}
err = dec.Decode(&res)
if err != nil {
	panic(err)
}
```

```go
// v2

var res struct {
	ID string
}

decoder := bson.NewDecoder(bson.NewDocumentReader(bytes.NewReader(DOCUMENT)))
decoder.ObjectIDAsHexString()
err := decoder.Decode(&res)
if err != nil {
	panic(err)
}
```

### RegistryBuilder

The `RegistryBuilder` struct and the `bson.NewRegistryBuilder` function have been removed in favor of `(*bson.Decoder).SetRegistry` and `(*bson.Encoder).SetRegistry`.

### StructTag / StructTagParserFunc

The `StructTag` struct as well as the `StructTagParserFunc` type have been removed. Therefore, users have to specify BSON tags manually rather than define custom BSON tag parsers.

### TransitionError

The `TransitionError` struct has been merged into the bson package.

### Other interfaces removed from bsoncodes package

`CodecZeroer` and `Proxy` have been removed.

## Bsonrw Package

The `bson/bsonrw` package has been merged into the `bson` package.

As a result, interfaces `ArrayReader`, `ArrayWriter`, `DocumentReader`, `DocumentWriter`, `ValueReader`, and `ValueWriter` are located in the `bson` package now.

Interfaces `BytesReader`, `BytesWriter`, and `ValueWriterFlusher` have been removed.

Functions `NewExtJSONValueReader` and `NewExtJSONValueWriter` have been moved to the bson package as well.

Moreover, the `ErrInvalidJSON` variable has been merged into the bson package.

### NewBSONDocumentReader / NewBSONValueReader

The `bsonrw.NewBSONDocumentReader` has been renamed to `NewDocumentReader`, which reads from an `io.Reader`, in the `bson` package.

The `NewBSONValueReader` has been removed.

This example creates a `Decoder` that reads from a byte slice.

```go
// v1

b, _ := bson.Marshal(bson.M{"isOK": true})
decoder, err := bson.NewDecoder(bsonrw.NewBSONDocumentReader(b))
```

```go
// v2

b, _ := bson.Marshal(bson.M{"isOK": true})
decoder := bson.NewDecoder(bson.NewDocumentReader(bytes.NewReader(b)))
```

### NewBSONValueWriter

The `bsonrw.NewBSONValueWriter` function has been renamed to `NewDocumentWriter` in the `bson` package.

This example creates an `Encoder` that writes BSON values to a `bytes.Buffer`.

```go
// v1

buf := new(bytes.Buffer)
vw, err := bsonrw.NewBSONValueWriter(buf)
encoder, err := bson.NewEncoder(vw)
```

```go
// v2

buf := new(bytes.Buffer)
vw := bson.NewDocumentWriter(buf)
encoder := bson.NewEncoder(vw)
```

## Mgocompat Package

The `bson/mgocompat` has been simplified. Its implementation has been merged into the `bson` package.

`ErrSetZero` has been renamed to `ErrMgoSetZero` in the `bson` package.

`NewRegistryBuilder` function has been simplified to `NewMgoRegistry` in the `bson` package.

Similarly, `NewRespectNilValuesRegistryBuilder` function has been simplified to `NewRespectNilValuesMgoRegistry` in the `bson` package.

## Primitive Package

The `bson/primitive` package has been merged into the `bson` package.

Additionally, the `bson.D` has implemented the `json.Marshaler` and `json.Unmarshaler` interfaces, where it uses a key-value representation in "regular" (i.e. non-Extended) JSON.

The `bson.D.String` and `bson.M.String` methods return an Extended JSON representation of the document.

```go
// v2

d := D{{"a", 1}, {"b", 2}}
fmt.Printf("%s\n", d)
// Output: {"a":{"$numberInt":"1"},"b":{"$numberInt":"2"}}
```

## Bson Package

### DefaultRegistry / NewRegistryBuilder

`DefaultRegistry` has been removed. Using `bson.DefaultRegistry` to either access the default registry behavior or to globally modify the default registry, will be impacted by this change and will need to configure their registry using another mechanism.

The `NewRegistryBuilder` function has been removed along with the `bsoncodec.RegistryBuilder` struct as mentioned above.

### Decoder

The BSON decoding logic has changed to decode into a `bson.D` by default.

The example shows the behavior change.

```go
// v1

b1 := bson.M{"a": 1, "b": bson.M{"c": 2}}
b2, _ := bson.Marshal(b1)
b3 := bson.M{}
bson.Unmarshal(b2, &b3)
fmt.Printf("b3.b type: %T\n", b3["b"])
// Output: b3.b type: primitive.M
```

```go
// v2

b1 := bson.M{"a": 1, "b": bson.M{"c": 2}}
b2, _ := bson.Marshal(b1)
b3 := bson.M{}
bson.Unmarshal(b2, &b3)
fmt.Printf("b3.b type: %T\n", b3["b"])
// Output: b3.b type: bson.D
```

Use `Decoder.DefaultDocumentM()` or set the `DefaultDocumentM` field of `options.BSONOptions` to always decode documents into the `bson.M` type.

For full V1 compatibility, use `Decoder.DefaultDocumentMap()` instead. While
`bson.M` is defined as `type M map[string]any`, Go's type system treats `bson.M`
and `map[string]any` as distinct types. This can break compatibility with
libraries that expect actual `map[string]any` types.

```go
b1 := map[string]any{"a": 1, "b": map[string]any{"c": 2}}
b2, _ := bson.Marshal(b1)

decoder := bson.NewDecoder(bson.NewDocumentReader(bytes.NewReader(b2)))
decoder.DefaultDocumentMap()

var b3 map[string]any
decoder.Decode(&b3)
fmt.Printf("b3.b type: %T\n", b3["b"])
// Output: b3.b type: map[string]interface {}
```

Or configure at the client level:

```go
clientOpts := options.Client().
    SetBSONOptions(&options.BSONOptions{
        DefaultDocumentMap: true,
    })
```

#### NewDecoder

The signature of `NewDecoder` has been updated without an error being returned.

#### NewDecoderWithContext

`NewDecoderWithContext` has been removed in favor of using the `SetRegistry` method to set a registry.

Correspondingly, the following methods have been removed:

- `UnmarshalWithRegistry`
- `UnmarshalWithContext`
- `UnmarshalValueWithRegistry`
- `UnmarshalExtJSONWithRegistry`
- `UnmarshalExtJSONWithContext`

For example, a boolean type can be stored in the database as a BSON boolean or 32/64-bit integer. Given a registry:

```go
type lenientBool bool

lenientBoolType := reflect.TypeOf(lenientBool(true))

lenientBoolDecoder := func(
	dc bsoncodec.DecodeContext,
	vr bsonrw.ValueReader,
	val reflect.Value,
) error {
	// All decoder implementations should check that val is valid, settable,
	// and is of the correct kind before proceeding.
	if !val.IsValid() || !val.CanSet() || val.Type() != lenientBoolType {
		return bsoncodec.ValueDecoderError{
			Name:     "lenientBoolDecoder",
			Types:    []reflect.Type{lenientBoolType},
			Received: val,
		}
	}

	var result bool
	switch vr.Type() {
	case bsontype.Boolean:
		b, err := vr.ReadBoolean()
		if err != nil {
			return err
		}
		result = b
	case bsontype.Int32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}
		result = i32 != 0
	case bsontype.Int64:
		i64, err := vr.ReadInt64()
		if err != nil {
			return err
		}
		result = i64 != 0
	default:
		return fmt.Errorf(
			"received invalid BSON type to decode into lenientBool: %s",
			vr.Type())
	}

	val.SetBool(result)
	return nil
}

// Create the registry
reg := bson.NewRegistry()
reg.RegisterTypeDecoder(
	lenientBoolType,
	bsoncodec.ValueDecoderFunc(lenientBoolDecoder))
```

For our custom decoder with such a registry, BSON 32/64-bit integer values are considered `true` if they are non-zero.

```go
// v1
// Use UnmarshalWithRegistry

// Marshal a BSON document with a single field "isOK" that is a non-zero
// integer value.
b, err := bson.Marshal(bson.M{"isOK": 1})
if err != nil {
	panic(err)
}

// Now try to decode the BSON document to a struct with a field "IsOK" that
// is type lenientBool. Expect that the non-zero integer value is decoded
// as boolean true.
type MyDocument struct {
	IsOK lenientBool `bson:"isOK"`
}
var doc MyDocument
err = bson.UnmarshalWithRegistry(reg, b, &doc)
if err != nil {
	panic(err)
}
fmt.Printf("%+v\n", doc)
// Output: {IsOK:true}
```

```go
// v1
// Use NewDecoderWithContext

// Marshal a BSON document with a single field "isOK" that is a non-zero
// integer value.
b, err := bson.Marshal(bson.M{"isOK": 1})
if err != nil {
	panic(err)
}

// Now try to decode the BSON document to a struct with a field "IsOK" that
// is type lenientBool. Expect that the non-zero integer value is decoded
// as boolean true.
type MyDocument struct {
	IsOK lenientBool `bson:"isOK"`
}
var doc MyDocument
dc := bsoncodec.DecodeContext{Registry: reg}
dec, err := bson.NewDecoderWithContext(dc, bsonrw.NewBSONDocumentReader(b))
if err != nil {
	panic(err)
}
err = dec.Decode(&doc)
if err != nil {
	panic(err)
}
fmt.Printf("%+v\n", doc)
// Output: {IsOK:true}
```

```go
// v2
// Use SetRegistry

// Marshal a BSON document with a single field "isOK" that is a non-zero
// integer value.
b, err := bson.Marshal(bson.M{"isOK": 1})
if err != nil {
	panic(err)
}

// Now try to decode the BSON document to a struct with a field "IsOK" that
// is type lenientBool. Expect that the non-zero integer value is decoded
// as boolean true.
type MyDocument struct {
	IsOK lenientBool `bson:"isOK"`
}
var doc MyDocument
dec := bson.NewDecoder(bson.NewDocumentReader(bytes.NewReader(b)))
dec.SetRegistry(reg)
err = dec.Decode(&doc)
if err != nil {
	panic(err)
}
fmt.Printf("%+v\n", doc)
// Output: {IsOK:true}
```

#### SetContext

The `SetContext` method has been removed in favor of using `SetRegistry` to set the registry of a decoder.

#### SetRegistry

The signature of `SetRegistry` has been updated without an error being returned.

#### Reset

The signature of `Reset` has been updated without an error being returned.

#### DefaultDocumentD / DefaultDocumentM

`Decoder.DefaultDocumentD()` has been removed since a document, including a top-level value (e.g. you pass in an empty interface value to Decode), is always decoded into a `bson.D` by default. Therefore, use `Decoder.DefaultDocumentM()` to always decode a document into a `bson.M` to avoid unexpected decode results.

#### ObjectIDAsHexString

`Decoder.ObjectIDAsHexString()` method enables decoding a BSON ObjectId as a hexadecimal string. Otherwise, the decoder returns an error by default instead of decoding as the UTF-8 representation of the raw ObjectId bytes, which results in a garbled and unusable string.

### Encoder

#### NewEncoder

The signature of `NewEncoder` has been updated without an error being returned.

#### NewEncoderWithContext

`NewEncoderWithContext` has been removed in favor of using the `SetRegistry` method to set a registry.

Correspondingly, the following methods have been removed:

- `MarshalWithRegistry`
- `MarshalWithContext`
- `MarshalAppend`
- `MarshalAppendWithRegistry`
- `MarshalAppendWithContext`
- `MarshalValueWithRegistry`
- `MarshalValueWithContext`
- `MarshalValueAppendWithRegistry`
- `MarshalValueAppendWithContext`
- `MarshalExtJSONWithRegistry`
- `MarshalExtJSONWithContext`
- `MarshalExtJSONAppendWithRegistry`
- `MarshalExtJSONAppendWithContext`

Here is an example of a registry that multiplies the input value by -1 when encoding for a `negatedInt`.

```go
type negatedInt int

negatedIntType := reflect.TypeOf(negatedInt(0))

negatedIntEncoder := func(
	ec bsoncodec.EncodeContext,
	vw bsonrw.ValueWriter,
	val reflect.Value,
) error {
	// All encoder implementations should check that val is valid and is of
	// the correct type before proceeding.
	if !val.IsValid() || val.Type() != negatedIntType {
		return bsoncodec.ValueEncoderError{
			Name:     "negatedIntEncoder",
			Types:    []reflect.Type{negatedIntType},
			Received: val,
		}
	}

	// Negate val and encode as a BSON int32 if it can fit in 32 bits and a
	// BSON int64 otherwise.
	negatedVal := val.Int() * -1
	if math.MinInt32 <= negatedVal && negatedVal <= math.MaxInt32 {
		return vw.WriteInt32(int32(negatedVal))
	}
	return vw.WriteInt64(negatedVal)
}

// Create the registry.
reg := bson.NewRegistry()
reg.RegisterTypeEncoder(
	negatedIntType,
	bsoncodec.ValueEncoderFunc(negatedIntEncoder))
```

Encode by creating a custom encoder with the registry:

```go
// v1
// Use MarshalWithRegistry

b, err := bson.MarshalWithRegistry(reg, bson.D{{"negatedInt", negatedInt(1)}})
if err != nil {
	panic(err)
}
fmt.Println(bson.Raw(b).String())
// Output: {"negatedint": {"$numberInt":"-1"}}
```

```go
// v1
// Use NewEncoderWithContext

buf := new(bytes.Buffer)
vw, err := bsonrw.NewBSONValueWriter(buf)
if err != nil {
	panic(err)
}
ec := bsoncodec.EncodeContext{Registry: reg}
enc, err := bson.NewEncoderWithContext(ec, vw)
if err != nil {
	panic(err)
}
err = enc.Encode(bson.D{{"negatedInt", negatedInt(1)}})
if err != nil {
	panic(err)
}
fmt.Println(bson.Raw(buf.Bytes()).String())
// Output: {"negatedint": {"$numberInt":"-1"}}
```

```go
// v2
// Use SetRegistry

buf := new(bytes.Buffer)
vw := bson.NewDocumentWriter(buf)
enc := bson.NewEncoder(vw)
enc.SetRegistry(reg)
err := enc.Encode(bson.D{{"negatedInt", negatedInt(1)}})
if err != nil {
	panic(err)
}
fmt.Println(bson.Raw(buf.Bytes()).String())
// Output: {"negatedint": {"$numberInt":"-1"}}
```

#### SetContext

The `SetContext` method has been removed in favor of using `SetRegistry` to set the registry of an encoder.

#### SetRegistry

The signature of `SetRegistry` has been updated without an error being returned.

#### Reset

The signature of `Reset` has been updated without an error being returned.

### RawArray Type

A new `RawArray` type has been added to the `bson` package as a primitive type to ​​represent a BSON array. Correspondingly, `RawValue.Array()` returns a `RawArray` instead of `Raw`.

### ValueMarshaler

The `MarshalBSONValue` method of the [ValueMarshaler](https://pkg.go.dev/go.mongodb.org/mongo-driver/v2/bson#ValueMarshaler) interface now returns a `byte` value representing the [BSON type](https://pkg.go.dev/go.mongodb.org/mongo-driver/v2/bson#Type). That allows external packages to implement the `ValueMarshaler` interface without having to import the `bson` package. Convert a returned `byte` value to [bson.Type](https://pkg.go.dev/go.mongodb.org/mongo-driver/v2/bson#Type) to compare with the BSON type constants. For example:

```go
btype, _, _ := m.MarshalBSONValue()
fmt.Println("type of data: %s: ", bson.Type(btype))
fmt.Println("type of data is an array: %v", bson.Type(btype) == bson.TypeArray)
```

### ValueUnmarshaler

The `UnmarshalBSONValue` method of the [ValueUnmarshaler](https://pkg.go.dev/go.mongodb.org/mongo-driver/v2/bson#ValueUnmarshaler) interface now accepts a `byte` value representing the [BSON type](https://pkg.go.dev/go.mongodb.org/mongo-driver/v2/bson#Type) for the first argument. That allows packages to implement `ValueUnmarshaler` without having to import the `bson` package. For example:

```go
if err := m.UnmarshalBSONValue(bson.TypeEmbeddedDocument, bytes); err != nil {
    log.Fatalf("failed to decode embedded document: %v", err)
}
```
