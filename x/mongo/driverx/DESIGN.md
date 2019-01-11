# Driver Library Design
This document outlines the design for this package. The design for the related topology package
will be documented in that package's DESIGN.md file.

# Deployment, Server, and Connection
Acquiring a `Connection` from a `Server` selected from a `Deployment` enables sending and receiving
wire messages. A `Deployment` represents an set of MongoDB servers and a `Server` represents a
member of that set. These three types form the operation execution stack.

```go
// Deployment represents a set of MongoDB servers from which one can be selected for use.
type Deployment interface {
	SelectServer(context.Context, description.ServerSelector) (Server, error)
	Description() description.Topology
}

// Server represents a MongoDB server. Implementations should pool connections.
type Server interface {
	Connection(context.Context) (Connection, error)
}

// Connection represents a connection to a MongoDB server.
type Connection interface {
	WriteWireMessage(context.Context, []byte) error
	ReadWireMessage(ctx context.Context, dst []byte) ([]byte, error)
	Description() description.Server
	Close() error
	ID() string
}
```

# Operations and Executor
An `Executor` selects a `Server`, transforms itself into a wire message, writes it to a `Connection`
from the selected `Server`, and reads the response. Errors are returned from the `Execute` method,
results are not. Operation implementations in this package return results via a `Result` method.
Retryable operations handle retrying inside the `Execute` method.

```go
// Executor handles running an operation.
type Executor interface {
    Execute(context.Context) error
}
```

Below is an example of executing a theoretical retryable operation.

```go
func example(ctx context.Context, d Deployment) (TheoreticalResult, error) {
    var tr TheoreticalResult
    to := TheoreticalOperation().Retry(RetryOnce).Ordered(true).BatchSize(7).Deployment(d)
    err := to.Execute(ctx)
    if err != nil {
        return tr, err
    }
    return to.Result(), nil
}
```

# Batch Cursors
This package implements a `BatchCursor` type that is returned from operations that result in a stream
of documents. While the API for `BatchCursor` is below.

```go
// BatchCursor is a batch implementation of a cursor. It returns documents
// in entire batches instead of one at a time. An individual document
// cursor can be built on top of this batch cursor.
type BatchCursor struct {
    // contains filtered or unexported fields
}

// Batch will return a DocumentSequence for the current batch of documents.
// The returned DocumentSequence is only valid until the next call to Next
// or Close.
func (bc *BatchCursor) Batch() *bsoncore.DocumentSequence

// Close closes this batch cursor.
func (bc *BatchCursor) Close(ctx context.Context) error

// Err returns the latest error encountered.
func (bc *BatchCursor) Err() error

// ID returns the cursor ID for this batch cursor.
func (bc *BatchCursor) ID() int64

// Next indicates if there is another batch available. Returning false does
// not necessarily indicate that the cursor is closed. This method will
// return false when an empty batch is returned.
//
// If Next returns true, there is a valid batch of documents available. If
// Next returns false, there is not a valid batch of documents available.
func (bc *BatchCursor) Next(ctx context.Context) bool

// Server returns a pointer to the cursor's server.
func (bc *BatchCursor) Server() *topology.Server
```

# Implementation
This section covers implementation details for this design.

## Drivergen
The `drivergen` command line application handles code generation for the repetitive components of
this package. The application uses struct tags defined on Operation to generate setter methods and
components of serializing the Operation into a wire message.

The `drivergen` struct tag is used code generation. The first field of the tag is the name. The
following fields are options that configure the meaning of the field and of the name. A value of `-`
indicates that this field should be ignored by `drivergen`.

Each tag has a type. The default type is a Setter, which will generate a setter method for that
field. All setters are assumed to be command parameters unless they are of a known type, e.g.
\*writeconcern.WriteConcern, in which case they will be handled specially. Two other types are
available: msgDocSeq and commandName. A type of msgDocSeq indicates that this field is an OP\_MSG
document sequence. A type of commandName indicates that this field's name is used as the
commandName. The Go field type for commandName should be `struct{}` so it doesn't take up any memory
space within the struct.

