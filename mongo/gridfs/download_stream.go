package gridfs

import (
	"context"

	"errors"

	"time"

	"github.com/mongodb/mongo-go-driver/mongo"
)

// ErrWrongIndex is used when the chunk retrieved from the server does not have the expected index.
var ErrWrongIndex = errors.New("chunk index does not match expected index")

// ErrWrongSize is used when the chunk retrieved from the server does not have the expected size.
var ErrWrongSize = errors.New("chunk size does not match expected size")

var errNoMoreChunks = errors.New("no more chunks remaining")

// DownloadStream is a io.Reader that can be used to download a file from a GridFS bucket.
type DownloadStream struct {
	numChunks     int32
	chunkSize     int32
	cursor        mongo.Cursor
	done          bool
	closed        bool
	buffer        []byte // store up to 1 chunk if the user provided buffer isn't big enough
	bufferStart   int
	expectedChunk int32 // index of next expected chunk
	readDeadline  time.Time
}

func newDownloadStream(cursor mongo.Cursor, chunkSize int32, fileLen int64) *DownloadStream {
	numChunks := int32(float64(fileLen) / float64(chunkSize))

	return &DownloadStream{
		numChunks: numChunks,
		chunkSize: chunkSize,
		cursor:    cursor,
		buffer:    make([]byte, chunkSize),
		done:      cursor == nil,
	}
}

// Close closes this download stream.
func (ds *DownloadStream) Close() error {
	if ds.closed {
		return ErrStreamClosed
	}

	ds.closed = true
	return nil
}

// SetReadDeadline sets the read deadline for this download stream.
func (ds *DownloadStream) SetReadDeadline(t time.Time) error {
	if ds.closed {
		return ErrStreamClosed
	}

	ds.readDeadline = t
	return nil
}

// Read reads the file from the server and writes it to a destination byte slice.
func (ds *DownloadStream) Read(p []byte) (int, error) {
	if ds.closed {
		return 0, ErrStreamClosed
	}

	if ds.done {
		return 0, nil
	}

	ctx, cancel := context.WithDeadline(context.Background(), ds.readDeadline)
	defer cancel()

	bytesCopied := 0
	var err error

	for bytesCopied < len(p) {
		if ds.bufferStart == 0 {
			// buffer empty
			err = ds.fillBuffer(ctx)
			if err != nil {
				if err == errNoMoreChunks {
					return bytesCopied, nil
				}

				return bytesCopied, err
			}
		}

		copied := copy(p, ds.buffer[ds.bufferStart:])
		bytesCopied += copied
		ds.bufferStart = (ds.bufferStart + copied) % int(ds.chunkSize)
	}

	return len(p), nil
}

func (ds *DownloadStream) fillBuffer(ctx context.Context) error {
	if !ds.cursor.Next(ctx) {
		ds.done = true
		return errNoMoreChunks
	}

	nextChunk, err := ds.cursor.DecodeBytes()
	if err != nil {
		return err
	}

	chunkIndex, err := nextChunk.Lookup("n")
	if err != nil {
		return err
	}

	if chunkIndex.Value().Int32() != ds.expectedChunk {
		return ErrWrongIndex
	}

	ds.expectedChunk++
	data, err := nextChunk.Lookup("data")
	if err != nil {
		return err
	}

	_, dataBytes := data.Value().Binary()
	if ds.expectedChunk != ds.numChunks && int32(len(dataBytes)) != ds.chunkSize {
		return ErrWrongSize
	}

	copy(ds.buffer, dataBytes)
	ds.bufferStart = 0
	return nil
}
