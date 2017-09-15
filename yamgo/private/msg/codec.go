package msg

import "io"

// Encoder encodes messages.
type Encoder interface {
	// Encode encodes a number of messages to the writer.
	Encode(io.Writer, ...Message) error
}

// Decoder decodes messages.
type Decoder interface {
	// Decode decodes one message from the reader.
	Decode(io.Reader) (Message, error)
}

// Codec encodes and decodes messages.
type Codec interface {
	Encoder
	Decoder
}
