package gridfs

import (
	"context"

	"io"

	"errors"

	"time"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/collectionopt"
	"github.com/mongodb/mongo-go-driver/mongo/findopt"
)

// TODO: add sessions options

// DefaultChunkSize is the default size of each file chunk.
const DefaultChunkSize int32 = 255 * 1000 // 255 KB

// ErrFileNotFound occurs if a user asks to download a file with a file ID that isn't found in the files collection.
var ErrFileNotFound = errors.New("file with given parameters not found")

// Bucket represents a GridFS bucket.
type Bucket struct {
	db         *mongo.Database
	chunksColl *mongo.Collection // collection to store file chunks
	filesColl  *mongo.Collection // collection to store file metadata

	name      string
	chunkSize int32
	wc        *writeconcern.WriteConcern
	rc        *readconcern.ReadConcern
	rp        *readpref.ReadPref

	firstWriteDone bool
	readBuf        []byte
	writeBuf       []byte

	readDeadline  time.Time
	writeDeadline time.Time
}

// Upload contains options to upload a file to a bucket.
type Upload struct {
	chunkSize int32
	metadata  *bson.Document
}

// NewBucket creates a GridFS bucket.
func NewBucket(db *mongo.Database, opts ...BucketOptioner) (*Bucket, error) {
	b := &Bucket{
		name:      "fs",
		chunkSize: DefaultChunkSize,
		db:        db,
		wc:        db.WriteConcern(),
		rc:        db.ReadConcern(),
		rp:        db.ReadPreference(),
	}

	bucketOpts, err := BundleBucket(opts...).Unbundle(true)
	if err != nil {
		return nil, err
	}

	for _, opt := range bucketOpts {
		err = opt.bucketOption(b)
		if err != nil {
			return nil, err
		}
	}

	var collOpts = []collectionopt.Option{
		collectionopt.WriteConcern(b.wc),
		collectionopt.ReadConcern(b.rc),
		collectionopt.ReadPreference(b.rp),
	}

	b.chunksColl = db.Collection(b.name+".chunks", collOpts...)
	b.filesColl = db.Collection(b.name+".files", collOpts...)
	b.readBuf = make([]byte, b.chunkSize)
	b.writeBuf = make([]byte, b.chunkSize)

	return b, nil
}

// SetWriteDeadline sets the write deadline for this bucket.
func (b *Bucket) SetWriteDeadline(t time.Time) error {
	b.writeDeadline = t
	return nil
}

// SetReadDeadline sets the read deadline for this bucket
func (b *Bucket) SetReadDeadline(t time.Time) error {
	b.readDeadline = t
	return nil
}

// OpenUploadStream creates a file ID new upload stream for a file given the filename.
func (b *Bucket) OpenUploadStream(filename string, opts ...UploadOptioner) (*UploadStream, error) {
	return b.OpenUploadStreamWithID(objectid.New(), filename, opts...)
}

// OpenUploadStreamWithID creates a new upload stream for a file given the file ID and filename.
func (b *Bucket) OpenUploadStreamWithID(fileID objectid.ObjectID, filename string, opts ...UploadOptioner) (*UploadStream, error) {
	ctx, cancel := deadlineContext(b.writeDeadline)
	if cancel != nil {
		defer cancel()
	}

	if err := b.checkFirstWrite(ctx); err != nil {
		return nil, err
	}

	upload, err := b.parseUploadOptions(opts...)
	if err != nil {
		return nil, err
	}

	return newUploadStream(upload, fileID, filename, b.chunksColl, b.filesColl), nil
}

// UploadFromStream creates a fileID and uploads a file given a source stream.
func (b *Bucket) UploadFromStream(filename string, source io.Reader, opts ...UploadOptioner) (objectid.ObjectID, error) {
	fileID := objectid.New()
	err := b.UploadFromStreamWithID(fileID, filename, source, opts...)
	return fileID, err
}

// UploadFromStreamWithID uploads a file given a source stream.
func (b *Bucket) UploadFromStreamWithID(fileID objectid.ObjectID, filename string, source io.Reader, opts ...UploadOptioner) error {
	us, err := b.OpenUploadStreamWithID(fileID, filename, opts...)
	if err != nil {
		return err
	}

	err = us.SetWriteDeadline(b.writeDeadline)
	if err != nil {
		_ = us.Close()
		return err
	}

	for {
		n, err := source.Read(b.readBuf)
		if err != nil && err != io.EOF {
			_ = us.Abort() // upload considered aborted if source stream returns an error
			return err
		}

		if n > 0 {
			_, err := us.Write(b.readBuf[:n])
			if err != nil {
				return err
			}
		}

		if n == 0 || err == io.EOF {
			break
		}
	}

	return us.Close()
}

