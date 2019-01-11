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

## OperationContext
The `OperationContext` type implements the generic logic of an operation. It handles selecting a
server, creating a wire message from a command, writing the wire message to a connection from the
selected server, reading a response from that connection, updating cluster clocks and operation
times, processing responses, and retrying. It is configurable. For operations that do not require
server selection, like getMore and killCursors, a `Server` can be provided directly. It can handle
batch splitting and retrying individual batches.

This type is public so that it can be used by external packages. This allows the `drivergen` tool to
generate code in packages outside of `driver`, which enables users to create their own operations
without requiring they be added to this package.

The type definition of `OperationContext` is below.

```go
type OperationContext struct {
    // The two fields in this group are required for all operations.

    // CommandFn is a function that's run to fill the command and it's arguments. This method should
    // not create the BSON document as it will have already been added. This function should not add
    // the read preference, read concern, or write concern fields. It should not add fields related
    // to cluster time nor client sessions. If this operation will be batch split, the documents to
    // be split should not be added by this function, instead a Batches instance should be added.
	CommandFn func(dst []byte, desc description.SelectedServer) ([]byte, error)

	// Database is the database that this operation will use.
	Database  string


    // The types in this group are pairs. They may set as either
    //      Deployment AND Selector
    //              - OR -
    //      Server AND TopologyKind
    // but no other combination is valid. It is invalid to set none of these.

    // Deployment is the MongoDB deployment to use for this operaiton.
	Deployment   Deployment

    // Selector is the server selector to use.
	Selector       description.ServerSelector

	// Server is a preselected server to use for this operaiton.
	Server       Server

	// TopologyKind is the kind of topology the server is from. This is necessary to properly encode
	// fields like read preference.
	TopologyKind description.TopologyKind


    // The fields below are optional.

    // ProcessResponseFn is a function to handle the responses from running the operation. The
    // Server used for the operaiton is provided so that cursors and similar types that require a
    // Server can be constructed.
	ProcessResponseFn func(response bsoncore.Document, srvr Server) error

    // ReadPreference is the read preference to use for this operaiton.
	ReadPreference *readpref.ReadPref

	// ReadConcern is the read concern to use for this operation.
	ReadConcern    *readconcern.ReadConcern

	// WriteConcern is the write concern to use for this operation.
	WriteConcern   *writeconcern.WriteConcern

    // Client is the client session to use for this operation. It will be updated with the results
    // of running the command.
	Client *session.Client

	// Clock is the cluster clock to use for this operation. It will be updated with the results of
	// running the command.
	Clock  *session.ClusterClock

	// CommandMonitor is the command monitor to use while running this operation.
	CommandMonitor *event.CommandMonitor

    // Batches contains the set of documents that will be batch split, along with the identifier for
    // the type 1 payload or command argument, and a boolean that specifies ordering.
	Batches   *Batches

    // While it's not invalid to set one of these fields but not the other, only setting one will
    // not enable retyrability.

    // RetryMode sets the retryability behavior. The options are no retrying, retry once, retry once
    // per command, and retry until the context hits it's deadline or is cancelled.
	RetryMode *RetryMode

	// RetryType sets the type of retry. The two options are RetryWrites and RetryReads.
	RetryType RetryType
}

// Validate will ensure that the correct fields are set. If there is an invalid configuration, this
// method will return an error describing the incorrect configuration.
func (OperationContext) Validate() error {}

// Execute will run this operation, potentially retrying depending on the configuration.
func (OperationContext) Execute(context.Context) error {}
```

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
Retries are handled via the `RetryMode` and `RetryType` types.

The three modes of `RetryMode` are `RetryOnce`, `RetryOncePerCommand`, and `RetryContext`. The
`RetryOnce` mode will retry the entire operation at most once, even if it's comprised of multiple
commands. The `RetryOncePerCommand` mode will retry each command the operation creates at most once.
The `RetryContext` mode will retry the operation and each command within the operation until the
`context.Context` either hits the deadline or is cancelled.

The two types for `RetryType` are `RetryWrites` and `RetryReads`, although the latter will be
implemented at a later time.

### Result Generation
Generating a result cannot be handled by code generation, so this code must be written by hand. A
method called `processResponse` will be called during `Execute`. This method takes a
`bsoncore.Document` and returns an `error`. This error is For operations that do not batch split, this error
is returned directly. For operations that do batch split, this error is only returned if it's
non-nil.

### Errors
Even though write command errors return `ok:1` they are still errors so this package treats them as
such. The helper functions that extract errors also extract write errors and write concern errors.
They are returned as regular errors.

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

# Open Questions

## How much and what type of code should be generated?
During the design process, reviewers raised a question of how much and what type of code is
generated. Their desire was to generate boilerplate code but not generate repeated code. Finding a
balance between generating repeated code versus separating that code into standalone functions is
being deferred until implementation when more information about the full capabilities of the code
generation tool is available.
