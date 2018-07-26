package gridfs

import (
	"reflect"

	"time"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
	"github.com/mongodb/mongo-go-driver/mongo/findopt"
)

var bucketBundle = new(BucketBundle)
var uploadBundle = new(UploadBundle)
var findBundle = new(FindBundle)
var nameBundle = new(NameBundle)

// BucketOptioner represents all options that can be used to configure a GridFS bucket.
type BucketOptioner interface {
	bucketOption(*Bucket) error
}

// UploadOptioner represents all options that can be used to configure a file upload.
type UploadOptioner interface {
	uploadOption(*Upload) error
}

// FindOptioner represents all options that can be used to configure a find command.
type FindOptioner interface {
	convertFindOption() findopt.Find
}

// NameOptioner represents all options that can be used for a download by name command.
type NameOptioner interface {
	nameOption()
}

// BucketBundle is a bundle of BucketOptioner options
type BucketBundle struct {
	option BucketOptioner
	next   *BucketBundle
}

// bucketOption implements the BucketOptioner interface
func (*BucketBundle) bucketOption(*Bucket) error {
	return nil
}

// BundleBucket bundles BucketOptioner options.
func BundleBucket(opts ...BucketOptioner) *BucketBundle {
	head := bucketBundle

	for _, opt := range opts {
		newBundle := BucketBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

// BucketName specifies the name of the bucket. Defaults to 'fs' if not specified.
func (bb *BucketBundle) BucketName(name string) *BucketBundle {
	bundle := &BucketBundle{
		option: BucketName(name),
		next:   bb,
	}

	return bundle
}

// ChunkSizeBytes specifies the chunk size for the bucket. Defaults to 255KB if not specified.
func (bb *BucketBundle) ChunkSizeBytes(b int32) *BucketBundle {
	bundle := &BucketBundle{
		option: ChunkSizeBytes(b),
		next:   bb,
	}

	return bundle
}

// WriteConcern specifies the write concern for the bucket.
func (bb *BucketBundle) WriteConcern(wc *writeconcern.WriteConcern) *BucketBundle {
	bundle := &BucketBundle{
		option: WriteConcern(wc),
		next:   bb,
	}

	return bundle
}

// ReadConcern specifies the read concern for the bucket.
func (bb *BucketBundle) ReadConcern(rc *readconcern.ReadConcern) *BucketBundle {
	bundle := &BucketBundle{
		option: ReadConcern(rc),
		next:   bb,
	}

	return bundle
}

// ReadPreference specifies the read preference for the bucket.
func (bb *BucketBundle) ReadPreference(rp *readpref.ReadPref) *BucketBundle {
	bundle := &BucketBundle{
		option: ReadPreference(rp),
		next:   bb,
	}

	return bundle
}

// Unbundle transforms a bundle into a slice of BucketOptioner options, deduplicating if specified.
func (bb *BucketBundle) Unbundle(deduplicate bool) ([]BucketOptioner, error) {
	options, err := bb.unbundle()
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

// Helper that recursively unwraps bundle into slice of options
func (bb *BucketBundle) unbundle() ([]BucketOptioner, error) {
	if bb == nil {
		return nil, nil
	}

	listLen := bb.bundleLength()

	options := make([]BucketOptioner, listLen)
	index := listLen - 1

	for listHead := bb; listHead != nil && listHead.option != nil; listHead = listHead.next {
		if converted, ok := listHead.option.(*BucketBundle); ok {
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

		options[index] = listHead.option
		index--
	}

	return options, nil
}

// Calculates the total length of a bundle, accounting for nested bundles.
func (bb *BucketBundle) bundleLength() int {
	if bb == nil {
		return 0
	}

	bundleLen := 0
	for ; bb != nil && bb.option != nil; bb = bb.next {
		if converted, ok := bb.option.(*BucketBundle); ok {
			// nested bundle
			bundleLen += converted.bundleLength()
			continue
		}

		bundleLen++
	}

	return bundleLen
}

// UploadBundle is a bundle of UploadOptioner options.
type UploadBundle struct {
	option UploadOptioner
	next   *UploadBundle
}

// uploadOption implements the UploadOptioner interface.
func (ub *UploadBundle) uploadOption(*Upload) error {
	return nil
}

// BundleUpload bundles UploadOptioner options.
func BundleUpload(opts ...UploadOptioner) *UploadBundle {
	head := uploadBundle

	for _, opt := range opts {
		newBundle := UploadBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

// ChunkSizeBytes specifies the chunk size for the bucket. Defaults to 255KB if not specified.
func (ub *UploadBundle) ChunkSizeBytes(b int32) *UploadBundle {
	bundle := &UploadBundle{
		option: ChunkSizeBytes(b),
		next:   ub,
	}

	return bundle
}

// Metadata specifies the metadata for a file upload.
func (ub *UploadBundle) Metadata(doc *bson.Document) *UploadBundle {
	bundle := &UploadBundle{
		option: Metadata(doc),
		next:   ub,
	}

	return bundle
}

// Unbundle transforms a bundle into a slice of BucketOptioner options, deduplicating if specified.
func (ub *UploadBundle) Unbundle(deduplicate bool) ([]UploadOptioner, error) {
	options, err := ub.unbundle()
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

// Helper that recursively unwraps bundle into slice of options
func (ub *UploadBundle) unbundle() ([]UploadOptioner, error) {
	if ub == nil {
		return nil, nil
	}

	listLen := ub.bundleLength()

	options := make([]UploadOptioner, listLen)
	index := listLen - 1

	for listHead := ub; listHead != nil && listHead.option != nil; listHead = listHead.next {
		if converted, ok := listHead.option.(*UploadBundle); ok {
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

		options[index] = listHead.option
		index--
	}

	return options, nil
}

// Calculates the total length of a bundle, accounting for nested bundles.
func (ub *UploadBundle) bundleLength() int {
	if ub == nil {
		return 0
	}

	bundleLen := 0
	for ; ub != nil && ub.option != nil; ub = ub.next {
		if converted, ok := ub.option.(*UploadBundle); ok {
			// nested bundle
			bundleLen += converted.bundleLength()
			continue
		}

		bundleLen++
	}

	return bundleLen
}

// FindBundle represents a bundle of GridFS Find options.
type FindBundle struct {
	option FindOptioner
	next   *FindBundle
}

func (fb *FindBundle) convertFindOption() findopt.Find {
	return nil
}

// BundleFind bundles FindOptioner options.
func BundleFind(opts ...FindOptioner) *FindBundle {
	head := findBundle

	for _, opt := range opts {
		newBundle := FindBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

// BatchSize specifies the number of documents to return per batch.
func (fb *FindBundle) BatchSize(i int32) *FindBundle {
	return &FindBundle{
		option: BatchSize(i),
		next:   fb,
	}
}

// Limit specifies the maximum number of documents to return.
func (fb *FindBundle) Limit(i int32) *FindBundle {
	return &FindBundle{
		option: Limit(i),
		next:   fb,
	}
}

// MaxTime specifies the maximum amount of time to allow the query to run.
func (fb *FindBundle) MaxTime(d time.Duration) *FindBundle {
	return &FindBundle{
		option: MaxTime(d),
		next:   fb,
	}
}

// NoCursorTimeout specifies that the server should not time out after an inactivity period.
func (fb *FindBundle) NoCursorTimeout(b bool) *FindBundle {
	return &FindBundle{
		option: NoCursorTimeout(b),
		next:   fb,
	}
}

// Skip specifies the number of documents to skip before returning.
func (fb *FindBundle) Skip(i int32) *FindBundle {
	return &FindBundle{
		option: Skip(i),
		next:   fb,
	}
}

// Sort specifies the order by which to sort results.
func (fb *FindBundle) Sort(sort interface{}) *FindBundle {
	return &FindBundle{
		option: Sort(sort),
		next:   fb,
	}
}

// Unbundle transforms a bundle into a slice of BucketOptioner options, deduplicating if specified.
func (fb *FindBundle) Unbundle(deduplicate bool) ([]FindOptioner, error) {
	options, err := fb.unbundle()
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

// Helper that recursively unwraps bundle into slice of options
func (fb *FindBundle) unbundle() ([]FindOptioner, error) {
	if fb == nil {
		return nil, nil
	}

	listLen := fb.bundleLength()

	options := make([]FindOptioner, listLen)
	index := listLen - 1

	for listHead := fb; listHead != nil && listHead.option != nil; listHead = listHead.next {
		if converted, ok := listHead.option.(*FindBundle); ok {
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

		options[index] = listHead.option
		index--
	}

	return options, nil
}

// Calculates the total length of a bundle, accounting for nested bundles.
func (fb *FindBundle) bundleLength() int {
	if fb == nil {
		return 0
	}

	bundleLen := 0
	for ; fb != nil && fb.option != nil; fb = fb.next {
		if converted, ok := fb.option.(*FindBundle); ok {
			// nested bundle
			bundleLen += converted.bundleLength()
			continue
		}

		bundleLen++
	}

	return bundleLen
}

// NameBundle bundles valid options for download by name commands.
type NameBundle struct {
	option NameOptioner
	next   *NameBundle
}

func (nb *NameBundle) nameOption() {}

// BundleName bundles NameOptioner options.
func BundleName(opts ...NameOptioner) *NameBundle {
	head := nameBundle

	for _, opt := range opts {
		newBundle := NameBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

// Revision specifies which revision of the file to retrieve. Defaults to -1.
// * Revision numbers are defined as follows:
// * 0 = the original stored file
// * 1 = the first revision
// * 2 = the second revision
// * etc…
// * -2 = the second most recent revision
// * -1 = the most recent revision
func (nb *NameBundle) Revision(i int32) *NameBundle {
	return &NameBundle{
		option: Revision(i),
		next:   nb,
	}
}

// Unbundle transforms a bundle into a slice of BucketOptioner options, deduplicating if specified.
func (nb *NameBundle) Unbundle(deduplicate bool) ([]NameOptioner, error) {
	options, err := nb.unbundle()
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

// Helper that recursively unwraps bundle into slice of options
func (nb *NameBundle) unbundle() ([]NameOptioner, error) {
	if nb == nil {
		return nil, nil
	}

	listLen := nb.bundleLength()

	options := make([]NameOptioner, listLen)
	index := listLen - 1

	for listHead := nb; listHead != nil && listHead.option != nil; listHead = listHead.next {
		if converted, ok := listHead.option.(*NameBundle); ok {
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

		options[index] = listHead.option
		index--
	}

	return options, nil
}

// Calculates the total length of a bundle, accounting for nested bundles.
func (nb *NameBundle) bundleLength() int {
	if nb == nil {
		return 0
	}

	bundleLen := 0
	for ; nb != nil && nb.option != nil; nb = nb.next {
		if converted, ok := nb.option.(*NameBundle); ok {
			// nested bundle
			bundleLen += converted.bundleLength()
			continue
		}

		bundleLen++
	}

	return bundleLen
}

// BucketName specifies the name of the bucket. Defaults to 'fs' if not specified.
func BucketName(name string) OptBucketName {
	return OptBucketName(name)
}

// ChunkSizeBytes specifies the chunk size for the bucket. Defaults to 255KB if not specified.
func ChunkSizeBytes(b int32) OptChunkSizeBytes {
	return OptChunkSizeBytes(b)
}

// WriteConcern specifies the write concern for the bucket.
func WriteConcern(wc *writeconcern.WriteConcern) OptWriteConcern {
	return OptWriteConcern{
		WriteConcern: wc,
	}
}

// ReadConcern specifies the read concern for the bucket.
func ReadConcern(rc *readconcern.ReadConcern) OptReadConcern {
	return OptReadConcern{
		ReadConcern: rc,
	}
}

// ReadPreference specifies the read preference for the bucket.
func ReadPreference(rp *readpref.ReadPref) OptReadPreference {
	return OptReadPreference{
		ReadPref: rp,
	}
}

// Metadata specifies the metadata for a file upload.
func Metadata(doc *bson.Document) OptMetadata {
	return OptMetadata{
		Document: doc,
	}
}

// BatchSize specifies the number of documents to return per batch.
func BatchSize(i int32) OptBatchSize {
	return OptBatchSize(i)
}

// Limit specifies the maximum number of documents to return.
func Limit(i int32) OptLimit {
	return OptLimit(i)
}

// MaxTime specifies the maximum amount of time to allow the query to run.
func MaxTime(d time.Duration) OptMaxTime {
	return OptMaxTime(d)
}

// NoCursorTimeout specifies that the server should not time out after an inactivity period.
func NoCursorTimeout(b bool) OptNoCursorTimeout {
	return OptNoCursorTimeout(b)
}

// Skip specifies the number of documents to skip before returning.
func Skip(i int32) OptSkip {
	return OptSkip(i)
}

// Sort specifies the order by which to sort results.
func Sort(sort interface{}) OptSort {
	return OptSort{
		Sort: sort,
	}
}

// Revision specifies which revision of the file to retrieve. Defaults to -1.
// * Revision numbers are defined as follows:
// * 0 = the original stored file
// * 1 = the first revision
// * 2 = the second revision
// * etc…
// * -2 = the second most recent revision
// * -1 = the most recent revision
func Revision(i int32) OptRevision {
	return OptRevision(i)
}

// OptBucketName specifies the name of the bucket. Defaults to 'fs' if not specified.
type OptBucketName string

// bucketOption implements the BucketOptioner interface.
func (opt OptBucketName) bucketOption(bucket *Bucket) error {
	bucket.name = string(opt)
	return nil
}

// OptChunkSizeBytes specifies the chunk size for the bucket. Defaults to 255KB if not specified.
type OptChunkSizeBytes int32

// bucketOption implements the BucketOptioner interface.
func (opt OptChunkSizeBytes) bucketOption(bucket *Bucket) error {
	bucket.chunkSize = int32(opt)
	return nil
}

// uploadOption implements the UploadOptioner interface.
func (opt OptChunkSizeBytes) uploadOption(upload *Upload) error {
	upload.chunkSize = int32(opt)
	return nil
}

// OptWriteConcern specifies the write concern for the bucket.
type OptWriteConcern struct {
	*writeconcern.WriteConcern
}

// bucketOption implements the BucketOptioner interface.
func (opt OptWriteConcern) bucketOption(bucket *Bucket) error {
	bucket.wc = opt.WriteConcern
	return nil
}

// OptReadConcern specifies the read concern for the bucket.
type OptReadConcern struct {
	*readconcern.ReadConcern
}

// bucketOption implements the BucketOptioner interface.
func (opt OptReadConcern) bucketOption(bucket *Bucket) error {
	bucket.rc = opt.ReadConcern
	return nil
}

// OptReadPreference specifies the read preference for the bucket.
type OptReadPreference struct {
	*readpref.ReadPref
}

// bucketOption implements the BucketOptioner interface.
func (opt OptReadPreference) bucketOption(bucket *Bucket) error {
	bucket.rp = opt.ReadPref
	return nil
}

// OptMetadata specifies the metadata for a file upload.
type OptMetadata struct {
	*bson.Document
}

// uploadOption implements the UploadOptioner interface.
func (opt OptMetadata) uploadOption(upload *Upload) error {
	upload.metadata = opt.Document
	return nil
}

// OptBatchSize specifies the number of documents to return per batch.
type OptBatchSize int32

func (opt OptBatchSize) convertFindOption() findopt.Find {
	return findopt.BatchSize(int32(opt))
}

// OptLimit specifies the maximum number of documents to return.
type OptLimit int32

func (opt OptLimit) convertFindOption() findopt.Find {
	return findopt.Limit(int64(opt))
}

// OptMaxTime specifies the maximum amount of time to allow the query to run.
type OptMaxTime time.Duration

func (opt OptMaxTime) convertFindOption() findopt.Find {
	return findopt.MaxTime(time.Duration(opt))
}

// OptNoCursorTimeout specifies that the server should not time out after an inactivity period.
type OptNoCursorTimeout bool

func (opt OptNoCursorTimeout) convertFindOption() findopt.Find {
	return findopt.NoCursorTimeout(bool(opt))
}

// OptSkip specifies the number of documents to skip before returning.
type OptSkip int32

func (opt OptSkip) convertFindOption() findopt.Find {
	return findopt.Skip(int64(opt))
}

// OptSort specifies the order by which to sort results.
type OptSort struct {
	Sort interface{}
}

func (opt OptSort) convertFindOption() findopt.Find {
	return findopt.Sort(opt.Sort)
}

// OptRevision specifies which revision of the file to retrieve. Defaults to -1.
// * Revision numbers are defined as follows:
// * 0 = the original stored file
// * 1 = the first revision
// * 2 = the second revision
// * etc…
// * -2 = the second most recent revision
// * -1 = the most recent revision
type OptRevision int32

func (OptRevision) nameOption() {}
