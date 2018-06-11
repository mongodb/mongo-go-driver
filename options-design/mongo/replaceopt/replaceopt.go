package replaceopt

import (
	"github.com/mongodb/mongo-go-driver/options-design/mongo/mongoopt"
	"github.com/mongodb/mongo-go-driver/options-design/option"
)

type Replace interface{ replace() }

type ReplaceBundle struct{}

func BundleReplace(...Replace) *ReplaceBundle { return nil }

func (rb *ReplaceBundle) replace()                                       {}
func (rb *ReplaceBundle) BypassDocumentValidation(b bool) *ReplaceBundle { return nil }
func (rb *ReplaceBundle) Collation(c *mongoopt.Collation) *ReplaceBundle { return nil }
func (rb *ReplaceBundle) Upsert(b bool) *ReplaceBundle                   { return nil }
func (rb *ReplaceBundle) Unbundle() []option.ReplaceOptioner             { return nil }

func BypassDocumentValidation(b bool) OptBypassDocumentValidation { return false }
func Collation(c *mongoopt.Collation) OptCollation                { return OptCollation{} }
func Upsert(b bool) OptUpsert                                     { return false }

type OptBypassDocumentValidation option.OptBypassDocumentValidation

func (OptBypassDocumentValidation) replace() {}

type OptCollation option.OptCollation

func (OptCollation) replace() {}

type OptUpsert option.OptUpsert

func (OptUpsert) replace() {}
