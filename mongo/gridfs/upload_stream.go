package gridfs

import (
	"errors"

	"math"

	"context"
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
	"github.com/mongodb/mongo-go-driver/mongo"
)

// UploadBufferSize is the size in bytes of one stream batch. Chunks will be written to the db after the sum of chunk
// lengths is equal to the batch size.
const UploadBufferSize = 16 * 1000000 // 16 MB

// ErrStreamClosed is an error returned if an operation is attempted on a closed/aborted stream.
var ErrStreamClosed = errors.New("stream is closed or aborted")

// UploadStream is used to upload files in chunks.
type UploadStream struct {
	*Upload // chunk size and metadata
	FileID  objectid.ObjectID

	chunkIndex    int
	chunksColl    *mongo.Collection // collection to store file chunks
	filename      string
	filesColl     *mongo.Collection // collection to store file metadata
	closed        bool
	buffer        []byte
	bufferIndex   int
	fileLen       int64
	writeDeadline time.Time
}

// NewUploadStream creates a new upload stream.
func newUploadStream(upload *Upload, fileID objectid.ObjectID, filename string, chunks *mongo.Collection, files *mongo.Collection) *UploadStream {
	return &UploadStream{
		Upload: upload,
		FileID: fileID,

		chunksColl: chunks,
		filename:   filename,
		filesColl:  files,
		buffer:     make([]byte, UploadBufferSize),
	}
}

// Close closes this upload stream.
func (us *UploadStream) Close() error {
	if us.closed {
		return ErrStreamClosed
	}

	ctx, cancel := deadlineContext(us.writeDeadline)
	if cancel != nil {
		defer cancel()
	}

	if us.bufferIndex != 0 {
		if err := us.uploadChunks(ctx); err != nil {
			return err
		}
	}

	if err := us.createFilesCollDoc(ctx); err != nil {
		return err
	}

	us.closed = true
	return nil
}

// SetWriteDeadline sets the write deadline for this stream.
func (us *UploadStream) SetWriteDeadline(t time.Time) error {
	if us.closed {
		return ErrStreamClosed
	}

	us.writeDeadline = t
	return nil
}

// Write transfers the contents of a byte slice into this upload stream. If the stream's underlying buffer fills up,
// the buffer will be uploaded as chunks to the server. Implements the io.Writer interface.
func (us *UploadStream) Write(p []byte) (int, error) {
	if us.closed {
		return 0, ErrStreamClosed
	}

	var ctx context.Context

	ctx, cancel := deadlineContext(us.writeDeadline)
	if cancel != nil {
		defer cancel()
	}

	origLen := len(p)
	for {
		if us.bufferIndex == UploadBufferSize {
			err := us.uploadChunks(ctx)
			if err != nil {
				return 0, err
			}
			us.bufferIndex = 0
		}

		if len(p) == 0 {
			break
		}

		n := copy(us.buffer[us.bufferIndex:], p) // copy as much as possible
		p = p[n:]
		us.bufferIndex += n
		break
	}

	return origLen, nil
}

// Abort closes the stream and deletes all file chunks that have already been written.
func (us *UploadStream) Abort() error {
	if us.closed {
		return ErrStreamClosed
	}

	ctx, cancel := deadlineContext(us.writeDeadline)
	if cancel != nil {
		defer cancel()
	}

	_, err := us.chunksColl.DeleteMany(ctx, bson.NewDocument(
		bson.EC.ObjectID("files_id", us.FileID),
	))
	if err != nil {
		return err
	}

	us.closed = true
	return nil
}

func (us *UploadStream) uploadChunks(ctx context.Context) error {
	numChunks := math.Ceil(float64(us.bufferIndex) / float64(us.chunkSize))

	docs := make([]interface{}, int(numChunks))

	for i := 0; i < us.bufferIndex; i += int(us.chunkSize) {
		var chunkData []byte
		if us.bufferIndex-i < int(us.chunkSize) {
			chunkData = us.buffer[i:us.bufferIndex]
		} else {
			chunkData = us.buffer[i : i+int(us.chunkSize)]
		}

		docs[us.chunkIndex] = bson.NewDocument(
			bson.EC.ObjectID("_id", objectid.New()),
			bson.EC.ObjectID("files_id", us.FileID),
			bson.EC.Int32("n", int32(us.chunkIndex)),
			bson.EC.Binary("data", chunkData),
		)

		us.chunkIndex++
		us.fileLen += int64(len(chunkData))
	}

	_, err := us.chunksColl.InsertMany(ctx, docs)
	if err != nil {
		return err
	}

	return nil
}

func (us *UploadStream) createFilesCollDoc(ctx context.Context) error {
	doc := bson.NewDocument(
		bson.EC.ObjectID("_id", us.FileID),
		bson.EC.Int64("length", us.fileLen),
		bson.EC.Int32("chunkSize", us.chunkSize),
		bson.EC.DateTime("uploadDate", time.Now().UnixNano()/int64(time.Millisecond)),
		bson.EC.String("filename", us.filename),
	)

	if us.metadata != nil {
		doc.Append(bson.EC.SubDocument("metadata", us.metadata))
	}

	_, err := us.filesColl.InsertOne(ctx, doc)
	if err != nil {
		return err
	}

	return nil
}
