package yamgo

import "github.com/10gen/mongo-go-driver/yamgo/options"

// In order to facilitate users not having to supply a default value (e.g. nil or an uninitialized
// struct) when not using any options for an operation, options are defined as functions that take
// the necessary state and return a private type which implements an interface for the given
// operation. This API will allow users to use the following syntax for operations:
//
// coll.UpdateOne(filter, update)
// coll.UpdateOne(filter, update, yamgo.Upsert(true))
// coll.UpdateOne(filter, update, yamgo.Upsert(true), yamgo.bypassDocumentValidation(false))
//
// To see which options are available on each operation, see the following files:
//
// - delete_options.go
// - insert_options.go
// - update_options.go

// ArrayFilters specifies to which array elements an update should apply.
func ArrayFilters(filters ...interface{}) *options.OptArrayFilters {
	opt := options.OptArrayFilters(filters)
	return &opt
}

// BypassDocumentValidation is used to opt out of document-level validation for a given write.
func BypassDocumentValidation(b bool) *options.OptBypassDocumentValidation {
	opt := options.OptBypassDocumentValidation(b)
	return &opt
}

// Collation allows users to specify language-specific rules for string comparison, such as rules
// for lettercase and accent marks.
//
// See https://docs.mongodb.com/manual/reference/collation/.
func Collation(c *options.CollationOptions) *options.OptCollation {
	opt := options.OptCollation{Collation: c}
	return &opt
}

// Ordered specifies whether the remaining writes should be aborted if one of the earlier ones fails.
func Ordered(b bool) *options.OptOrdered {
	opt := options.OptOrdered(b)
	return &opt
}

// Upsert specifies that a new document should be inserted if no document matches the update
// filter.
func Upsert(b bool) *options.OptUpsert {
	opt := options.OptUpsert(b)
	return &opt
}
