package countopt

import (
	"reflect"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/mongo/mongoopt"
)

var countBundle = new(CountBundle)

// Count is options for the count() function
type Count interface {
	count()
	ConvertCountOption() option.CountOptioner
}

// CountBundle is a bundle of Count options
type CountBundle struct {
	option Count
	next   *CountBundle
}

// Implement the Count interface
func (cb *CountBundle) count() {}

// ConvertCountOption implements the Count interface
func (cb *CountBundle) ConvertCountOption() option.CountOptioner {
	return nil
}

// BundleCount bundles Count options
func BundleCount(opts ...Count) *CountBundle {
	head := countBundle

	for _, opt := range opts {
		newBundle := CountBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

// Collation specifies a collation.
func (cb *CountBundle) Collation(c *mongoopt.Collation) *CountBundle {
	bundle := &CountBundle{
		option: Collation(c),
		next:   cb,
	}

	return bundle
}

// Limit adds an option to limit the maximum number of documents to count.
func (cb *CountBundle) Limit(i int64) *CountBundle {
	bundle := &CountBundle{
		option: Limit(i),
		next:   cb,
	}

	return bundle
}

// Skip adds an option to specify the number of documents to skip before counting.
func (cb *CountBundle) Skip(i int64) *CountBundle {
	bundle := &CountBundle{
		option: Skip(i),
		next:   cb,
	}

	return bundle
}

// Hint adds an option to specify the index to use.
func (cb *CountBundle) Hint(hint interface{}) *CountBundle {
	bundle := &CountBundle{
		option: Hint(hint),
		next:   cb,
	}

	return bundle
}

// MaxTimeMs adds an option to specify the maximum amount of time to allow the operation to run.
func (cb *CountBundle) MaxTimeMs(i int32) *CountBundle {
	bundle := &CountBundle{
		option: MaxTimeMs(i),
		next:   cb,
	}

	return bundle
}

// Unbundle transforms a bundle into a slice of options, optionally deduplicating.
func (cb *CountBundle) Unbundle(deduplicate bool) ([]option.CountOptioner, error) {
	options, err := cb.unbundle()
	if err != nil {
		return nil, err
	}

	if !deduplicate {
		return options, nil
	}

	// iterate backwards and make dedup slice
	optionsSet := make(map[reflect.Type]struct{})

	for i := len(options) - 1; i >= 0; i-- {
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

// Calculates the total length of a bundle, accounting for nested bundles.
func (cb *CountBundle) bundleLength() int {
	if cb == nil {
		return 0
	}

	bundleLen := 0
	for ; cb != nil && cb.option != nil; cb = cb.next {
		if converted, ok := cb.option.(*CountBundle); ok {
			// nested bundle
			bundleLen += converted.bundleLength()
			continue
		}

		bundleLen++
	}

	return bundleLen
}

// Helper that recursively unwraps bundle into slice of options
func (cb *CountBundle) unbundle() ([]option.CountOptioner, error) {
	if cb == nil {
		return nil, nil
	}

	listLen := cb.bundleLength()

	options := make([]option.CountOptioner, listLen)
	index := listLen - 1

	for listHead := cb; listHead != nil && listHead.option != nil; listHead = listHead.next {
		// if the current option is a nested bundle, Unbundle it and add its options to the current array
		if converted, ok := listHead.option.(*CountBundle); ok {
			nestedOptions, err := converted.unbundle()
			if err != nil {
				return nil, err
			}

			// where to start inserting nested options
			startIndex := index - len(nestedOptions) + 1

			// add nested options in order
			for _, nestedOp := range nestedOptions {
				options[startIndex] = nestedOp
				startIndex++
			}
			index -= len(nestedOptions)
			continue
		}

		options[index] = listHead.option.ConvertCountOption()
		index--
	}

	return options, nil
}

// String implements the Stringer interface
func (cb *CountBundle) String() string {
	if cb == nil {
		return ""
	}

	str := ""
	for head := cb; head != nil && head.option != nil; head = head.next {
		if converted, ok := head.option.(*CountBundle); ok {
			str += converted.String()
			continue
		}

		str += head.option.ConvertCountOption().String()
	}

	return str
}

// Collation specifies a Collation.
func Collation(collation *mongoopt.Collation) OptCollation {
	return OptCollation{
		Collation: collation.Convert(),
	}
}

// Limit limits the maximum number of documents to count.
func Limit(i int64) OptLimit {
	return OptLimit(i)
}

// Skip specifies the number of documents to skip before counting.
func Skip(i int64) OptSkip {
	return OptSkip(i)
}

// Hint specifies the index to use.
func Hint(hint interface{}) OptHint {
	return OptHint{hint}
}

// MaxTimeMs specifies the maximum amount of time to allow the operation to run.
func MaxTimeMs(i int32) OptMaxTimeMs {
	return OptMaxTimeMs(i)
}

// OptCollation specifies a collation.
type OptCollation option.OptCollation

func (OptCollation) count() {}

// ConvertCountOption implements the Count interface.
func (opt OptCollation) ConvertCountOption() option.CountOptioner {
	return option.OptCollation(opt)
}

// OptLimit limits the maximum number of documents to count.
type OptLimit option.OptLimit

// ConvertCountOption implements the Count interface.
func (opt OptLimit) ConvertCountOption() option.CountOptioner {
	return option.OptLimit(opt)
}

func (OptLimit) count() {}

// OptSkip specifies the number of documents to skip before counting.
type OptSkip option.OptSkip

// ConvertCountOption implements the Count interface.
func (opt OptSkip) ConvertCountOption() option.CountOptioner {
	return option.OptSkip(opt)
}

func (OptSkip) count() {}

// OptHint specifies the index to use.
type OptHint option.OptHint

// ConvertCountOption implements the Count interface.
func (opt OptHint) ConvertCountOption() option.CountOptioner {
	return option.OptHint(opt)
}

func (OptHint) count() {}

// OptMaxTimeMs specifies the maximum amount of time to allow the operation to run.
type OptMaxTimeMs option.OptMaxTime

// ConvertCountOption implements the Count interface.
func (opt OptMaxTimeMs) ConvertCountOption() option.CountOptioner {
	return option.OptMaxTime(opt)
}

func (OptMaxTimeMs) count() {}
