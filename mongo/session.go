package mongo

import (
	"github.com/mongodb/mongo-go-driver/mongo/aggregateopt"
	"github.com/mongodb/mongo-go-driver/mongo/countopt"
	"github.com/mongodb/mongo-go-driver/mongo/deleteopt"
	"github.com/mongodb/mongo-go-driver/mongo/distinctopt"
	"github.com/mongodb/mongo-go-driver/mongo/findopt"
	"github.com/mongodb/mongo-go-driver/mongo/insertopt"
	"github.com/mongodb/mongo-go-driver/mongo/replaceopt"
	"github.com/mongodb/mongo-go-driver/mongo/updateopt"
)

// Session represents a set of sequential operations executed by an application that are related in some way.
type Session struct {
	aggregateopt.AggregateSessionOpt
	countopt.CountSessionOpt
	deleteopt.DeleteSessionOpt
	distinctopt.DistinctSessionOpt
	findopt.FindSessionOpt
	findopt.FindOneSessionOpt
	findopt.DeleteOneSessionOpt
	findopt.ReplaceOneSessionOpt
	findopt.UpdateOneSessionOpt
	updateopt.UpdateSessionOpt
	replaceopt.ReplaceSessionOpt
	insertopt.InsertSessionOpt
	// TODO also include core session
}

var (
	_ aggregateopt.Aggregate = (*Session)(nil)
	_ countopt.Count         = (*Session)(nil)
	_ deleteopt.Delete       = (*Session)(nil)
	_ distinctopt.Distinct   = (*Session)(nil)
	_ findopt.Find           = (*Session)(nil)
	_ findopt.One            = (*Session)(nil)
	_ findopt.UpdateOne      = (*Session)(nil)
	_ findopt.ReplaceOne     = (*Session)(nil)
	_ findopt.DeleteOne      = (*Session)(nil)
	_ insertopt.Many         = (*Session)(nil)
	_ insertopt.One          = (*Session)(nil)
	_ updateopt.Update       = (*Session)(nil)
	_ replaceopt.Replace     = (*Session)(nil)
)
