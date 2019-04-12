# Topology Package Design
This document outlines the design for this package.

## Connection
Connections are handled by two main types and an auxiliary type. The two main types are `connection`
and `Connection`. The first holds most of the logic required to actually read and write wire
messages. Instances can be created with the `newConnection` method. Inside the `newConnection`
method the auxiliary type, `initConnection` is used to perform the connection handshake. This is
required because the `connection` type does not fully implement `driver.Connection` which is
required during handshaking. The `Connection` type is what is actually returned to a consumer of the
`topology` package. This type does implement the `driver.Connection` type, holds a reference to a
`connection` instance, and exists mainly to prevent accidental continued usage of a connection after
closing it.

The connection implementations in this package are conduits for wire messages but they have no
ability to encode, decode, or validate wire messages. That must be handled by consumers.
