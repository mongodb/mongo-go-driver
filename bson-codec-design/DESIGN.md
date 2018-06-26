# BSON Encoder & Decoder Design
The v1 design for the BSON library's encoder and decoder contain design issues that
require a redesign. The main design flaw is the lack of a coherent, flexible design that can
incorporate the needs of users. Assumptions made during the v1 design have proven incorrect or too
inflexible to be as useful as initially intended.

## Definitions
<dl>
<dt>Codec Processing</dt>
<dd>The complete process of either encoding or decoding.</dd>
<dt>Unmarshal</dt>
<dd>The process of converting a slice of bytes into a native Go type</dd>
<dt>Marshal</dt>
<dd>The process of converting a native Go type into a slice of bytes</dd>
<dt>Encode</dt>
<dd>The process of converting a native Go type into BSON and writing it to an
<code>io.Writer</code></dd>
<dt>Decode</dt>
<dd>The process of reading a BSON document from an <code>io.Reader</code> and converting it to a
native Go type</dd>
<dt>v1 Design </dt>
<dd>The initial design for the encoding and decoding functionality of the BSON library</dd>
<dt>Proposed Design</dt>
<dd>The design laid out in this document</dd>
<dt>Codec</dt>
<dd>A type that can encode and decode individual values to a ValueWriter or ValueReader</dd>
<dt>ValueReader</dt>
<dd>An interface that allows constructing BSON values. Implementations include BSON, Extended JSON,
and `*bson.Document`.</dd>
<dt>ValueWriter</dt>
<dd>An interface that allows reading BSON values. Implementations include BSON, Extended JSON, and
`*bson.Document`.</dd>
</dl>

## Motivation
The v1 design for codec processing in the BSON library made a number of assumptions about the
control the user had over their types, the amount of granular control users would want to have over
codec processing, and the flexibility required during codec processing. A key initial assumption of
the v1 design was that users who needed to handle types in a custom way would be able to make those
types implement the `bson.Marshaler` and `bson.Unmarshaler` interfaces. This turned out to be too
restrictive because users sometimes want to handle the processing for types they do not define
themselves. For example, users who generate types using a protocol buffer library
or users who want to handle the processing of types in the standard library.

The design of the v1 encoder and decoder for the Go driver presents problems when users wish to
have more control over how specific types are handled when encoding or decoding. The control that
users require exists on two main levels:
- control over how individual types are encoded or decoded
- control over how tags are processed

The granularity of control users require can be defined in three levels:
- Control over how individual types are encoded and decoded when they are embedded in another type,
such as a map or a struct
- Control over how individual types are encoded and decoded when they are provided directly to the
encoder or decoder
- Control over how tags are processed and the parameters that are returned

The proposed design allows for the first two granularities to be handled together.

## High Level Design
### Registries
To enable users to control how types are encoded and decoded, the proposed design uses a type
registry. This registry will be provided in any context where encoding or decoding occurs. The registry
will be used to lookup a type's codec, which will then be used for the actual encoding a decoding.
This will allow users to control nearly all aspects of the encoding and decoding process. One
failing of the v1 design is that users cannot control the encoding and decoding of types
they do not own, e.g. `time.Time`. The proposed design handles this by allowing users to register a
codec for any type.

Pointer and value types are handled separately in the registry as this simplifies the registry and
the implementations of codecs, since they will only have to worry about a single type and not the
type plus a pointer to that type. While likely not useful, there is no design
reason to prevent users from registering codecs for internal types like `bson.Element`,
`bson.Document`, or `bson.Reader`. Users can override the functionality of built in types as
well. The registry will be used for all encoding and decoding operations, since the cost of calling
`reflect.TypeOf` is inexpensive.

There is a single type, `Registry`, that contains an instance of a `TypeRegistry` and
`InterfaceRegistry`. The `Registry` type is used to find the required codec during codec processing.
The `Registry` type will also have a generic codec registry, which only contains two codecs, one for
structs and one for maps. These codecs are only used if both the `TypeRegistry` and the
`InterfaceRegistry` do not return a codec. The user can override these generic codecs.

The `Registry` type is the object users are expected to pass around and use. Call sites are expected
to acquire from a pool or create an `Encoder` or `Decoder`, use it, and then either return it to the
pool or discard it. This avoids bundling a `Registry` with an `Encoder` or a `Decoder` or having to
create a builder type that produces an `Encoder` or `Decoder`. For configuration purposes, users are
expected to pass a `Registry` into their applications or libraries.

A `Registry` is goroutine safe, allowing an application to create a single `Registry` and use it
throughout their code.

