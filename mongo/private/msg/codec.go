// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

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
