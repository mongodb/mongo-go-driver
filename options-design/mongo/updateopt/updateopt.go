package updateopt

import (
	"github.com/mongodb/mongo-go-driver/options-design/bson"
	"github.com/mongodb/mongo-go-driver/options-design/mongo/mongoopt"
	"github.com/mongodb/mongo-go-driver/options-design/option"
)

type Update interface {
	update()
}

type UpdateBundle struct{}

func BundleUpdate(...Update) *UpdateBundle {
	return nil
}

func (ub *UpdateBundle) update()                                       {}
func (ub *UpdateBundle) ArrayFilters(b []*bson.Document) *UpdateBundle { return nil }
func (ub *UpdateBundle) BypassDocumentValidation(b bool) *UpdateBundle { return nil }
func (ub *UpdateBundle) Collation(c *mongoopt.Collation) *UpdateBundle { return nil }
func (ub *UpdateBundle) Upsert(b bool) *UpdateBundle                   { return nil }
func (ub *UpdateBundle) Unbundle() []option.UpdateOptioner             { return nil }

func ArrayFilters(b []*bson.Document) OptArrayFilters             { return nil }
func BypassDocumentValidation(b bool) OptBypassDocumentValidation { return false }
func Collation(c *mongoopt.Collation) OptCollation                { return OptCollation{} }
func Upsert(b bool) OptUpsert                                     { return false }

type OptArrayFilters option.OptArrayFilters

func (OptArrayFilters) update() {}

type OptBypassDocumentValidation option.OptBypassDocumentValidation

func (OptBypassDocumentValidation) update() {}

type OptCollation option.OptCollation

func (OptCollation) update() {}

type OptUpsert option.OptUpsert

func (OptUpsert) update() {}
