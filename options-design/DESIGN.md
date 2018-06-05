# Options Design - [Design Ticket](https://jira.mongodb.org/browse/GODRIVER-444)
This document details a new design for passing optional parameters to Driver API functions and
methods. This document covers the motivation for the change, the high level design, and relevant
implementation details. Alternate designs that were ultimately rejected are covered in the first
appendix. This document refers to the options design for the Driver API specifically. References to
the Core API are to provide a basis for implementation.

## Definitions
<dl>
<dt>The Current Design</dt>
<dd>The implemented design that existed before this document was implemented.</dd>
<dt>The Proposed Design</dt>
<dd>The design that this document is proposing</dd>
<dt>Driver API</dt>
<dd>The high level driver library, implementing user facing specifications the <a
href="https://github.com/mongodb/specifications/blob/master/source/crud/crud.rst">CRUD
Specification</a>. This is mainly the <code>mongo</code> package.</dd>
<dt>Core API</dt>
<dd>The low level, verbose library, implementing specifications that are generally not user facing,
such as the <a
href="https://github.com/mongodb/specifications/blob/master/source/server-discovery-and-monitoring/server-discovery-and-monitoring.rst">Server
Discovery And Monitoring Specification</a></dd> </dl>

## Motivation
The current design for options uses [self referential
functions](https://commandcenter.blogspot.com/2014/01/self-referential-functions-and-design.html)
implemented in the Core API's `option` package and a constructor type in the `mongo` package for
creating these. The current design does not make it easy to discover which options are valid for
which methods. Additionally, there are limits in the ways users can bundle together options as they
are passed through an application.

The proposed design continues to use the Core API's `option` package for self referential options,
but replaces the constructor type with a series of namespaced packages. These packages effectively
operate as the constructor and enable a better discover and bundling experience. The proposed design
also has the benefit of allowing the core API to separate two options that have the same driver API
name but different command names. Finally, the proposed design ensures that users are not directly
exposed to the Core API's `option` package, which has caused confusion in the past.

The proposed design purposefully provides users with a number of ways to construct and bundle
options. This enables users to construct options differently as they learn to use the driver and as
the applications they build become more complex. For instance, when a user is writing a proof of
concept application and they aren't familiar with MongoDB, they might lean heavily on the bundle
types to provide the set of options available for a method or a constructor. Later on, that same
user might build another proof of concept application, but now knows which options are available for
methods and constructors, so they just the functions that are provided and skip the bundles. If the
user then builds a complex application with defaults, overrides, and configuration from various
sources, they'll find the flexibility of being able to use both the functions and bundles together
useful. For instance, they might combine a few bundles from different configuration sources, then
provide default options for specific functions in their application and in other places provide
overrides. For example, a user in one part of their application might want to by default perform
upserts when doing an update but might have a configuration flag that turns off this functionality.
In another case the user might want to limit the number of documents returned to 100 no matter what.
The proposed design provides the foundational types, functions, and methods to enable these usage
patterns.

## High Level Design
The proposed design uses namespace packages, and interfaces, types, and functions within those
packages. Each namespace package contains:
- an interface for each collection method that fall under the namespace
- functions for each option that is available
- a bundle to facilitate creating groups of options

To enable option discovery, the bundle types use a [builder style
API](https://en.wikipedia.org/wiki/Builder_pattern) for appending additional options.

The namespaces are arranged to group together options that are most similar to each other. For
instance, while findAndModify based operations are both read and write methods, they are placed in
the `findopt` package because of a larger overlap of their options with the find and findOne
operation options.

The namespace packages are:
- aggregateopt
- countopt
- distinctopt
- findopt
- insertopt
- deleteopt
- replaceopt
- updateopt
- bulkwriteopt
- clientopt
- collectionopt
- dbopt
- indexopt

### Interfaces
The interfaces for each package are named to align with the collection methods to which they will be
a parameter of. For example, the `insertopt.Many` interface is a parameter for the
`collection.InsertMany` method. For packages that serve a single type, the design is similar but the
naming may change. For example, to avoid import cycles, the `clientopt` package uses an interface
called Option, and the `clientopt.Client` type is a struct that is used by the `mongo.NewClient`
method to construct the `mongo.Client` type.

### Functions
Each namespace package will have a set of functions that return options for the interfaces contained
in that package. Each of these functions will return a type that satisfies the interfaces for which
the option is valid. For example, `insertopt.BypassDocumentValidation` returns a type that
satisfies both the `insertopt.Many` and `insertopt.One` interfaces.

### Bundles
For each interface in a package there is also a bundle. The purpose of the bundle, as described
above, is to enable creating a group of options and to provide a discoverable way to find options
that satisfy a given interface.

Each bundle is named [interface name]Bundle, so the bundle type for the `insertopt.Many` interface
is named `insertopt.ManyBundle`. For each of the bundle types there is a constructor function
called Bundle[interface name], with a variadic parameter of the interface and a return of the
[interface name]Bundle type. For example, the function signature for the `insertopt.BundleMany`
function would be:
```go
func BundleMany(...Many) ManyBundle
```

These bundle types also implement the interface they are associated with. For example, the
`insertopt.ManyBundle` type implements the `insertopt.Many` interface. This allows the bundle to
be passed directly to methods or functions and for bundles to be nested.

Each bundle with have methods that mirror the option functions in the package that return values
valid for the interface of the bundle. For instance, the `insertopt.BundleMany` type will have a
`BypassDocumentValidation` method. Finally, each bundle with have a method to flatten the bundle
into the options it was created from. This method is called Unbundle. The deduplicate parameter for
this method will only return the last instance of an option, even if multiple were specified. The
return value is a slice of options from the core API's `option` package. The primary purpose behind
the Unbundle method is to simplify the usage of these option namespaces within the mongo package.
The priority of the options is read from left to right, with options appearing to the left being
overridden by options on the right. This includes initial parameters to a bundle, forming a linked
list.

When a user creates a bundle, they'll do something like this:
```go
BundleFoo(Bar(), Baz(3), Qux(4)).BarBar().BazBaz().Qux(7).Baz(9)
```

The order those options are processed in is:
```go
Bar() -> Baz() -> Qux() -> BarBar() -> BazBaz() -> Qux() -> Baz()
```

When the deduplicate parameter is `false`, the return value would be:
```go
[]option.Option{OptBar{}, OptBaz{3}, OptQux{4}, OptBarBar{}, OptBazBaz{}, OptQux{7}, OptBaz{9}}
```

When the deduplicate parameter is `true`, the return value would be:
```go
[]option.Option{OptBar{}, OptBarBar{}, OptBazBaz{}, OptQux{7}, OptBaz{9}}
```

When deduplicate is true, the final order of the options is maintained and the intermediate options
are removed. In the above example, `OptBaz{3}` and `OptQux{4}` have been removed. Bundle
implementations do not attempt to modify the options, nor do they move the last option to the
position of the first option. The reason for this is enable a smoother debugging process. If we
moved options around it would not be possible to know the order that options were added to a bundle
or subbundle. Additionally, it means that the deduplication process only removes elements so that
if, in the future, it matters if options are in a specific order, the deduplication process does not
cause bugs by rearranging options. The deduplication process as a whole is necessary so that
simplifications can be made to the Core API's `option` package in the future. For example, the
`option.OptWriteConcern` type (at the time of the current design) has special searching logic to
ensure that if a write concern has already been added to a document that we do not add an
additional write concern. Using the bundle deduplication functionality, `option.OptWriteConcern`
no longer needs to have this additional logic.

When processing, a later option in the chain overrides the earlier option in the chain. In this
example the option values for Qux would be 7 and for Baz would be 9.

The way the bundles are designed pushes the complexity of unbundling into the options packages and
keeps the mongo package simpler. This allows the mongo package to internally use bundles as a method
to simplify implementation. For example:

```go
func (coll *Collection) Foo(ctx context.Context, opt ...foopt.Foo) error {
    optopt := fooopt.BundleFoo(opt...).Unbundle(true)
    for _, opt := range optopt {
        // Do something with the options...
    }
}
```

The method signature for an Unbundle method follows this pattern:
```go
func (fb *FooBundle) Unbundle(deduplicate bool) []option.Optioner
```

Operations that take both cursor options and their own options will return a slice of option.Option
types. Operations that only take their own options will return a slice of their own option package
type, for example the `findopt.ManyBundle.Unbundle` method will return `[]option.Optioner` while the
`findopt.DeleteBundle.Unbundle` method will return `[]option.FindOneAndDeleteOptioner`.

As a special case, the clientopt package does not follow this pattern, and instead returns a
`*clientopt.Client`. This is done because the clientopt package cannot import the mongo.Client
type, so instead of returning options that take a `mongo.Client`, we just return a
`*clientopt.Client` that the `mongo.NewClient` function can use to copy the final options to the
`mongo.Client` type.

#### Debugging
To enable debugging, the bundle types will implement `fmt.Stringer`, and print a structure the
represents how the bundle was constructed. The parameters to the functions or methods used will be
preserved and printed to the best degree possible.

An additional method called `StringDeduplicated` will do the same operation as the `fmt.Stringer`
but will first deduplicate the options.

## Code
The [example.go](example.go) contains a compilable playground of this design.

## Appendix 1: Rejected Designs
This section outlines some of the considered and rejected designs for driver API options.
### Options as struct types
An earlier design considered using structs for option types. While this is simpler than the proposed
design it has some difficult tradeoffs. Either the default case of no value requires a `nil` or
empty struct parameter, or the number of methods is doubled so there is a method that takes an
option struct and another one that does not. Neither of these cases provide a good API experience
for users. Additionally, structs require users to handle all the bundling logic themselves.

### Options as builder methods
In the process of creating this design, the team discussed using a builder style API for options.
For example, adding the `bypassDocumentValidation` option to the `collection.InsertMany` method
would look like this:
```go
func insertMany(ctx context.Context, coll *mongo.Collection, doc *bson.Document) {
    res, err := coll.InsertMany(ctx, doc).BypassDocumentValidation(true).Execute()
    // handle error and do something with the result
}
```
The main issue with this style of options is that a user could forget to call `Execute`, for
instance if they were doing an unacknowledged write and don't check for the error return. As a next
step it was suggested that we could move the parameters usually passed to the collection method to
the `Execute` method, it would be used like this:
```go
func insertMany(ctx context.Context, coll *mongo.Collection, doc *bson.Document) {
    res, err := coll.InsertMany().BypassDocumentValidation(true).Execute(ctx, doc)
    // handle error and do something with the result
}
```
While this method does prevent users from forgetting to call `Execute` it was decided that this
design deviated too much from the CRUD specification. Additionally, this design does not enable the
bundling functionality that the proposed design does.
