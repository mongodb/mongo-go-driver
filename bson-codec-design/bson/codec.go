package bson

type Codec interface {
	EncodeValue(Registry, ValueWriter, interface{}) error
	DecodeValue(Registry, ValueReader, interface{}) error
}

type CodecZeroer interface {
	Codec
	IsZero(interface{}) bool
}
