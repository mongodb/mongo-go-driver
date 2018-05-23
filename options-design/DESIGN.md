# Options Design
The design for options uses self referential functions and a builder in the mongo package for
creating these. The main issue with the current structure is that it is difficult to discover which
options are valid for what methods. This method also limits the way in which users can bundle
together options that are passed through their application. The design laid out in this document
attempts to solve these discoverability and bundling issues while ensuring that the useful self
referential functions are kept intact. This design ensures that the core/options package is not
exposed from the mongo package’s API, which has caused confusion for users in the past.

This design uses packages for namespacing, and interfaces, types, and functions within those
namespaces. Each namespace package contains an interface for each collection method that fall under
the namespace, functions for each option that is available, and a bundle to facilitate creating
groups of options. Since the bundle uses a fluent API for appending additional options, it can also
be used for discovery of available options.

The namespaces are arranged to group together options that are most similar to each other. For
instance, while findAndModify based operations are both read and write methods, they are placed in
the findopts package because of a larger overlap of their options with the find and findOne
operation options. The packages are aggregateopts, countopts, distinctopts, findopts, insertopts,
deleteopts, replaceopts, updateopts, bulkwriteopts, clientopts, collectionopts, databaseopts, and
indexopts. The interfaces for each package are named to align with the collection methods they are
meant to be a parameter of, for example, the insertopts.Many interface is meant to be a parameter of
the collection.InsertMany method. This is slightly different for non-collection method options, such
as clientopts, where the interface is for the type. Each of the functions will return a type that
satisfies the interfaces for which it is a valid options, for example,
insertopts.BypassDocumentValidation would return a type that satisfies both the insertopts.Many and
insertopts.One interfaces. Finally, each package has a bundle for each interface. The purpose of the
bundle is to allow creating bundles of options and to provide a discoverable way to select options
for a given method. Each package has a Bundle\* function that takes a variadic number of the
interface for that bundle and returns a \*Bundle which implements the interface and also
contains methods, each of which is an option that satisfies that interface. Each \*Bundle type
has an Unbundle method that will flatten the bundle into the options it was created from. The
Unbundle method has a deduplicate parameter that will only return the last instance of each
option that was set. Since each \*Bundle type implements the interface, they can be nested.

The reason for having an Unbundle method is that when a user uses a bundle, the priority of the
options should be read from left to right, including the initial parameters and because each of the
bundle types will be a linked list. When a user creates a bundle they’ll doing something like this: 
BundleFoo(Bar(), Baz(3), Qux(4)).BarBar().BazBaz().Qux(7).Baz(9)

The order those options are processed in needs to be Bar() -> Baz() -> Qux() -> BarBar() -> BazBaz()
-> Qux() -> Baz(), with a later thing in the chain overriding the earlier thing in the chain. In
this example the option values for Qux would be 7 and for Baz would be 9.

Internally, the mongo package will use Bundles to construct the slice of options that will
ultimately be used. The way the bundles are designed makes pushes the complexity of unbundling into
the options packages and keeps the mongo package simpler.

## Code
[findopts.go](findopts.go)

[example.go](example.go)
