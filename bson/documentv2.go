package bson

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"

	"github.com/mongodb/mongo-go-driver/bson/bsoncore"
	"github.com/mongodb/mongo-go-driver/bson/bsontype"
)

// KeyNotFound is an error type returned from the Lookup methods on Document. This type contains
// information about which key was not found and if it was actually not found or if a component of
// the key except the last was not a document nor array.
type KeyNotFound struct {
	Key   []string      // The keys that were searched for.
	Depth uint          // Which key either was not found or was an incorrect type.
	Type  bsontype.Type // The type of the key that was found but was an incorrect type.
}

func (knf KeyNotFound) Error() string {
	depth := knf.Depth
	if depth >= uint(len(knf.Key)) {
		depth = uint(len(knf.Key)) - 1
	}

	if len(knf.Key) == 0 {
		return "no keys were provided for lookup"
	}

	if knf.Type != bsontype.Type(0) {
		return fmt.Sprintf(`key "%s" was found but was not valid to traverse BSON type %s`, knf.Key[depth], knf.Type)
	}

	return fmt.Sprintf(`key "%s" was not found`, knf.Key[depth])
}

type Documentv2 struct {
	elems []Elementv2
	index []uint32
}

// Document is a mutable ordered map that represents a BSON document.
func NewDocumentv2(elems ...Elementv2) *Documentv2 {
	doc := &Documentv2{
		elems: make([]Elementv2, 0, len(elems)),
		index: make([]uint32, 0, len(elems)),
	}
	doc.AppendElements(elems...)
	return doc
}