// OpenDownloadStream creates a stream from which the contents of the file can be read.
func (b *Bucket) OpenDownloadStream(fileID objectid.ObjectID) (*DownloadStream, error) {
	return b.openDownloadStream(bson.NewDocument(
		bson.EC.ObjectID("_id", fileID),
	))
}

// DownloadToStream downloads the file with the specified fileID and writes it to the provided io.Writer.
// Returns the number of bytes written to the steam and an error, or nil if there was no error.
func (b *Bucket) DownloadToStream(fileID objectid.ObjectID, stream io.Writer) (int64, error) {
	ds, err := b.OpenDownloadStream(fileID)
	if err != nil {
		return 0, err
	}

	return b.downloadToStream(ds, stream)
}

// OpenDownloadStreamByName opens a download stream for the file with the given filename.
func (b *Bucket) OpenDownloadStreamByName(filename string, opts ...NameOptioner) (*DownloadStream, error) {
	var numSkip int32 = -1
	var sortOrder int32 = 1

	for _, opt := range opts {
		if revision, ok := opt.(OptRevision); ok {
			numSkip = int32(revision)
		}
	}

	if numSkip < 0 {
		sortOrder = -1
		numSkip = (-1 * numSkip) - 1
	}

	findOpts := []findopt.Find{
		findopt.Skip(int64(numSkip)),
		findopt.Sort(bson.NewDocument(
			bson.EC.Int32("uploadDate", sortOrder),
		)),
	}

	return b.openDownloadStream(bson.NewDocument(
		bson.EC.String("filename", filename),
	), findOpts...)
}

// DownloadToStreamByName downloads the file with the given name to the given io.Writer.
func (b *Bucket) DownloadToStreamByName(filename string, stream io.Writer, opts ...NameOptioner) (int64, error) {
	ds, err := b.OpenDownloadStreamByName(filename, opts...)
	if err != nil {
		return 0, err
	}

	return b.downloadToStream(ds, stream)
}

// Delete deletes all chunks and metadata associated with the file with the given file ID.
func (b *Bucket) Delete(fileID objectid.ObjectID) error {
	// delete document in files collection and then chunks to minimize race conditions

	ctx, cancel := deadlineContext(b.writeDeadline)
	if cancel != nil {
		defer cancel()
	}

	res, err := b.filesColl.DeleteOne(ctx, bson.NewDocument(
		bson.EC.ObjectID("_id", fileID),
	))
	if err == nil && res.DeletedCount == 0 {
		err = ErrFileNotFound
	}
	if err != nil {
		_ = b.deleteChunks(ctx, fileID) // can attempt to delete chunks even if no docs in files collection matched
		return err
	}

	return b.deleteChunks(ctx, fileID)
}

// Find returns the files collection documents that match the given filter.
func (b *Bucket) Find(filter interface{}, opts ...FindOptioner) (mongo.Cursor, error) {
	ctx, cancel := deadlineContext(b.readDeadline)
	if cancel != nil {
		defer cancel()
	}

	findOpts := make([]findopt.Find, 0, len(opts))
	for i, opt := range opts {
		findOpts[i] = opt.convertFindOption()
	}

	return b.filesColl.Find(ctx, filter, findOpts...)
}

// Rename renames the stored file with the specified file ID.
func (b *Bucket) Rename(fileID objectid.ObjectID, newFilename string) error {
	ctx, cancel := deadlineContext(b.writeDeadline)
	if cancel != nil {
		defer cancel()
	}

	res, err := b.filesColl.UpdateOne(ctx, bson.NewDocument(
		bson.EC.ObjectID("_id", fileID),
	), bson.NewDocument(
		bson.EC.SubDocument("$set", bson.NewDocument(
			bson.EC.String("filename", newFilename),
		)),
	))
	if err != nil {
		return err
	}

	if res.MatchedCount == 0 {
		return ErrFileNotFound
	}

	return nil
}

// Drop drops the files and chunks collections associated with this bucket.
func (b *Bucket) Drop() error {
	ctx, cancel := deadlineContext(b.writeDeadline)
	if cancel != nil {
		defer cancel()
	}

	err := b.filesColl.Drop(ctx)
	if err != nil {
		return err
	}

	return b.chunksColl.Drop(ctx)
}

func (b *Bucket) openDownloadStream(filter interface{}, opts ...findopt.Find) (*DownloadStream, error) {
	ctx, cancel := deadlineContext(b.readDeadline)
	if cancel != nil {
		defer cancel()
	}

	cursor, err := b.findFile(ctx, filter, opts...)
	if err != nil {
		return nil, err
	}

	fileRdr, err := cursor.DecodeBytes()
	if err != nil {
		return nil, err
	}

	fileLenElem, err := fileRdr.Lookup("length")
	if err != nil {
		return nil, err
	}
	fileIDElem, err := fileRdr.Lookup("_id")
	if err != nil {
		return nil, err
	}

	fileLen := fileLenElem.Value().Int32()
	if fileLen == 0 {
		return newDownloadStream(nil, b.chunkSize, 0), nil
	}

	chunksCursor, err := b.findChunks(ctx, fileIDElem.Value().ObjectID())
	if err != nil {
		return nil, err
	}
	return newDownloadStream(chunksCursor, b.chunkSize, int64(fileLen)), nil
}

