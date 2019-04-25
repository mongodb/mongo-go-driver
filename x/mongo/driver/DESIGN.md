# Driver Library Design
This document outlines the design for this package.

## Deployment, Server, and Connection
Acquiring a `Connection` from a `Server` selected from a `Deployment` enables sending and receiving
wire messages. A `Deployment` represents an set of MongoDB servers and a `Server` represents a
member of that set. These three types form the operation execution stack.

### Compression
Compression is handled by Connection type while uncompression is handled automatically by the
Operation type. This is done because the compressor to use for compressing a wire message is
chosen by the connection during handshake, while uncompression can be performed without this
information. This does make the design of compression non-symmetric, but it makes the design simpler
to implement and more consistent.
