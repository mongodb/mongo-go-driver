package aggregateopt

import (
	"time"

	"reflect"

	"github.com/mongodb/mongo-go-driver/core/option"
)

var aggregateBundle = new(AggregateBundle)

// Aggregate is options for the aggregate() function
type Aggregate interface {
	aggregate()
	ConvertOption() option.Optioner
}

// AggregateBundle is a bundle of Aggregate options
type AggregateBundle struct {
	option Aggregate
	next   *AggregateBundle
}

// Implement the Aggregate interface
func (ab *AggregateBundle) aggregate() {}

// ConvertOption implements the Aggregate interface
func (ab *AggregateBundle) ConvertOption() option.Optioner { return nil }

// BundleAggregate bundles Aggregate options
func BundleAggregate(opts ...Aggregate) *AggregateBundle {
	head := aggregateBundle

	for _, opt := range opts {
		newBundle := AggregateBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

// AllowDiskUse specifies the AllowDiskUse option
func (ab *AggregateBundle) AllowDiskUse(b bool) *AggregateBundle {
	bundle := &AggregateBundle{
		option: OptAllowDiskUse(b),
		next:   ab,
	}

	return bundle
}

// BatchSize specifies the BatchSize option
func (ab *AggregateBundle) BatchSize(i int32) *AggregateBundle {
	bundle := &AggregateBundle{
		option: OptBatchSize(i),
		next:   ab,
	}

	return bundle
}

// BypassDocumentValidation specifies the BypassDocumentValidation option
func (ab *AggregateBundle) BypassDocumentValidation(b bool) *AggregateBundle {
	bundle := &AggregateBundle{
		option: OptBypassDocumentValidation(b),
		next:   ab,
	}

	return bundle
}

//Collation specifies the Collation option
func (ab *AggregateBundle) Collation(c option.Collation) *AggregateBundle {
	bundle := &AggregateBundle{
		option: OptCollation{Collation: &c},
		next:   ab,
	}

	return bundle
}

// MaxTime specifies the MaxTime option
func (ab *AggregateBundle) MaxTime(d time.Duration) *AggregateBundle {
	bundle := &AggregateBundle{
		option: OptMaxTime(d),
		next:   ab,
	}

	return bundle
}

// Comment specifies the Comment option
func (ab *AggregateBundle) Comment(s string) *AggregateBundle {
	bundle := &AggregateBundle{
		option: OptComment(s),
		next:   ab,
	}

	return bundle
}

// Hint specifies the Hint option
func (ab *AggregateBundle) Hint(hint interface{}) *AggregateBundle {
	bundle := &AggregateBundle{
		option: OptHint{hint},
		next:   ab,
	}

	return bundle
}

func bundleLength(ab *AggregateBundle) int {
	if ab == nil {
		return 0
	}

	bundleLen := 0
	for ; ab != nil && ab.option != nil; ab = ab.next {
		if converted, ok := ab.option.(*AggregateBundle); ok {
			// nested bundle
			bundleLen += bundleLength(converted)
			continue
		}

		bundleLen++
	}

	return bundleLen
}

// Unbundle transforms a bundle into a slice of options
func (ab *AggregateBundle) Unbundle(deduplicate bool) ([]option.Optioner, error) {
	if ab == nil {
		return nil, nil
	}

	listLen := bundleLength(ab)

	options := make([]option.Optioner, listLen)
	index := listLen - 1

	for listHead := ab; listHead != nil && listHead.option != nil; listHead = listHead.next {
		// check for nested bundles
		// if the current option is a nested bundle, Unbundle it and add its options to the current array
		if converted, ok := listHead.option.(*AggregateBundle); ok {
			nestedOptions, err := converted.Unbundle(deduplicate)
			if err != nil {
				return nil, err
			}

			startIndex := index - len(nestedOptions) + 1 // where to start inserting nested options

			// add nested options in order
			for _, nestedOp := range nestedOptions {
				options[startIndex] = nestedOp
				startIndex++
			}
			index -= len(nestedOptions)
			continue
		}

		options[index] = listHead.option.ConvertOption()
		index--
	}

	if !deduplicate {
		return options, nil
	}

	// iterate backwards and make dedup slice
	optionsSet := make(map[reflect.Type]struct{})

	for i := listLen - 1; i >= 0; i-- {
		currOption := options[i]
		optionType := reflect.TypeOf(currOption)

		if _, ok := optionsSet[optionType]; ok {
			// option already found
			options = append(options[:i], options[i+1:]...)
			continue
		}

		optionsSet[optionType] = struct{}{}
	}

	return options, nil
}

// AllowDiskUse specifies the AllowDiskUse option
func AllowDiskUse(b bool) OptAllowDiskUse {
	return OptAllowDiskUse(b)
}

// BatchSize specifies the BatchSize option
func BatchSize(i int32) OptBatchSize {
	return OptBatchSize(i)
}

// BypassDocumentValidation specifies the BypassDocumentValidation option
func BypassDocumentValidation(b bool) OptBypassDocumentValidation {
	return OptBypassDocumentValidation(b)
}

// Collation specifies the Collation option
func Collation(c option.Collation) OptCollation {
	return OptCollation{Collation: &c}
}

// MaxTime specifies the MaxTime option
func MaxTime(d time.Duration) OptMaxTime {
	return OptMaxTime(d)
}

// Comment specifies the Comment option
func Comment(s string) OptComment {
	return OptComment(s)
}

// Hint specifies the Hint option
func Hint(hint interface{}) OptHint {
	return OptHint{hint}
}

// OptAllowDiskUse is for internal use
type OptAllowDiskUse option.OptAllowDiskUse

func (OptAllowDiskUse) aggregate() {}

// ConvertOption implements the Aggregate interface
func (opt OptAllowDiskUse) ConvertOption() option.Optioner {
	return option.OptAllowDiskUse(opt)
}

// OptBatchSize is for internal use
type OptBatchSize option.OptBatchSize

func (OptBatchSize) aggregate() {}

// ConvertOption implements the Aggregate interface
func (opt OptBatchSize) ConvertOption() option.Optioner {
	return option.OptBatchSize(opt)
}

// OptBypassDocumentValidation is for internal use
type OptBypassDocumentValidation option.OptBypassDocumentValidation

// ConvertOption implements the Aggregate interface
func (opt OptBypassDocumentValidation) ConvertOption() option.Optioner {
	return option.OptBypassDocumentValidation(opt)
}

func (OptBypassDocumentValidation) aggregate() {}

// OptCollation is for internal use
type OptCollation option.OptCollation

func (OptCollation) aggregate() {}

// ConvertOption implements the Aggregate interface
func (opt OptCollation) ConvertOption() option.Optioner {
	return option.OptCollation(opt)
}

// OptMaxTime is for internal use
type OptMaxTime option.OptMaxTime

func (OptMaxTime) aggregate() {}

// ConvertOption implements the Aggregate interface
func (opt OptMaxTime) ConvertOption() option.Optioner {
	return option.OptMaxTime(opt)
}

// OptComment is for internal use
type OptComment option.OptComment

func (OptComment) aggregate() {}

// ConvertOption implements the Aggregate interface
func (opt OptComment) ConvertOption() option.Optioner {
	return option.OptComment(opt)
}

// OptHint is for internal use
type OptHint option.OptHint

func (OptHint) aggregate() {}

// ConvertOption implements the Aggregate interface
func (opt OptHint) ConvertOption() option.Optioner {
	return option.OptHint(opt)
}
