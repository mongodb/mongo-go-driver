package bson

import (
	"io"
	"reflect"

	"github.com/mongodb/mongo-go-driver/bson-codec-design/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson-codec-design/bson/objectid"
)

type Document struct{}

type Codec interface {
	EncodeValue(Registry, ValueWriter, interface{}) error
	DecodeValue(Registry, ValueReader, interface{}) error
}

type CodecZeroer interface {
	Codec
	IsZero(interface{}) bool
}

type ArrayReader interface {
	ReadValue() (ValueReader, error)
}

type DocumentReader interface {
	ReadElement() (string, ValueReader, error)
}

type ValueReader interface {
	Type() Type
	Skip() error

	ReadArray() (ArrayReader, error)
	ReadBinary() (b []byte, btype byte, err error)
	ReadBoolean() (bool, error)
	ReadDocument() (DocumentReader, error)
	ReadCodeWithScope() (code string, dr DocumentReader, err error)
	ReadDBPointer() (ns string, oid objectid.ObjectID, err error)
	ReadDateTime() (int64, error)
	ReadDecimal128() (decimal.Decimal128, error)
	ReadDouble() (float64, error)
	ReadInt32() (int32, error)
	ReadInt64() (int64, error)
	ReadJavascript() (code string, err error)
	ReadMaxKey() error
	ReadMinKey() error
	ReadNull() error
	ReadObjectID() (objectid.ObjectID, error)
	ReadRegex() (pattern, options string, err error)
	ReadString() (string, error)
	ReadSymbol() (symbol string, err error)
	ReadTimeStamp(t, i uint32, err error)
	ReadUndefined() error
}

type reader struct{}

// valueReader is for reading BSON values.
type valueReader struct{}

func newValueReader(io.Reader) *valueReader { return nil }

// extJSONValueReader is for reading extended JSON.
type extJSONValueReader struct{}

func newExtJSONValueReader(io.Reader) *extJSONValueReader { return nil }

// documentValueReader is for reading *Document.
type documentValueReader struct{}

func newDocumentValueReader(*Document) *documentValueReader { return nil }

type ArrayWriter interface {
	WriteElement() (ValueWriter, error)
	WriteEndArray() error
}

type DocumentWriter interface {
	WriteElement(string) (ValueWriter, error)
	WriteEndDocument() error
}

type ValueWriter interface {
	WriteArray() (ArrayWriter, error)
	WriteBinary(b []byte) error
	WriteBinaryWithSubtype(b []byte, btype byte) error
	WriteBoolean(bool) error
	WriteCodeWithScope(code string) (DocumentWriter, error)
	WriteDBPointer(ns string, oid objectid.ObjectID) error
	WriteDateTime(dt int64) error
	WriteDecimal128(decimal.Decimal128) error
	WriteDouble(float64) error
	WriteInt32(int32) error
	WriteInt64(int64) error
	WriteJavascript(code string) error
	WriteMaxKey() error
	WriteMinKey() error
	WriteNull() error
	WriteObjectID(objectid.ObjectID) error
	WriteRegex(pattern, options string) error
	WriteString(string) error
	WriteDocument() (DocumentWriter, error)
	WriteSymbol(symbol string) error
	WriteTimestamp(t, i uint32) error
	WriteUndefined() error
}

type valueWriter struct{}
type extJSONValueWriter struct{}
type documentValueWriter struct{}

func newValueWriter(io.Writer) *valueWriter                 { return nil }
func newExtJSONWriter(io.Writer) *extJSONValueWriter        { return nil }
func newDocumentValueWriter(*Document) *documentValueWriter { return nil }

type writer struct{}

// TODO: This needs to be flushed out much more
type StructTagParser interface {
	ParseStructTags(reflect.StructField) map[string]string
}

// A Registry is used to store and retrieve codecs for types and interfaces. This type is the main
// typed passed around and Encoders and Decoders are constructed from it.
type Registry struct {
	tr *typeRegistry
	ir *interfaceRegistry

	dm Codec // default map codec
	ds Codec // default struct codec
}

func NewRegistry() *Registry { return nil }

// Register will register the provided Codec to the provided type. If the type is
// an interface, it will be registered in the interface registry. If the type is
// a pointer to or a type that is not an interface, it will be registered in the type
// registry.
func (r *Registry) Register(reflect.Type, Codec) error { return nil }

// SetDefaultMapCodec will set the Codec used when encoding or decoding a map that does
// not have another codec registered for it.
func (r *Registry) SetDefaultMapCodec(Codec) error { return nil }

// SetDefaultStructCodec will set the Codec used when encoding or decoding a struct that
// does not have another codec registered for it.
func (r *Registry) SetDefaultStructCodec(Codec) error { return nil }

// Lookup will inspect the type registry for either the type or a pointer to the type,
// if it doesn't find a codec it will inspect the interface registry for an interface
// that the type satisfies, if it doesn't find a codec there it will attempt to
// return either the default map codec or the default struct codec. If none of those
// apply, an error will be returned.
func (r *Registry) Lookup(reflect.Type) (Codec, error) { return nil, nil }

// The type registry handles codecs that are for specifics types that are not interfaces.
// This registry will handle both the types themselves and pointers to those types.
type typeRegistry struct{}