The `variadic` option indicates that a setter's parameter should be variadic instead of a slice.

The `pointerExempt` option indicates that a setter's parameter type should be a pointer and assigned
directly to the field instead of a value who's address is assigned to the field. This option does
not need to be set for known fields.

The `legacy` option indicates that this command requires different execution on legacy servers. This
option will cause code to be generated that will call an `executeLegacy` method when the max
wire version of the server is less than 4.

The set of known fields are as follows:

- \*writeconcern.WriteConcern
- \*readconcern.ReadConcern
- \*readpref.ReadPref
- \*session.Client
- \*session.ClusterClock
- description.ServerSelector

### Sessions
Sessions are handled by the consumer of this package. This package does not create nor close
sessions. This package will identify whether a session is implicit or explicit and error
accordingly. In the case of an explicit session, if the `Deployment` or selected `Connection` does not support
sessions an error is returned and the operation is not run. In the case of an implicit session, if
the `Deployment` or selected `Connection` does not support sessions, the session is not encoded into
the command that is sent. `BatchCursor`s returned from this package will use the session from the
original operation for subsequent get more operations.

### Retryability
Retries are handled via the `RetryMode` type. The three modes are `RetryOnce`, `RetryOncePerCommand`,
and `RetryContext`. The `RetryOnce` mode will retry the entire operation at most once, even if it's
comprised of multiple commands. The `RetryOncePerCommand` mode will retry each command the operation
creates at most once. The `RetryContext` mode will retry the operation and each command within the
operation until the `context.Context` either hits the deadline or is cancelled.

### Result Generation
Generating a result cannot be handled by code generation, so this code must be written by hand. A
method called `processResponse` will be called during `Execute`. This method takes a
`bsoncore.Document` and returns an `error`. This error is For operations that do not batch split, this error
is returned directly. For operations that do batch split, this error is only returned if it's
non-nil.

### Errors
In the previous `driver` package, write errors were returned as part of the result object instead of
an error of their own. The `mongo` package took care of separating the errors and returning them
explicitly as an error. Instead of combining the results and errors together, this design returns
write errors as errors from the `driver` package.

### Generated Methods
Methods are generated for each specified command parameter field. Additionally, the following
methods are generated:

- Select
- SelectAndExecute
- Execute
- RetryExecute (for retryable operations)
- SelectExecuteAndRetry (for retryable operations)
- execute
- createWireMessage
- createMsgWireMessage
- createQueryWireMessage

# API Examples
Below are examples of the public API this package will expose for operations.

```go
type CommandOperation struct {
    // contains filtered or unexported fields
}

// Command constructs and returns a new CommandOperation.
func Command(cmd bsoncore.Document) *CommandOperation {}

func (co *CommandOperation) Clock(clock *session.ClusterClock) *CommandOperation {}

// Command sets the command that will be run.
func (co *CommandOperation) Command(cmd bsoncore.Document) *CommandOperation {}

// Database sets the database to run the command against.
func (co *CommandOperation) Database(database string) *CommandOperation {}

// Deployment sets the Deployment to run the command against.
func (co *CommandOperation) Deployment(d Deployment) *CommandOperation {}

// Execute runs this operation against the provided server.
func (co *CommandOperation) Execute(ctx context.Context, srvr Server) error {}

// ReadConcern sets the read concern to use when running the command.
func (co *CommandOperation) ReadConcern(rc *readconcern.ReadConcern) *CommandOperation {}

func (co *CommandOperation) ReadPreference(readPref *readpref.ReadPref) *CommandOperation {}

func (co *CommandOperation) Result() bsoncore.Document {}

// Select retrieves a server to be used when executing an operation.
func (co *CommandOperation) Select(ctx context.Context) (Server, error) {}

// SelectAndExecute selects a server and runs this operation against it.
func (co *CommandOperation) SelectAndExecute(ctx context.Context) error {}

func (co *CommandOperation) ServerSelector(selector description.ServerSelector) *CommandOperation {}

func (co *CommandOperation) Session(client *session.Client) *CommandOperation {}
```

