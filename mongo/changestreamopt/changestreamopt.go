package changestreamopt

import (
	"reflect"

	"time"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/mongo/mongoopt"
)

var csBundle = new(ChangeStreamBundle)

// ChangeStream represents all passable params for the changeStream() function.
type ChangeStream interface {
	changeStream()
}

// ChangeStreamOption represents the options for the changeStream() function.
type ChangeStreamOption interface {
	ChangeStream
	ConvertChangeStreamOption() option.ChangeStreamOptioner
}

// ChangeStreamSession is the session for the changeStream() function
type ChangeStreamSession interface {
	ChangeStream
	ConvertChangeStreamSession() *session.Client
}

// ChangeStreamBundle is a bundle of ChangeStream options
type ChangeStreamBundle struct {
	option ChangeStream
	next   *ChangeStreamBundle
}

func (csb *ChangeStreamBundle) changeStream() {}

// ConvertChangeStreamOption implements the ChangeStream interface.
func (csb *ChangeStreamBundle) ConvertChangeStreamOption() option.ChangeStreamOptioner {
	return nil
}

// BundleChangeStream bundles ChangeStream options.
func BundleChangeStream(opts ...ChangeStream) *ChangeStreamBundle {
	head := csBundle

	for _, opt := range opts {
		newBundle := ChangeStreamBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

// BatchSize specifies the number of documents to return in a batch.
func (csb *ChangeStreamBundle) BatchSize(i int32) *ChangeStreamBundle {
	bundle := &ChangeStreamBundle{
		option: BatchSize(i),
		next:   csb,
	}

	return bundle
}

// Collation adds an option to specify a collation.
func (csb *ChangeStreamBundle) Collation(c *mongoopt.Collation) *ChangeStreamBundle {
	bundle := &ChangeStreamBundle{
		option: Collation(c),
		next:   csb,
	}

	return bundle
}

// FullDocument specifies if a copy of the whole document should be returned.
func (csb *ChangeStreamBundle) FullDocument(fd mongoopt.FullDocument) *ChangeStreamBundle {
	bundle := &ChangeStreamBundle{
		option: FullDocument(fd),
		next:   csb,
	}

	return bundle
}

// MaxAwaitTime specifies the maximum amount of time for the server to wait on new documents.
func (csb *ChangeStreamBundle) MaxAwaitTime(d time.Duration) *ChangeStreamBundle {
	bundle := &ChangeStreamBundle{
		option: MaxAwaitTime(d),
		next:   csb,
	}

	return bundle
}

// ResumeAfter specifies whether the change stream should be resumed after stopping.
func (csb *ChangeStreamBundle) ResumeAfter(d *bson.Document) *ChangeStreamBundle {
	bundle := &ChangeStreamBundle{
		option: ResumeAfter(d),
		next:   csb,
	}

	return bundle
}

// Unbundle transforms a bundle into a slice of options, optionally deduplicating
func (csb *ChangeStreamBundle) Unbundle(deduplicate bool) ([]option.ChangeStreamOptioner, *session.Client, error) {

	options, sess, err := csb.unbundle()
	if err != nil {
		return nil, nil, err
	}

	if !deduplicate {
		return options, sess, nil
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

	return options, sess, nil
}

// Calculates the total length of a bundle, accounting for nested bundles.
func (csb *ChangeStreamBundle) bundleLength() int {
	if csb == nil {
		return 0
	}

	bundleLen := 0
	for ; csb != nil; csb = csb.next {
		if csb.option == nil {
			continue
		}
		if converted, ok := csb.option.(*ChangeStreamBundle); ok {
			// nested bundle
			bundleLen += converted.bundleLength()
			continue
		}

		if _, ok := csb.option.(ChangeStreamSessionOpt); !ok {
			bundleLen++
		}
	}

	return bundleLen
}

// Helper that recursively unwraps bundle into slice of options
func (csb *ChangeStreamBundle) unbundle() ([]option.ChangeStreamOptioner, *session.Client, error) {
	if csb == nil {
		return nil, nil, nil
	}

	var sess *session.Client
	listLen := csb.bundleLength()

	options := make([]option.ChangeStreamOptioner, listLen)
	index := listLen - 1

	for listHead := csb; listHead != nil; listHead = listHead.next {
		if listHead.option == nil {
			continue
		}

		// if the current option is a nested bundle, Unbundle it and add its options to the current array
		if converted, ok := listHead.option.(*ChangeStreamBundle); ok {
			nestedOptions, s, err := converted.unbundle()
			if err != nil {
				return nil, nil, err
			}
			if s != nil && sess == nil {
				sess = s
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

		switch t := listHead.option.(type) {
		case ChangeStreamOption:
			options[index] = t.ConvertChangeStreamOption()
			index--
		case ChangeStreamSession:
			if sess == nil {
				sess = t.ConvertChangeStreamSession()
			}
		}
	}

	return options, sess, nil
}

// String implements the Stringer interface
func (csb *ChangeStreamBundle) String() string {
	if csb == nil {
		return ""
	}

	str := ""
	for head := csb; head != nil && head.option != nil; head = head.next {
		if converted, ok := head.option.(*ChangeStreamBundle); ok {
			str += converted.String()
			continue
		}

		if conv, ok := head.option.(ChangeStreamOption); !ok {
			str += conv.ConvertChangeStreamOption().String() + "\n"
		}
	}

	return str
}

// BatchSize specifies the number of documents to return in each batch.
func BatchSize(i int32) OptBatchSize {
	return OptBatchSize(i)
}

// Collation specifies a collation.
func Collation(c *mongoopt.Collation) OptCollation {
	return OptCollation{Collation: c.Convert()}
}

// FullDocument specifies whether a copy of the whole document should be returned.
func FullDocument(fd mongoopt.FullDocument) OptFullDocument {
	return OptFullDocument(fd)
}

// MaxAwaitTime specifies the max amount of time for the server to wait on new documents.
func MaxAwaitTime(d time.Duration) OptMaxAwaitTime {
	return OptMaxAwaitTime(d)
}

// ResumeAfter specifies if a change stream should be resumed after stopping.
func ResumeAfter(d *bson.Document) OptResumeAfter {
	return OptResumeAfter{
		ResumeAfter: d,
	}
}

// OptBatchSize specifies the number of documents to return in each batch.
type OptBatchSize option.OptBatchSize

func (OptBatchSize) changeStream() {}

// ConvertChangeStreamOption implements the ChangeStream interface
func (opt OptBatchSize) ConvertChangeStreamOption() option.ChangeStreamOptioner {
	return option.OptBatchSize(opt)
}

// OptCollation specifies a collation.
type OptCollation option.OptCollation

func (OptCollation) changeStream() {}

// ConvertChangeStreamOption implements the ChangeStream interface.
func (opt OptCollation) ConvertChangeStreamOption() option.ChangeStreamOptioner {
	return option.OptCollation(opt)
}

// OptFullDocument specifies whether a copy of the whole document should be returned.
type OptFullDocument option.OptFullDocument

func (OptFullDocument) changeStream() {}

// ConvertChangeStreamOption implements the ChangeStream interface.
func (opt OptFullDocument) ConvertChangeStreamOption() option.ChangeStreamOptioner {
	return option.OptFullDocument(opt)
}

// OptMaxAwaitTime specifies the max amount of time for the server to wait on new documents.
type OptMaxAwaitTime option.OptMaxAwaitTime

func (OptMaxAwaitTime) changeStream() {}

// ConvertChangeStreamOption implements the ChangeStream interface.
func (opt OptMaxAwaitTime) ConvertChangeStreamOption() option.ChangeStreamOptioner {
	return option.OptMaxAwaitTime(opt)
}

// OptResumeAfter specifies if the stream should be resumed after stopping.
type OptResumeAfter option.OptResumeAfter

func (OptResumeAfter) changeStream() {}

// ConvertChangeStreamOption implements the ChangeStream interface.
func (opt OptResumeAfter) ConvertChangeStreamOption() option.ChangeStreamOptioner {
	return option.OptResumeAfter(opt)
}

// ChangeStreamSessionOpt is an count session option.
type ChangeStreamSessionOpt struct{}

func (ChangeStreamSessionOpt) changeStream() {}

// ConvertChangeStreamSession implements the ChangeStreamSession interface.
func (ChangeStreamSessionOpt) ConvertChangeStreamSession() *session.Client {
	return nil
}