// lookup handles finding a codec for the registered type. Will return an error if no codec
// could be found.
func (tr *typeRegistry) lookup(reflect.Type) (Codec, error) { return nil, nil }

// register adds a new codec to this registry for the given type. It handles registering the
// codec for both the type and pointer to the type.
func (tr *typeRegistry) register(reflect.Type, Codec) error { return nil }

// The interface registry handles codecs that are for interface types.
type interfaceRegistry struct{}

// lookup handles finding a codec for the registered interface. Will return an error if no codec
// could be found.
func (ir *interfaceRegistry) lookup(reflect.Type) (Codec, error) { return nil, nil }

// register adds a new codec to this registry for the given interface.
func (ir *interfaceRegistry) register(reflect.Type, Codec) error { return nil }

// Marshal returns the BSON encoding of val.
//
// Marshal will use the default registry created by NewRegistry to recursively
// marshal val into a []byte. Marshal will inspect struct tags and alter the
// marshaling process accordingly.
func Marshal(val interface{}) ([]byte, error) { return nil, nil }

// MarshalAppend will append the BSON encoding of val to src. If src is not
// large enough to hold the BSON encoding of val, src will be grown.
func MarshalAppend(src []byte, val interface{}) ([]byte, error) { return nil, nil }

// MarshalWithRegistry returns the BSON encoding of val using Registry r.
func MarshalWithRegistry(r Registry, val interface{}) ([]byte, error) { return nil, nil }

// MarshalAppendWithRegistry will append the BSON encoding of val to src using
// Registry r. If src is not large enough to hold the BSON encoding of val, src
// will be grown.
func MarshalAppendWithRegistry(r Registry, src []byte, val interface{}) ([]byte, error) {
	return nil, nil
}

// Unmarshal parses the BSON-encoded data and stores the result in the value
// pointed to by val. If val is nil or not a pointer, Unmarshal returns
// InvalidUnmarshalError.
func Unmarshal(data []byte, val interface{}) error { return nil }

// UnmarshalWithRegistry parses the BSON-encoded data using Registry r and
// stores the result in the value pointed to by val. If val is nil or not
// a pointer, UnmarshalWithRegistry returns InvalidUnmarshalError.
func UnmarshalWithRegistry(r Registry, data []byte, val interface{}) error { return nil }

// SerializationType represents the serialization type being handled.
type SerializationType int

// The constants below are the serialization types available.
const (
	BSON SerializationType = iota
	ExtJSON
)

// An Encoder writes a serialization format to an output stream.
type Encoder struct{}

// NewEncoder returns a new encoder that uses Registry r to write serialization type st to w.
func NewEncoder(r *Registry, w io.Writer, st SerializationType) (*Encoder, error) { return nil, nil }

// Encode writes the BSON encoding of val to the stream.
//
// The documentation for Marshal contains details about the conversion of Go
// values to BSON.
func (e *Encoder) Encode(val interface{}) error { return nil }

// Reset will reset the state of the encoder, using the same *Registry used in
// the original construction but using w for writing with serialization type st.
func (e *Encoder) Reset(w io.Writer, st SerializationType) error { return nil }

// SetRegistry replaces the current registry of the encoder with r.
func (e *Encoder) SetRegistry(r *Registry) error { return nil }

// A Decoder reads and decodes BSON documents from a stream.
type Decoder struct{}

// NewDecoder returns a new decoder that uses Registry reg to read serialization type st from r.
func NewDecoder(reg *Registry, r io.Reader, st SerializationType) (*Decoder, error) { return nil, nil }

// Decode reads the next BSON document from the stream and decodes it into the
// value pointed to by val.
//
// The documentation for Unmarshal contains details about of BSON into a Go
// value.
func (d *Decoder) Decode(val interface{}) error { return nil }

// Reset will reset the state of the decoder, using the same *Registry used in
// the original construction but using r for reading with serialization type st.
func (d *Decoder) Reset(r io.Reader, st SerializationType) error { return nil }

// SetRegistry replaces the current registry of the decoder with r.
func (d *Decoder) SetRegistry(r *Registry) error { return nil }

// A DocumentEncoder transforms Go types into a Document.
//
// A DocumentEncoder is goroutine safe and the EncodeDocument method can be called
// from different goroutines simultaneously.
type DocumentEncoder struct{}

// NewDocumentEncoder returns a new document encoder that uses Registry r.
func NewDocumentEncoder(r *Registry) (*DocumentEncoder, error) { return nil, nil }

// EncodeDocument encodes val into a *Document.
//
// The documentation for Marshal contains details about the conversion of Go values to
// *Document.
func (de *DocumentEncoder) EncodeDocument(val interface{}) (*Document, error) { return nil, nil }

// A DocumentDecoder transforms a Document into a Go type.
//
// A DocumentDecoder is goroutine safe and the DecodeDocument method can be called
// from different goroutines simultaneously.
type DocumentDecoder struct{}

// NewDocumentDecoder returns a new document decoder that uses Registry r.
func NewDocumentDecoder(r *Registry) (*DocumentDecoder, error) { return nil, nil }

// DecodeDocument decodes d into val.
//
// The documentation for Unmarshal contains details about the conversion of a *Document into
// a Go value.
func (dd *DocumentDecoder) DecodeDocument(d *Document, val interface{}) error { return nil }