```go
type FindOperation struct {
    // contains filtered or unexported fields
}

// Find constructs and returns a new FindOperation.
func Find(filter bson.Raw) *FindOperation {}

// AllowPartialResults when true allows partial results to be returned if
// some shards are down.
func (fo *FindOperation) AllowPartialResults(allowPartialResults bool) *FindOperation {}

// AwaitData when true makes a cursor block before returning when no data
// is available.
func (fo *FindOperation) AwaitData(awaitData bool) *FindOperation {}

// BatchSize specifies the number of documents to return in every batch.
func (fo *FindOperation) BatchSize(batchSize int64) *FindOperation {}

// ClusterClock sets the cluster clock for this operation.
func (fo *FindOperation) ClusterClock(clock *session.ClusterClock) *FindOperation {}

// Collation specifies a collation to be used.
func (fo *FindOperation) Collation(collation bson.Raw) *FindOperation {}

// Comment sets a string to help trace an operation.
func (fo *FindOperation) Comment(comment string) *FindOperation {}

// Deployment sets the deployment to use for this operation.
func (fo *FindOperation) Deployment(d Deployment) *FindOperation {}

// Execute runs this operation against the provided server.
func (fo *FindOperation) Execute(ctx context.Context, srvr Server) error {}

// Filter determines what results are returned from find.
func (fo *FindOperation) Filter(filter bson.Raw) *FindOperation {}

// Hint specifies the index to use.
func (fo *FindOperation) Hint(hint bson.RawValue) *FindOperation {}

// Limit sets a limit on the number of documents to return.
func (fo *FindOperation) Limit(limit int64) *FindOperation {}

// Max sets an exclusive upper bound for a specific index.
func (fo *FindOperation) Max(max bson.Raw) *FindOperation {}

// MaxTimeMS specifies the maximum amount of time to allow the query to
// run.
func (fo *FindOperation) MaxTimeMS(maxTimeMS int64) *FindOperation {}

// Min sets an inclusive lower bound for a specific index.
func (fo *FindOperation) Min(min bson.Raw) *FindOperation {}

// Namespace sets the database and collection to run this operation
// against.
func (fo *FindOperation) Namespace(ns Namespace) *FindOperation {}

// NoCursorTimeout when true prevents cursor from timing out after an
// inactivity period.
func (fo *FindOperation) NoCursorTimeout(noCursorTimeout bool) *FindOperation {}

// OplogReplay when true replays a replica set's oplog.
func (fo *FindOperation) OplogReplay(oplogReplay bool) *FindOperation {}

// Project limits the fields returned for all documents.
func (fo *FindOperation) Projection(projection bson.Raw) *FindOperation {}

// ReadConcern specifies the read concern for this operation.
func (fo *FindOperation) ReadConcern(readConcern *readconcern.ReadConcern) *FindOperation {}

// ReadPreference set the read prefernce used with this operation.
func (fo *FindOperation) ReadPreference(readPref *readpref.ReadPref) *FindOperation {}

// ReturnKey when true returns index keys for all result documents.
func (fo *FindOperation) ReturnKey(returnKey bool) *FindOperation {}

// Select retrieves a server to be used when executing an operation.
func (fo *FindOperation) Select(ctx context.Context) (Server, error) {}

// SelectAndExecute selects a server and runs this operation against it.
func (fo *FindOperation) SelectAndExecute(ctx context.Context) error {}

// ServerSelector sets the selector used to retrieve a server.
func (fo *FindOperation) ServerSelector(serverSelector description.ServerSelector) *FindOperation {}

// Session sets the session for this operation.
func (fo *FindOperation) Session(client *session.Client) *FindOperation {}

// ShowRecordID when true adds a $recordId field with the record identifier
// to returned documents.
func (fo *FindOperation) ShowRecordID(showRecordID bool) *FindOperation {}

// SingleBatch specifies whether the results should be returned in a single
// batch.
func (fo *FindOperation) SingleBatch(singleBatch bool) *FindOperation {}

// Skip specifies the number of documents to skip before returning.
func (fo *FindOperation) Skip(skip int64) *FindOperation {}

// Sort specifies the order in which to return results.
func (fo *FindOperation) Sort(sort bson.Raw) *FindOperation {}

// Tailable keeps a cursor open and resumable after the last data has been
// retrieved.
func (fo *FindOperation) Tailable(tailable bool) *FindOperation {}
```