// ReadDocument will create a Document using the provided slice of bytes. If the
// slice of bytes is not a valid BSON document, this method will return an error.
func ReadDocumentv2(b []byte) (*Documentv2, error) {
	var doc = new(Documentv2)
	err := doc.UnmarshalBSON(b)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

// Copy makes a shallow copy of this document.
func (d *Documentv2) Copy() *Documentv2 {
	if d == nil {
		return nil
	}

	doc := &Documentv2{
		elems: make([]Elementv2, len(d.elems), cap(d.elems)),
		index: make([]uint32, len(d.index), cap(d.index)),
	}

	copy(doc.elems, d.elems)
	copy(doc.index, d.index)

	return doc
}

// Len returns the number of elements in the document.
func (d *Documentv2) Len() int {
	if d == nil {
		return 0
	}

	return len(d.elems)
}

// Append adds an element to the end of the document, creating it from the key and value provided.
func (d *Documentv2) Append(key string, val Valuev2) *Documentv2 {
	return d.AppendElements(Elementv2{Key: key, Value: val})
}

// AppendElements adds each element to the end of the document, in order.
func (d *Documentv2) AppendElements(elems ...Elementv2) *Documentv2 {
	if d == nil {
		d = &Documentv2{elems: make([]Elementv2, 0, len(elems)), index: make([]uint32, 0, len(elems))}
	}

	for _, elem := range elems {
		d.elems = append(d.elems, elem)
		i := sort.Search(len(d.index), func(i int) bool { return d.elems[d.index[i]].Key >= elem.Key })
		if i < len(d.index) {
			d.index = append(d.index, 0)
			copy(d.index[i+1:], d.index[i:])
			d.index[i] = uint32(len(d.elems) - 1)
		} else {
			d.index = append(d.index, uint32(len(d.elems)-1))
		}
	}
	return d
}

// Prepend adds an element to the beginning of the document, creating it from the key and value provided.
func (d *Documentv2) Prepend(key string, val Valuev2) *Documentv2 {
	return d.PrependElements(Elementv2{Key: key, Value: val})
}

// PrependElements adds each element to the beginning of the document, in order.
func (d *Documentv2) PrependElements(elems ...Elementv2) *Documentv2 {
	if d == nil {
		d = &Documentv2{elems: make([]Elementv2, 0, len(elems)), index: make([]uint32, 0, len(elems))}
	}

	// In order to insert the prepended elements in order we need to make space
	// at the front of the elements slice.
	d.elems = append(d.elems, elems...)
	copy(d.elems[len(elems):], d.elems)

	for idx, elem := range elems {
		d.elems[idx] = elem
		for idx := range d.index {
			d.index[idx]++
		}
		i := sort.Search(len(d.index), func(i int) bool { return d.elems[d.index[i]].Key >= elem.Key })
		if i < len(d.index) {
			d.index = append(d.index, 0)
			copy(d.index[i+1:], d.index[i:])
			d.index[i] = 0
		} else {
			d.index = append(d.index, 0)
		}
	}
	return d
}

// Set replaces an element of a document. If an element with a matching key is
// found, the element will be replaced with the one provided. If the document
// does not have an element with that key, the element is appended to the
// document instead.
func (d *Documentv2) Set(key string, val Valuev2) *Documentv2 {
	return d.SetElements(Elementv2{Key: key, Value: val})
}

// SetElements replaces elements in a document. If an element with a matching key is
// found, the element will be replaced with the one provided. If the document
// does not have an element with that key, the element is appended to the
// document instead.
func (d *Documentv2) SetElements(elems ...Elementv2) *Documentv2 {
	if d == nil {
		d = &Documentv2{elems: make([]Elementv2, 0, len(elems)), index: make([]uint32, 0, len(elems))}
	}

	for _, elem := range elems {
		i := sort.Search(len(d.index), func(i int) bool { return d.elems[d.index[i]].Key >= elem.Key })
		if i < len(d.index) && d.elems[d.index[i]].Key == elem.Key {
			d.elems[d.index[i]] = elem
			return d
		}

		d.elems = append(d.elems, elem)
		position := uint32(len(d.elems) - 1)
		if i < len(d.index) {
			d.index = append(d.index, 0)
			copy(d.index[i+1:], d.index[i:])
			d.index[i] = position
		} else {
			d.index = append(d.index, position)
		}
	}

	return d
}

// Lookup searches the document and potentially subdocuments or arrays for the
// provided key. Each key provided to this method represents a layer of depth.
//
// This method will return an empty Value if they key does not exist. To know if they key actually
// exists, use LookupErr.
func (d *Documentv2) Lookup(key ...string) Valuev2 {
	val, _ := d.LookupErr(key...)
	return val
}

// LookupErr searches the document and potentially subdocuments or arrays for the
// provided key. Each key provided to this method represents a layer of depth.
func (d *Documentv2) LookupErr(key ...string) (Valuev2, error) {
	elem, err := d.LookupElementErr(key...)
	return elem.Value, err
}

// LookupElement searches the document and potentially subdocuments or arrays for the
// provided key. Each key provided to this method represents a layer of depth.
//
// This method will return an empty Element if they key does not exist. To know if they key actually
// exists, use LookupElementErr.
func (d *Documentv2) LookupElement(key ...string) Elementv2 {
	elem, _ := d.LookupElementErr(key...)
	return elem
}

// LookupElementErr searches the document and potentially subdocuments or arrays for the
// provided key. Each key provided to this method represents a layer of depth.
func (d *Documentv2) LookupElementErr(key ...string) (Elementv2, error) {
	// KeyNotFound operates by being created where the error happens and then the depth is
	// incremented by 1 as each function unwinds. Whenever this function returns, it also assigns
	// the Key slice to the key slice it has. This ensures that the proper depth is identified and
	// the proper keys.
	if d == nil {
		return Elementv2{}, KeyNotFound{Key: key}
	}

	if len(key) == 0 {
		return Elementv2{}, KeyNotFound{Key: key}
	}

	var elem Elementv2
	var err error
	i := sort.Search(len(d.index), func(i int) bool { return d.elems[d.index[i]].Key >= key[0] })
	if i < len(d.index) && d.elems[d.index[i]].Key == key[0] {
		elem = d.elems[d.index[i]]
		if len(key) == 1 {
			return elem, nil
		}
		switch elem.Value.Type() {
		case bsontype.EmbeddedDocument:
			elem, err = elem.Value.Document().LookupElementErr(key[1:]...)
		case bsontype.Array:
			var index uint64
			index, err = strconv.ParseUint(key[1], 10, 0)
			if err != nil {
				err = KeyNotFound{Key: key}
				break
			}

			var val Valuev2
			// TODO: THIS IS WRONG! We need lookupTraverse to return an Element.
			_, err = elem.Value.Array().lookupTraverse(uint(index), key[2:]...)
			elem = Elementv2{Key: key[1], Value: val}
		default:
			err = KeyNotFound{Type: elem.Value.Type()}
		}
	}
	switch tt := err.(type) {
	case KeyNotFound:
		tt.Depth++
		tt.Key = key
		return Elementv2{}, tt
	case nil:
		return elem, nil
	default:
		return Elementv2{}, err
	}
}

// Delete removes the keys from the Document. The deleted element is
// returned. If the key does not exist, then an empty Element is returned and the delete is
// a no-op. The same is true if something along the depth tree does not exist
// or is not a traversable type.
func (d *Documentv2) Delete(key ...string) Elementv2 {
	if d == nil {
		return Elementv2{}
	}

	if len(key) == 0 {
		return Elementv2{}
	}
	// Do a binary search through the index, delete the element from
	// the index and delete the element from the elems array.
	var elem Elementv2
	i := sort.Search(len(d.index), func(i int) bool { return d.elems[d.index[i]].Key >= key[0] })
	if i < len(d.index) && d.elems[d.index[i]].Key == key[0] {
		keyIndex := d.index[i]
		elem = d.elems[keyIndex]
		if len(key) == 1 {
			d.index = append(d.index[:i], d.index[i+1:]...)
			d.elems = append(d.elems[:keyIndex], d.elems[keyIndex+1:]...)
			for j := range d.index {
				if d.index[j] > keyIndex {
					d.index[j]--
				}
			}
			return elem
		}
		switch elem.Value.Type() {
		case bsontype.EmbeddedDocument:
			elem = elem.Value.Document().Delete(key[1:]...)
		case bsontype.Array:
			// TODO: Array needs to have a Delete.
			_ = elem.Value.Array().doc.Delete(key[1:]...)
		default:
			elem = Elementv2{}
		}
	}
	return elem
}

// Index retrieves the element at the given index in a Document. It panics if the index is
// out-of-bounds or if d is nil.
func (d *Documentv2) Index(index uint) Elementv2 {
	return d.elems[index]
}

// IndexOK is the same as Index, but returns a boolean instead of panicking.
func (d *Documentv2) IndexOK(index uint) (Elementv2, bool) {
	if d == nil {
		return Elementv2{}, false
	}

	if index >= uint(len(d.elems)) {
		return Elementv2{}, false
	}

	return d.Index(index), true
}

func (d *Documentv2) Elements() []Elementv2 {
	if d == nil {
		return nil
	}
	elems := make([]Elementv2, 0, len(d.elems))
	for _, elem := range d.elems {
		elems = append(elems, elem)
	}
	return elems
}

// Reset clears a document so it can be reused.
func (d *Documentv2) Reset() {
	if d == nil {
		return
	}

	for idx := range d.elems {
		d.elems[idx].Key = ""
		d.elems[idx].Value = d.elems[idx].Value.reset()
	}
	d.elems = d.elems[:0]
	d.index = d.index[:0]
}

// MarshalBSON implements the Marshaler interface.
func (d *Documentv2) MarshalBSON() ([]byte, error) { return d.AppendMarshalBSON(nil) }

func (d *Documentv2) AppendMarshalBSON(dst []byte) ([]byte, error) {
	if d == nil {
		return dst, ErrNilDocument
	}

	idx, dst := bsoncore.AppendDocumentStart(dst)
	for _, elem := range d.elems {
		t, data, err := elem.Value.MarshalBSONValue()
		if err != nil {
			// TODO: Maybe we should be more descriptive, e.g. which element failed to marshal,
			// which index it's at, etc...
			return nil, err
		}
		dst = append(dst, byte(t))
		dst = append(dst, elem.Key...)
		dst = append(dst, 0x00)
		dst = append(dst, data...)
	}
	dst, err := bsoncore.AppendDocumentEnd(dst, idx)
	if err != nil {
		return nil, err
	}
	return dst, nil
}

// UnmarshalBSON implements the Unmarshaler interface.
func (d *Documentv2) UnmarshalBSON(b []byte) error {
	if d == nil {
		return ErrNilDocument
	}

	if err := Raw(b).Validate(); err != nil {
		return err
	}

	elems, err := Raw(b).Elements()
	if err != nil {
		return err
	}
	for _, _ = range elems {
	}
	return nil
}

// Equal compares this document to another, returning true if they are equal.
func (d *Documentv2) Equal(d2 *Documentv2) bool {
	if d == nil && d2 == nil {
		return true
	}

	if d == nil || d2 == nil {
		return false
	}

	if (len(d.elems) != len(d2.elems)) || (len(d.index) != len(d2.index)) {
		return false
	}
	for index := range d.elems {
		b := d.elems[index].Equal(d2.elems[index])
		if !b {
			return false
		}

		if d.index[index] != d2.index[index] {
			return false
		}
	}
	return true
}

// String implements the fmt.Stringer interface.
func (d *Documentv2) String() string {
	if d == nil {
		return "<nil>"
	}

	var buf bytes.Buffer
	buf.Write([]byte("bson.Document{"))
	for idx, elem := range d.elems {
		if idx > 0 {
			buf.Write([]byte(", "))
		}
		fmt.Fprintf(&buf, "%s", elem)
	}
	buf.WriteByte('}')

	return buf.String()
}

// embed implements the Embeddable interface.
func (d *Documentv2) embed() {}
