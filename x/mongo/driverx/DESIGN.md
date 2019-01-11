# Driver Library Design
This document outlines the design for this package. The design for the releated topology package
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

# Selector, Executor, and RetryableExecutor
A `Selector` curries the `SelectServer` method of a `Deployment` and serves as a helper method on
Operations. An `Executor` transforms itself into a wire message, writes it to a `Server`, and reads
the response. Errors are returned from the `Execute` method, results are not. Operation
implementations in this package return results via a `Result` method. A `RetryableExecutor`
encapsulates an `Executor` capable of retrying execution. Operations in this package
implement `Selector` and `Executor` and retryable operations implement `RetryableExecutor`.

Operations implement two methods that combine Selecting and Executing and Selecting and Retrying.
Options for an Operation are set via builder-style methods which return the Operation.

Below is an example of executing a theoretical retryable operation.

```go
func example(ctx context.Context, d Deployment) (TheoreticalResult, error) {
    var te TheoreticalResult
    to := TheoreticalOperation().Retry(true).Ordered(true).BatchSize(7).Deployment(d)
    srvr, err := to.Select(ctx)
    if err != nil {
        return te, err
    }
    err = to.Execute(ctx, srvr)
    if Retryable(err) {
        original := err
        srvr, err = to.Select(ctx)
        if err != nil {
            return te, original
        }
        err = to.RetryExecute(ctx, srvr, original)
    }
    if err != nil {
        return te, err
    }
    return to.Result(), nil
}
```

Below is the same example, using the SelectAndExecute and SelectAndRetryExecute helper methods:

```go
func example(ctx context.Context, d Deployment) (TheoreticalResult, error) {
    var te TheoreticalResult
    to := TheoreticalOperation().Retry(true).Ordered(true).BatchSize(7).Deployment(d)
    err := to.SelectAndExecute(ctx)
    if Retryable(err) {
        original := err
        err = to.SelectAndRetryExecute(ctx, original)
    }
    if err != nil {
        return te, err
    }
    return to.Result(), nil
}
```

Finally, a helper function, SelectExecuteAndRetry is provided to encapsulate this:

```go
func example(ctx context.Context, d Deployment) (TheoreticalResult, error) {
    var te TheoreticalResult
    to := TheoreticalOperation().Retry(true).Ordered(true).BatchSize(7).Deployment(d)
    err := SelectExecuteAndRetry(ctx, to)
    if err != nil {
        return te, err
    }
    return to.Result(), nil
}
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
wireversion of the server is less than 4.

The set of known fields are as follows:

- \*writeconcern.WriteConcern
- \*readconcern.ReadConcern
- \*session.Client
- \*session.ClusterClock

### Result Generation
Generating a result cannot be handled by code generation, so this code must be written by hand. A
method called `handleResponse` will be called during `execute`. This method takes a
`bsoncore.Document` and an `error` and returns an `error`. For operations that do not batch split, this error
is returned directly. For operations that do batch split, this error is only returned if it's
non-nil.