```go
type InsertOperation struct {
    // contains filtered or unexported fields
}

// Insert constructs and returns a new InsertOperation.
func Insert(documents ...bsoncore.Document) *InsertOperation {}

// BypassDocumentValidation allows the operation to opt-out of document
// level validation. Valid for server versions >= 3.2. For servers < 3.2,
// this setting is ignored.
func (io *InsertOperation) BypassDocumentValidation(bypassDocumentValidation bool) *InsertOperation {}

// ClusterClock sets the cluster clock for this operation.
func (io *InsertOperation) ClusterClock(clock *session.ClusterClock) *InsertOperation {}

// Deployment sets the deployment to use for this operation.
func (io *InsertOperation) Deployment(d Deployment) *InsertOperation {}

// Documents adds documents to this operation that will be inserted when
// this operation is executed.
func (io *InsertOperation) Documents(documents ...bsoncore.Document) *InsertOperation {}

// Execute runs this operation against the provided server. If the error
// returned is retryable, either SelectAndRetryExecute or Select followed
// by RetryExecute can be called to retry this operation.
func (io *InsertOperation) Execute(ctx context.Context, srvr Server) error {}

// Namespace sets the database and collection to run this operation
// against.
func (io *InsertOperation) Namespace(ns Namespace) *InsertOperation {}

// Ordered sets ordered. If true, when a write fails, the operation will
// return the error, when false write failures do not stop execution of the
// operation.
func (io *InsertOperation) Ordered(ordered bool) *InsertOperation {}

// Result returns the result from executing this operation. This should
// only be called after Execute and any retries.
func (io *InsertOperation) Result() result.Insert {}

// Retry enables retryable writes for this operation. Retries are not
// handled automatically, instead a boolean is returned from Execute and
// SelectAndExecute that indicates if the operation can be retried.
// Retrying is handled by calling RetryExecute.
func (io *InsertOperation) Retry(retry bool) *InsertOperation {}

// RetryExecute retries this operation against the provided server. This
// method should only be called after a retryable error is returned from
// either SelectAndExecute or Execute.
func (io *InsertOperation) RetryExecute(ctx context.Context, srvr Server, original error) error {}

// Select retrieves a server to be used when executing an operation.
func (io *InsertOperation) Select(ctx context.Context) (Server, error) {}

// SelectAndExecute selects a server and runs this operation against it.
func (io *InsertOperation) SelectAndExecute(ctx context.Context) error {}

// SelectAndExecute selects a server and retries this operation against it.
// This is a convenience method and should only be called after a retryable
// error is returned from SelectAndExecute or Execute.
func (io *InsertOperation) SelectAndRetryExecute(ctx context.Context, original error) error {}

// ServerSelector sets the selector used to retrieve a server.
func (io *InsertOperation) ServerSelector(serverSelector description.ServerSelector) *InsertOperation {}

// Session sets the session for this operation.
func (io *InsertOperation) Session(client *session.Client) *InsertOperation {}

// WriteConcern sets the write concern for this operation.
func (io *InsertOperation) WriteConcern(writeConcern *writeconcern.WriteConcern) *InsertOperation {}
```

# Open Questions

## How much and what type of code should be generated?
During the design process, reviewers raised a question of how much and what type of code is
generated. Their desire was to generate boilerplate code but not generate repeated code. Finding a
balance between generating repeated code versus separating that code into standalone functions is
being deferred until implementation when more information about the full capabilities of the code
generation tool is available.
