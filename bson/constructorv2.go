package bson

import (
	"encoding/binary"
	"math"

	"github.com/mongodb/mongo-go-driver/bson/bsontype"
)

var _ Embeddable = (*Document)(nil)
var _ Embeddable = (*Array)(nil)

// Embeddable is the interface implemented by types that can be embedded into a Value. There are
// only two implementors of this type Document and Array.
type Embeddable interface {
	embed()
}

// Double constructs a BSON double Value.
func Double(f64 float64) Valuev2 {
	v := Valuev2{t: bsontype.Double}
	binary.LittleEndian.PutUint64(v.bootstrap[0:8], math.Float64bits(f64))
	return v
}

// String constructs a BSON string Value.
func String(str string) Valuev2 {
	v := Valuev2{t: bsontype.String}
	switch {
	case len(str) < 16:
		copy(v.bootstrap[:], str)
	default:
		v.primitive = str
	}
	return v
}

// Embed constructs a Value from the given Embeddable. The type will be a BSON embedded document for
// *Document, a BSON array for *Array, and a BSON null for either a nil pointer to *Document,
// *Array, or the value nil.
func Embed(embed Embeddable) Valuev2 {
	var v Valuev2
	switch tt := embed.(type) {
	case *Document:
		if tt == nil {
			v.t = bsontype.Null
			break
		}
		v.t = bsontype.EmbeddedDocument
		v.primitive = tt
	case *Array:
		if tt == nil {
			v.t = bsontype.Null
			break
		}
		v.t = bsontype.Array
		v.primitive = tt
	default:
		v.t = bsontype.Null
	}
	return v
}