func deadlineContext(deadline time.Time) (context.Context, context.CancelFunc) {
	if deadline.Equal(time.Time{}) {
		return context.Background(), nil
	}

	return context.WithDeadline(context.Background(), deadline)
}

func (b *Bucket) downloadToStream(ds *DownloadStream, stream io.Writer) (int64, error) {
	err := ds.SetReadDeadline(b.readDeadline)
	if err != nil {
		_ = ds.Close()
		return 0, err
	}

	copied, err := io.Copy(stream, ds)
	if err != nil {
		_ = ds.Close()
		return 0, err
	}

	return copied, ds.Close()
}

func (b *Bucket) deleteChunks(ctx context.Context, fileID objectid.ObjectID) error {
	_, err := b.chunksColl.DeleteMany(ctx, bson.NewDocument(
		bson.EC.ObjectID("files_id", fileID),
	))
	return err
}

func (b *Bucket) findFile(ctx context.Context, filter interface{}, opts ...findopt.Find) (mongo.Cursor, error) {
	cursor, err := b.filesColl.Find(ctx, filter, opts...)
	if err != nil {
		return nil, err
	}

	if !cursor.Next(ctx) {
		_ = cursor.Close(ctx)
		return nil, ErrFileNotFound
	}

	return cursor, nil
}

func (b *Bucket) findChunks(ctx context.Context, fileID objectid.ObjectID) (mongo.Cursor, error) {
	chunksCursor, err := b.chunksColl.Find(ctx, bson.NewDocument(
		bson.EC.ObjectID("files_id", fileID),
	), findopt.Sort(bson.NewDocument(
		bson.EC.Int32("n", 1), // sort by chunk index
	)))
	if err != nil {
		return nil, err
	}

	return chunksCursor, nil
}

// Create an index if it doesn't already exist
func createIndexIfNotExists(ctx context.Context, iv mongo.IndexView, model mongo.IndexModel) error {
	c, err := iv.List(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = c.Close(ctx)
	}()

	var found bool
	for c.Next(ctx) {
		rdr, err := c.DecodeBytes()
		if err != nil {
			return err
		}

		keyElem, err := rdr.Lookup("key")
		if err != nil {
			return err
		}

		if model.Keys.Equal(keyElem.Value().MutableDocument()) {
			found = true
			break
		}
	}

	if !found {
		_, err = iv.CreateOne(ctx, model)
		if err != nil {
			return err
		}
	}

	return nil
}

// create indexes on the files and chunks collection if needed
func (b *Bucket) createIndexes(ctx context.Context) error {
	// must use primary read pref mode to check if files coll empty
	cloned, err := b.filesColl.Clone(collectionopt.ReadPreference(readpref.Primary()))
	if err != nil {
		return err
	}

	docRes := cloned.FindOne(ctx, bson.NewDocument(), findopt.Projection(bson.NewDocument(
		bson.EC.Int32("_id", 1),
	)))

	err = docRes.Decode(nil)
	if err == mongo.ErrNoDocuments {
		filesIv := b.filesColl.Indexes()
		chunksIv := b.chunksColl.Indexes()

		filesModel := mongo.IndexModel{
			Keys: bson.NewDocument(
				bson.EC.Int32("filename", 1),
				bson.EC.Int32("uploadDate", 1),
			),
		}

		chunksModel := mongo.IndexModel{
			Keys: bson.NewDocument(
				bson.EC.Int32("files_id", 1),
				bson.EC.Int32("n", 1),
			),
		}

		if err = createIndexIfNotExists(ctx, filesIv, filesModel); err != nil {
			return err
		}
		if err = createIndexIfNotExists(ctx, chunksIv, chunksModel); err != nil {
			return err
		}
	}

	return nil
}

func (b *Bucket) checkFirstWrite(ctx context.Context) error {
	if !b.firstWriteDone {
		// before the first write operation, must determine if files collection is empty
		// if so, create indexes if they do not already exist

		if err := b.createIndexes(ctx); err != nil {
			return err
		}
		b.firstWriteDone = true
	}

	return nil
}

func (b *Bucket) parseUploadOptions(opts ...UploadOptioner) (*Upload, error) {
	upload := &Upload{
		chunkSize: b.chunkSize, // upload chunk size defaults to bucket's value
	}

	var err error
	for _, opt := range opts {
		err = opt.uploadOption(upload)
		if err != nil {
			return nil, err
		}
	}

	return upload, nil
}
