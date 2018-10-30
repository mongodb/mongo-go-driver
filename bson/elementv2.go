package bson

import "fmt"

// Element represents a BSON element.
//
// NOTE: Element cannot be the value of a map nor a property of a struct without special handling.
// The default encoders and decoders will not process Element correctly. To do so would require
// information loss since an Element contains a key, but the keys used when encoding a struct are
// the struct field names. Instead of using an Element, use a Value as a value in a map or a
// property of a struct.
type Elementv2 struct {
	Key   string
	Value Valuev2
}

// Equal compares e and e2 and returns true if they are equal.
func (e Elementv2) Equal(e2 Elementv2) bool {
	if e.Key != e2.Key {
		return false
	}
	return e.Value.Equal(e2.Value)
}

func (e Elementv2) String() string {
	// TODO(GODRIVER-612): When bsoncore has appenders for extended JSON use that here.
	return fmt.Sprintf(`bson.Element{"%s": %v}`, e.Key, e.Value)
}
