package deleteopt

import (
	"github.com/mongodb/mongo-go-driver/options-design/mongo/mongoopt"
	"github.com/mongodb/mongo-go-driver/options-design/option"
)

type Delete interface{ delete() }

type DeleteBundle struct{}

func BundleDelete(...Delete) *DeleteBundle { return nil }

func (db *DeleteBundle) delete() {}

func (db *DeleteBundle) Collation(c *mongoopt.Collation) *DeleteBundle { return nil }

func Collation(c *mongoopt.Collation) OptCollation { return OptCollation{} }

type OptCollation option.OptCollation

func (OptCollation) delete() {}
