package aggregateopt

import (
	"github.com/mongodb/mongo-go-driver/options-design/option"
	"time"
)

type Aggregate interface {
	aggregate()
}

type AggregateBundle struct{}

func BundleAggregate(...Aggregate) *AggregateBundle{ return nil }

func (ab *AggregateBundle) AllowDiskUse(b bool) *AggregateBundle {
	return nil
}

func (ab *AggregateBundle) BatchSize(i int32) *AggregateBundle {
	return nil
}

func (ab *AggregateBundle) BypassDocumentValidation(b bool) *AggregateBundle {
	return nil
}

func (ab *AggregateBundle) Collation(c option.Collation) *AggregateBundle {
	return nil
}

func (ab *AggregateBundle) MaxTime(d time.Duration) *AggregateBundle {
	return nil
}

func (ab *AggregateBundle) Comment(s string) *AggregateBundle {
	return nil
}

func (ab *AggregateBundle) Hint(hint interface{}) *AggregateBundle {
	return nil
}

func (ab *AggregateBundle) Unbundle(deduplicate bool) []option.Optioner {
	return nil
}

func AllowDiskUse(b bool) OptAllowDiskUse { return false }
func BatchSize(i int32) OptBatchSize { return 0 }
func BypassDocumentValidation(b bool) OptBypassDocumentValidation { return false }
func Collation(c option.Collation) OptCollation { return OptCollation{} }
func MaxTime(d time.Duration) OptMaxTime { return 0 }
func Comment(s string) OptComment { return "" }
func Hint(hint interface{}) OptHint { return OptHint{} }

func (ab *AggregateBundle) aggregate() {}

type OptAllowDiskUse option.OptAllowDiskUse

func (OptAllowDiskUse) aggregate() {}

type OptBatchSize option.OptBatchSize

func (OptBatchSize) aggregate() {}

type OptBypassDocumentValidation option.OptBypassDocumentValidation

func (OptBypassDocumentValidation) aggregate() {}

type OptCollation option.OptCollation

func (OptCollation) aggregate() {}

type OptMaxTime option.OptMaxTime

func (OptMaxTime) aggregate() {}

type OptComment option.OptComment

func (OptComment) aggregate() {}

type OptHint option.OptHint

func (OptHint) aggregate() {}

