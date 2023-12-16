package mnet

// Streamer represents a Connection that supports streaming wire protocol
// messages using the moreToCome and exhaustAllowed flags.
//
// The StartStreaming and IsStreaming functions correspond to the moreToCome
// flag on server responses. If a response has moreToCome set,
// SetStreaming(true) will be called and CurrentlyStreaming() should return
// true.
//
// Streamable corresponds to the exhaustAllowed flag. The operations layer will
// set exhaustAllowed on outgoing wire messages to inform the server that the
// driver supports streaming.
type Streamer interface {
	StartStreaming(bool)
	IsStreaming() bool
	Streamable() bool
}

// defaultStreamer is the streamer that is assigned to a Connection object on
// construction if one has not been provided.
type defaultStreamer struct{}

// Assert defaultStreamer implements Streamer at compile-time.
var _ Streamer = &defaultStreamer{}

// TODO: Add logic
func (*defaultStreamer) StartStreaming(bool) {}
func (*defaultStreamer) IsStreaming() bool   { return false }
func (*defaultStreamer) Streamable() bool    { return false }