#### No Global Registries
The proposed design does not contain a global type nor a global interface registry. This is mainly
because [package level variables are generally discouraged in
Go](http://peter.bourgon.org/blog/2017/06/09/theory-of-modern-go.html). Instead, registries are
explicitly passed to the things that require them. For instance, when constructing an Encoder or a
Decoder the registry is a parameter.

The `Marshal` and `Unmarshal` functions use a default registry that is generated by a call to
`NewRegistry` and there are `MarshalWithRegistry` and `UnmarshalWithRegistry` that take a
`Registry`. This allows users who need to use a custom registry to do so while ensuring that the
functionality of `Marshal` and `Unmarshal` remains unchanged.

To enable creation of default registries, the proposed design has a function called
`NewRegistry` that handles registering the default type and interface codecs.

#### Type Registry
The type registry maps particular types to an instance of a codec. When performing an encode or
decode, this registry will be consulted for the particular codec to use. During the processing the
types are used directly and not cast. This means that if a user wants to handle an interface, a type
that implements that interfaces, and a pointer to the type that implements that interface, they can
do so by registering those three types with an associated codec. Users who want to handle interfaces
instead of the underlying types can register the interface with the Interface Registry, which will
be consulted after the Type Registry.

#### Interface Registry
The interface registry maps a particular interface to a codec. This registry is used to detect when
a type implements the given interface and handles processing of that type. For instance, the `bson`
package contains the `bson.Marshaler` and `bson.Unmarshaler` interfaces. Since an interface is less
specific than a type, the Interface Registry is consulted after the Type registry. Therefore if type
implements and interface and there is a codec registered for the type in the Type Registry and there
is a codec registered for the interface in the Interface Registry, the codec in the Type Registry
will be used. The Interface Registry must be iterated in a stable order when processing. This
ensures that if a type implements multiple interfaces, the first interface in the stable order is
used.

#### Codecs
A codec is an interface composed of two methods called `EncodeValue` and `DecodeValue`. Since a
registry can be used by multiple goroutines, codecs should ensure that they are goroutine safe.

### Struct Tag Handling
The struct codec will have a configurable struct tag handler which is responsible for parsing struct
tags and returning the values of the struct tags. The main advantage of this is allowing users to
parse struct tags that have names other than `bson`. Since private fields can't be reliably decoded,
the struct tag handler will not be run for those fields. Users can register their type with the Type
Registry if they wish to encode or decode private fields.

The following fields are recognized:
<dl>
<dt>inline</dt>
<dd>Indicates that the elements of this value be flatten into the parent. The result is the
properties or values are handled as if they were properties of the parent struct. This is only valid for
structs and maps. Only a single map can be inlined for a given struct. When a type is registered, it
is inspected to ensure that there are no conflicts, e.g. that there isn't a key of the same
name in the parent struct and an inlined struct. Conflict resolution for maps occurs during the
encoding and decoding process.</dd>
<dt>omitempty</dt>
<dd>Indicates that the field should only be encoded if it is not an empty slice or map, a nil
pointer, nor the zero value for that type. If the type implements the `bson.Zeroer` interface, then
that will be used to determine if the type is zero.</dd>
<dt>minsize</dt>
<dd>Indicates that an <code>int</code>, <code>int64</code>, <code>uint</code>, or
<code>uint64</code> should be encoded to a BSON int32 instead of a BSON int64 when the value is
small enough to fit in a BSON int32.</dd>
<dt>truncate</dt>
<dd>Indicates that a BSON double should lose precision to fit within a <code>float32</code></dd>

### Marshal, Unmarshal, Encoder, Decoder
There are two different methods of transforming types into BSON and BSON into types. The Marshal and
Unmarshal functions handle transforming to and from a slice of bytes while the Encoder and Decoder
handle transforming to an `io.Writer` and from an `io.Reader`. These naming conventions follow the
`encoding` subpackages of the standard library.

#### Encoder and Decoder
The `Encoder` and `Decoder` types can serialize and unserialize both BSON and Extended JSON. The
serialization type is determined by a mode specified when creating the `Encoder` or `Decoder`.

The serialization mode and `io.Reader`/`io.Writer` can be updated on an `Decoder` or `Encoder` by
using the `Reset` method. This allows these types to be pooled and the user to configure the type
when they use it.

#### DocumentEncoder and DocumentDecoder
The `DocumentEncoder` and `DocumentDecoder` types are used to transform a Go type to a
`*bson.Document` and to transform a `*bson.Document` to a Go type. These types are useful when a
user wants to go directly to a `*bson.Document` and does not want to first go through a `[]byte` or
wants to go directly from a `*bson.Document` to a type without going through a `[]byte` first. Since
these types aren't holding a stream state, they can be used concurrently.

### Reader and Writer types
To help facilitate a fast encoding a decoding process, new types are required that enable the
reading and writing of BSON at a more granular level. The v1 design contains a `bson.Reader` type
that is built directly from a `[]byte`. While useful for what it does, there is no way to store
position information and it can only return entire elements, not a substructure of an element. This
means it is not useful for reading an entire document and all it's subdocuments in a single pass.
Additionally, there is no Writer type. The encoding and decoding use cases require a reader and a
writer that can sequentially read or write a document, including its subdocuments, in a high
performance manner, with minimal allocations, and at a sub-element granularity.

### mgobson
To ease the transition from the mgo driver to the mongo-go-driver, the proposed design includes a
bson compatibility layer in the form of the `mgobson` library. It will support all of the types that
are supported by the `mgo/bson` package, including the `bson.M`, `bson.D`, `bson.RawD`, and
`bson.Raw` types.

This library will handle registering the `bson.Getter` and `bson.Setter` interfaces with the
Interface Registry.

### Options and Error Handling

## Code
