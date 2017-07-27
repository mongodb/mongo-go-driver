package compress

import (
	"fmt"
	"io"
	"strings"

	"bytes"

	"github.com/10gen/mongo-go-driver/internal"
	"github.com/10gen/mongo-go-driver/msg"
)

var nonCompressibleCommands map[string]struct{}
var dollarQuery = []byte{'$', 'q', 'u', 'e', 'r', 'y', 0}

func init() {
	nonCompressibleCommands = make(map[string]struct{})

	nonCompressibleCommands["ismaster"] = struct{}{}
	nonCompressibleCommands["isMaster"] = struct{}{}
	nonCompressibleCommands["saslStart"] = struct{}{}
	nonCompressibleCommands["saslContinue"] = struct{}{}
	nonCompressibleCommands["getNonce"] = struct{}{}
	nonCompressibleCommands["authenticate"] = struct{}{}
	nonCompressibleCommands["createUser"] = struct{}{}
	nonCompressibleCommands["updateUser"] = struct{}{}
	nonCompressibleCommands["copydbSaslStart"] = struct{}{}
	nonCompressibleCommands["copydbgetnonce"] = struct{}{}
	nonCompressibleCommands["copydb"] = struct{}{}
}

// NewCodec creates a Codec that handles compression.
func NewCodec(inner msg.Codec, compressors ...Compressor) *Codec {
	return &Codec{
		inner:       inner,
		compressors: compressors,
		headerBytes: make([]byte, 25),
	}
}

// Codec is a codec that uses a Compressor
type Codec struct {
	inner       msg.Codec
	compressors []Compressor
	compressor  Compressor
	headerBytes []byte
}

// SetCompressors sets the compressors that will be used to compress
// messages.
func (c *Codec) SetCompressors(names []string) {
	for _, compressor := range c.compressors {
		for _, name := range names {
			if compressor.Name() == name {
				c.compressor = compressor
				return
			}
		}
	}
}

// Decode decodes a message from the reader.
func (c *Codec) Decode(reader io.Reader) (msg.Message, error) {
	_, err := io.ReadFull(reader, c.headerBytes)
	if err != nil {
		return nil, internal.WrapError(err, "unable to decode message length")
	}

	// we don't need these guys, but left here for documentation
	// length := readInt32(c.headerBytes, 0)
	// requestID := readInt32(c.headerBytes, 4)
	// responseTo := readInt32(c.headerBytes, 8)
	opCode := readInt32(c.headerBytes, 12)

	if opCode != compressOpcode {
		mr := io.MultiReader(bytes.NewBuffer(c.headerBytes), reader)
		return c.inner.Decode(mr)
	}

	opCode = readInt32(c.headerBytes, 16)
	uncompressedSize := readInt32(c.headerBytes, 20)
	compressorID := readUint8(c.headerBytes, 24)
	uncompressed := make([]byte, uncompressedSize)

	found := false
	for _, compressor := range c.compressors {
		if compressor.ID() == compressorID {
			found = true
			err := compressor.Decompress(reader, uncompressed)
			if err != nil {
				return nil, internal.WrapErrorf(err, "failed decompressing using %s", compressor.Name())
			}
		}
	}

	if !found {
		return nil, fmt.Errorf("unsupported compressor: %d", compressorID)
	}

	setInt32(c.headerBytes, 0, uncompressedSize+16)
	setInt32(c.headerBytes, 12, opCode)

	mr := io.MultiReader(bytes.NewBuffer(c.headerBytes[0:16]), bytes.NewBuffer(uncompressed))
	return c.inner.Decode(mr)
}

// Encode encodes a number of messages to the writer.
func (c *Codec) Encode(writer io.Writer, msgs ...msg.Message) error {
	if c.compressor == nil {
		return c.inner.Encode(writer, msgs...)
	}

	b := make([]byte, 0, defaultEncodeBufferSize)
	var err error
	for _, msg := range msgs {
		start := len(b)
		var msgBuffer bytes.Buffer
		err = c.inner.Encode(&msgBuffer, msg)
		if err != nil {
			return err
		}

		msgBytes := msgBuffer.Bytes()
		if !c.shouldCompress(msgBytes, msg) {
			b = append(b, msgBytes...)
			continue
		}

		compressorID, compressedBytes, err := c.compress(msgBytes)
		if err != nil {
			return err
		}

		requestID := readInt32(msgBytes, 4)
		responseTo := readInt32(msgBytes, 8)
		originalOpcode := readInt32(msgBytes, 12)

		b = addHeader(b, 0, requestID, responseTo, compressOpcode)
		b = addInt32(b, originalOpcode)
		b = addInt32(b, int32(len(msgBytes)-16)) // less header size
		b = append(b, compressorID)
		b = append(b, compressedBytes...)
		setInt32(b, int32(start), int32(len(b)-start))
	}

	_, err = writer.Write(b)
	if err != nil {
		return internal.WrapError(err, "unable to encode messages")
	}
	return nil
}

func (c *Codec) compress(msgBytes []byte) (uint8, []byte, error) {
	var compressed bytes.Buffer
	err := c.compressor.Compress(msgBytes[16:], &compressed)
	if err != nil {
		return 0, nil, internal.WrapErrorf(err, "failed compressing using %s", c.compressor.Name())
	}
	return c.compressor.ID(), compressed.Bytes(), err
}

func (c *Codec) shouldCompress(msgBytes []byte, m msg.Message) bool {
	switch typedM := m.(type) {
	case *msg.Query:
		if strings.HasSuffix(typedM.FullCollectionName, ".$cmd") {
			queryPos := 20 + len(typedM.FullCollectionName) + 1 + 8 + 4

			// command could either be just a document, or it could be a document wrapped in $query
			if msgBytes[queryPos] == 3 {
				queryPos++
				if !bytes.Equal(msgBytes[queryPos:queryPos+len(dollarQuery)], dollarQuery) {
					return false
				}

				queryPos += len(dollarQuery) + 4 // $query + length
			}

			queryPos++

			// we should be at the start of the first element in the document
			npIdx := bytes.Index(msgBytes[queryPos:], []byte{0})
			if npIdx < 0 {
				return false
			}

			if _, ok := nonCompressibleCommands[string(msgBytes[queryPos:queryPos+npIdx])]; ok {
				return false
			}
		}
	}

	return true
}
