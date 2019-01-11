# Topology Package Design
This document outlines the design for this package. The design related for the driver package will
be documented in that package's DESIGN.md file.

# Topology, Server, and Connection
This package implements the operation execution stack described in the `driver` package's design.
The `Topology` type implements the `driver.Deployment` interface, `Server` implements the
`driver.Server` interface, and `Connection` implements the `driver.Connection` interface.

## Topology Design
The design of the `Topology` type remains the same except for a change to the `SelectServer` method,
which now returns a `driver.Server`.

## Server Design
The design of the `Server` type largely remains the same.

### Connection method
The `Connection` method to return a `driver.Connection` and adding a new `LegacyConnection` method,
which returns a `connection.Connection`.

### Connecting
Previously, the `Connect` method of `Server` took a `context.Context`. While this makes sense for a
`Connect` implementation that is synchronous and does I/O, the implementation of `Connect` is
asynchronous and the parameter is never used. Therefore, there is no `context.Context` parameter for
`Server`, which helps avoid confusion about what `Connect` does.

### Pool
The `*pool` type handles connection pooling. The connection's it holds are of type `*connection`,
but those are not returned outside of a `*Server`, instead a wrapper type `*Connection` is returned.
This ensures that when the `Close` method is called the connection is returned to the pool instead
of being closed. It also ensures that the connection can no longer be used.

### Legacy connection
The `legacyConnection` type is a temporary type used while the transition from the legacy driver
package is made. This type implements the `connection.Connection` type and wraps the `connection`
type. Since the `connection.connection` type was previously responsible for command monitoring, this
type will also be responsible for command monitoring. The `connection` type is not responsible for
this because it is handled by the `OperationContext` type within the `driver` package. Since this
type wraps `connection` and not `Connection` it needs to handle returning the underlying
`connection` to the pool or closing it if there was an unrecoverable error.

This type is implemented by converting the bytes version of a wire message into a
`wiremessage.WireMessage` type. The rest of the functionality that was handled by the
`connection.connection` type is handled by `connection`.

# Implementation
The follow section lays out the implementation and migration process.

## Base `driver` package
Before we can update the `topology` package, we first need the basic interfaces defined in the
`driver` package. At a minimum we need the `Deployment`, `Server`, and `Connection` interfaces. This
will require renaming the current `driver` package to `driverlegacy` and creating a new `driver`
package.

## Update `topology.Topology` type
The `topology.Topology` type will require the following changes:

- Rename `SelectServer` to `SelectServerLegacy`
- Add a new method `SelectServer` that implements the `SelectServer` method of `driver.Deployment`
  interface

## Update `topology.Server` type
The `topology.Server` type will require the following changes:

- Remove the `context.Context` argument from `Connect`, since it's not actually used
- Move the `topo` parameter from the `NewServer` method to the `Connect` method
    - This is necessary because `Disconnect` sets this to `nil`. If we can't reset it then a
      `Server` can no longer be connected usefully after it is disconnected
- Rename `Connection` to `ConnectionLegacy`
- Add a new method `Connection` that implements the `Connection` method of the `driver.Server`
  interface
- Move the `connection.Pool` type into the `topology` package
    - Remove the `Pool` interface, since we no longer need to expose it
- Rename `sconn` to `connection`
- Move the required functionality from `connection.connection` into `topology.connection`
- Add a `Connection` type, this will serve the role that `pooledConnection` did previously.
    - This type is returned from `Server.Connection`, but `pool.Connection` should return
      `connection`
- Implement SDAM error handling on `connection`
    - This is different from the previous implementation since error extraction occurs in the
      `driver` package. We should directly inspect the wire message bytes to determine if there was
      a command error (include write concern errors) and update the server description and drain the
      pool as required

## Implement the `legacyConnection` type
Since the `topology` package will be migrated first, it needs to function with both the `driver` and
`driverlegacy` packages. To enable this, we need a secondary connection type that implements the
legacy functionality. This type is called `legacyConnection` and it handles the glue code necessary
for `driverlegacy` to continue functioning. The type should be implemented as outlined above. The
`ConnectionLegacy` method of `Server` returns this type, although the actual return type remains a
`connection.Connection`.
