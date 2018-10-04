package bson

import "github.com/mongodb/mongo-go-driver/bson/bsoncodec"

// DefaultRegistry is the default bsoncodec.Registry. It contains the default codecs and the
// primitive codecs.
var DefaultRegistry *bsoncodec.Registry = NewRegistryBuilder().Build()

// NewRegistryBuilder creates a new RegistryBuilder configured with the default encoders and
// deocders from the bsoncodec.DefaultValueEncoders and bsoncodec.DefaultValueDecoders types and the
// PrimitiveCodecs type in this package.
func NewRegistryBuilder() *bsoncodec.RegistryBuilder {
	return primitiveCodecs.RegisterPrimitiveCodecs(nil)
}
