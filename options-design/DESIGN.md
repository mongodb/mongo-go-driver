# Options Design
The design for options uses [self referential
functions](https://commandcenter.blogspot.com/2014/01/self-referential-functions-and-design.html)
and a constructor type in the mongo package for creating these. This design does not make it easy to
discover which options are valid for which methods. Additionally, this methods limits the ways in
which users can bundle together options as they are passed through their application. This design
builds on top of the self referential options design while adopting a new design for the options
constructor that enables a better discovery and bundling experience. This design also allows the
core API to separate two options that have the same driver API name but different command names.
Finally, this design ensures that users are not directly exposed to the core/options package, which
has caused confusion in the past.


## High Level Design
This design uses namespace packages, and interfaces, types, and functions within those
namespaces. Each namespace package contains:
- an interface for each collection method that fall under the namespace
- functions for each option that is available
- a bundle to facilitate creating groups of options

To enable option discovery, the bundle types use a [builder style
API](https://en.wikipedia.org/wiki/Builder_pattern) for appending additional options.

The namespaces are arranged to group together options that are most similar to each other. For
instance, while findAndModify based operations are both read and write methods, they are placed in
the findopts package because of a larger overlap of their options with the find and findOne
operation options.

The namespace packages are:
- aggregateopts
- countopts
- distinctopts
- findopts
- insertopts
- deleteopts
- replaceopts
- updateopts
- bulkwriteopts
- clientopts
- collectionopts
- databaseopts
- indexopts

### Interfaces
The interfaces for each package are named to align with the collection methods to which they will be
a parameter of. For example, the insertopts.Many interface is a parameter for the
collection.InsertMany method. For packages that serve a single type, the design is similar but the
naming may change. For example, to avoid import cycles, the clientopts package uses an interface
called Option , and the clientopts.Client type is a struct that is used by the mongo.NewClient
method to construct the mongo.Client type.

### Functions
Each namespace package will have a set of functions that return options for the interfaces contained
in that package. Each of these functions will return a type that satisfies the interfaces for which
the option is valid. For example,
insertopts.BypassDocumentValidation returns a type that satisfies both the insertopts.Many and
insertopts.One interfaces.

### Bundles
For each interface in a package there is also a bundle. The purpose of the bundle, as described
above, is to enable creating a group of options and to provide a discoverable way to find options
that satisfy a given interface.

Each bundle is named [interface name]Bundle, so the bundle type for the `insertopts.Many` interface
is named `insertopts.ManyBundle`. For each of the bundle types there is a constructor function
called Bundle[interface name], with a variadic parameter of the interface and a return of the
[interface name]Bundle type. For example, the function signature for the `insertopts.BundleMany`
function would be:
```
func BundleMany(...Many) ManyBundle
```

These bundle types also implement the interface they are associated with. This allows the bundle to
be passed directly to methods or functions and for bundles to be nested arbitrarily deep. Each
bundle will have methods attached, each of which mirrors an option function in the package that is a
valid option for that interface. Finally, each bundle will have an Unbundle method. The purpose of
this method is to flatten the bundle into the options it was created from. The Unbundle method has a
deduplicate parameter that will cause the method to only return the last instance of each option
that was set. The primary purpose behind the Unbundle method is to simplify the usage of these
option namespaces within the mongo package. The priority of the options is read from left to right,
with options appearing to the left being overridden by options on the right. This includes initial
parameters to a bundle, ultimately forming a linked list.

When a user creates a bundle, they'll do something like this:
```
BundleFoo(Bar(), Baz(3), Qux(4)).BarBar().BazBaz().Qux(7).Baz(9)
```

The order those options are processed in is:
```
Bar() -> Baz() -> Qux() -> BarBar() -> BazBaz() -> Qux() -> Baz()
```

A later option in the chain overrides the earlier option in the chain. In this example the option
values for Qux would be 7 and for Baz would be 9.

The mongo package will use Bundles to construct the slice of options that will
be used. The way the bundles are designed pushes the complexity of unbundling into
the options packages and keeps the mongo package simpler.

The method signature for an Unbundle method follows this pattern:
```
func (fb *FooBundle) Unbundle(deduplicate bool) []option.Optioner
```

Operations that take both cursor options and their own options will return a slice of option.Option
types. Operations that only take their own options will return a slice of their own option package
type, for example the `findopts.ManyBundle.Unbundle` method will return `[]option.Optioner` while the
`findopts.DeleteBundle.Unbundle` method will return `[]option.FindOneAndDeleteOptioner`.

As a special case, the clientopts package does not follow this pattern, and instead returns a
`\*clientopts.Client`.

#### Debugging
To enable debugging, the bundle types will implement `fmt.Stringer`, and print a structure the
represents how the bundle was constructed. The parameters to the functions or methods used will be
preserved and printed to the best degree possible.

## Implementation Details


## Code
The [example.go](example.go) contains a compilable playground of this design.
