package bson

import (
	"bytes"
	"errors"
	"io"
	"sort"

	"github.com/10gen/mongo-go-driver/bson/elements"
)

// ErrInvalidReadOnlyDocument indicates that the underlying bytes of a bson.Reader are invalid.
var ErrInvalidReadOnlyDocument = errors.New("invalid read-only document")

// ErrInvalidKey indicates that the BSON representation of a key is missing a null terminator.
var ErrInvalidKey = errors.New("invalid document key")

// ErrInvalidLength indicates that a length in a binary representation of a BSON document is invalid.
var ErrInvalidLength = errors.New("document length is invalid")

// ErrEmptyKey indicates that no key was provided to a Lookup method.
var ErrEmptyKey = errors.New("empty key provided")

// ErrNilElement indicates that a nil element was provided when none was expected.
var ErrNilElement = errors.New("element is nil")

// ErrNilDocument indicates that an operation was attempted on a nil *bson.Document.
var ErrNilDocument = errors.New("document is nil")

// ErrInvalidDocumentType indicates that a type which doesn't represent a BSON document was
// was provided when a document was expected.
var ErrInvalidDocumentType = errors.New("invalid document type")

// ErrInvalidDepthTraversal indicates that a provided path of keys to a nested value in a document
// does not exist.
//
// TODO(skriptble): This error message is pretty awful.
// Please fix.
var ErrInvalidDepthTraversal = errors.New("invalid depth traversal")

// ErrElementNotFound indicates that an Element matching a certain condition does not exist.
var ErrElementNotFound = errors.New("element not found")

// ErrOutOfBounds indicates that an index provided to access something was invalid.
var ErrOutOfBounds = errors.New("out of bounds")

// Document is a mutable ordered map that compactly represents a BSON document.
type Document struct {
	// The default behavior or Append, Prepend, and Replace is to panic on the
	// insertion of a nil element. Setting IgnoreNilInsert to true will instead
	// silently ignore any nil parameters to these methods.
	IgnoreNilInsert bool
	elems           []*Element
	index           []uint32
}

// NewDocument creates an empty Document. The numberOfElems parameter will
// preallocate the underlying storage which can prevent extra allocations.
func NewDocument(elems ...*Element) *Document {
	doc := &Document{
		elems: make([]*Element, 0, len(elems)),
		index: make([]uint32, 0, len(elems)),
	}

	doc.Append(elems...)

	return doc
}

// ReadDocument will create a Document using the provided slice of bytes. If the
// slice of bytes is not a valid BSON document, this method will return an error.
func ReadDocument(b []byte) (*Document, error) {
	var doc = new(Document)
	err := doc.UnmarshalBSON(b)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

// Len returns the number of elements in the document.
func (d *Document) Len() int {
	return len(d.elems)
}

// Keys returns all of the element keys for this document. If recursive is true,
// this method will also return the keys of any subdocuments or arrays.
func (d *Document) Keys(recursive bool) (Keys, error) {
	return d.recursiveKeys(recursive)
}

// recursiveKeys handles the recursion case for retrieving all of a Document's
// keys.
func (d *Document) recursiveKeys(recursive bool, prefix ...string) (Keys, error) {
	ks := make(Keys, 0, len(d.elems))
	for _, elem := range d.elems {
		key := elem.Key()
		ks = append(ks, Key{Prefix: prefix, Name: key})
		if !recursive {
			continue
		}
		// TODO(skriptble): Should we support getting the keys of a code with
		// scope document?
		switch elem.value.Type() {
		case '\x03':
			subprefix := append(prefix, key)
			subkeys, err := elem.value.MutableDocument().recursiveKeys(recursive, subprefix...)
			if err != nil {
				return nil, err
			}
			ks = append(ks, subkeys...)
		case '\x04':
			subprefix := append(prefix, key)
			subkeys, err := elem.value.MutableArray().doc.recursiveKeys(recursive, subprefix...)
			if err != nil {
				return nil, err
			}
			ks = append(ks, subkeys...)
		}
	}
	return ks, nil
}

// Append adds each element to the end of the document, in order. If a nil element is passed
// as a parameter this method will panic. To change this behavior to silently
// ignore a nil element, set IgnoreNilInsert to true on the Document.
//
// If a nil element is inserted and this method panics, it does not remove the
// previously added elements.
func (d *Document) Append(elems ...*Element) *Document {
	for _, elem := range elems {
		if elem == nil {
			if d.IgnoreNilInsert {
				continue
			}
			// TODO(skriptble): Maybe Append and Prepend should return an error
			// instead of panicking here.
			panic(ErrNilElement)
		}
		d.elems = append(d.elems, elem)
		i := sort.Search(len(d.index), func(i int) bool {
			return bytes.Compare(
				d.keyFromIndex(i), elem.value.data[elem.value.start+1:elem.value.offset]) >= 0
		})
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

// Prepend adds each element to the beginning of the document, in order. If a nil element is passed
// as a parameter this method will panic. To change this behavior to silently
// ignore a nil element, set IgnoreNilInsert to true on the Document.
//
// If a nil element is inserted and this method panics, it does not remove the
// previously added elements.
func (d *Document) Prepend(elems ...*Element) *Document {
	// In order to insert the prepended elements in order we need to make space
	// at the front of the elements slice.
	d.elems = append(d.elems, elems...)
	copy(d.elems[len(elems):], d.elems)

	remaining := len(elems)
	for idx, elem := range elems {
		if elem == nil {
			if d.IgnoreNilInsert {
				// Having nil elements in a document would be problematic.
				copy(d.elems[idx:], d.elems[idx+1:])
				d.elems[len(d.elems)-1] = nil
				d.elems = d.elems[:len(d.elems)-1]
				continue
			}
			// Not very efficient, but we're about to blow up so ¯\_(ツ)_/¯
			for j := idx; j < remaining; j++ {
				copy(d.elems[j:], d.elems[j+1:])
				d.elems[len(d.elems)-1] = nil
				d.elems = d.elems[:len(d.elems)-1]
			}
			panic(ErrNilElement)
		}
		remaining--
		d.elems[idx] = elem
		i := sort.Search(len(d.index), func(i int) bool {
			return bytes.Compare(
				d.keyFromIndex(i), elem.value.data[elem.value.start+1:elem.value.offset]) >= 0
		})
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

// Set replaces an element of a document. If an element with a matching key is
// found, the element will be replaced with the one provided. If the document
// does not have an element with that key, the element is appended to the
// document instead. If a nil element is passed as a parameter this method will
// panic. To change this behavior to silently ignore a nil element, set
// IgnoreNilInsert to true on the Document.
//
// If a nil element is inserted and this method panics, it does not remove the
// previously added elements.
//
// TODO(skriptble): Do we need to panic on a nil element? Semantically, if you
// ask to replace an element in the document with a nil element, you aren't
// asking for anything to be done.
func (d *Document) Set(elem *Element) *Document {
	if elem == nil {
		if d.IgnoreNilInsert {
			return d
		}
		panic(ErrNilElement)
	}

	key := elem.Key() + "\x00"
	i := sort.Search(len(d.index), func(i int) bool { return bytes.Compare(d.keyFromIndex(i), []byte(key)) >= 0 })
	if i < len(d.index) && bytes.Compare(d.keyFromIndex(i), []byte(key)) == 0 {
		d.elems[i] = elem
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

	return d
}

// Lookup searches the document and potentially subdocuments or arrays for the
// provided key. Each key provided to this method represents a layer of depth.
func (d *Document) Lookup(key ...string) (*Element, error) {
	if len(key) == 0 {
		return nil, ErrEmptyKey
	}
	var elem *Element
	var err error
	first := []byte(key[0] + "\x00")
	i := sort.Search(len(d.index), func(i int) bool { return bytes.Compare(d.keyFromIndex(i), first) >= 0 })
	if i < len(d.index) && bytes.Compare(d.keyFromIndex(i), first) == 0 {
		elem = d.elems[d.index[i]]
		if len(key) == 1 {
			return elem, nil
		}
		switch elem.value.Type() {
		case '\x03':
			elem, err = elem.value.MutableDocument().Lookup(key[1:]...)
		case '\x04':
			elem, err = elem.value.MutableArray().doc.Lookup(key[1:]...)
		default:
			// TODO(skriptble): This error message should be more clear, e.g.
			// include information about what depth was reached, what the
			// incorrect type was, etc...
			err = ErrInvalidDepthTraversal
		}
	}
	if err != nil {
		return nil, err
	}
	if elem == nil {
		// TODO(skriptble): This should also be a clearer error message.
		// Preferably we should track the depth at which the key was not found.
		return nil, ErrElementNotFound
	}
	return elem, nil
}

// Delete removes the keys from the Document. The deleted element is
// returned. If the key does not exist, then nil is returned and the delete is
// a no-op. The same is true if something along the depth tree does not exist
// or is not a traversable type.
//
// TODO(skriptble): This method differs from Lookup when it comes to errors.
// Should this method return errors to be consistent?
func (d *Document) Delete(key ...string) *Element {
	if len(key) == 0 {
		return nil
	}
	// Do a binary search through the index, delete the element from
	// the index and delete the element from the elems array.
	var elem *Element
	first := []byte(key[0] + "\x00")
	i := sort.Search(len(d.index), func(i int) bool { return bytes.Compare(d.keyFromIndex(i), first) >= 0 })
	if i < len(d.index) && bytes.Compare(d.keyFromIndex(i), first) == 0 {
		keyIndex := d.index[i]
		elem = d.elems[keyIndex]
		if len(key) == 1 {
			d.index = append(d.index[:i], d.index[i+1:]...)
			d.elems = append(d.elems[:keyIndex], d.elems[keyIndex+1:]...)
			return elem
		}
		switch elem.value.Type() {
		case '\x03':
			elem = elem.value.MutableDocument().Delete(key[1:]...)
		case '\x04':
			elem = elem.value.MutableArray().doc.Delete(key[1:]...)
		default:
			elem = nil
		}
	}
	return elem
}

// ElementAt retrieves the element at the given index in a Document.
//
// TODO(skriptble): This method could be variadic and return the element at the
// provided depth.
func (d *Document) ElementAt(index uint) (*Element, error) {
	if int(index) >= len(d.elems) {
		return nil, ErrOutOfBounds
	}
	return d.elems[index], nil
}

// Iterator creates an Iterator for this document and returns it.
func (d *Document) Iterator() *Iterator {
	return newIterator(d)
}

// Concat will take the keys from the provided document and concat them onto
// the end of this document.
//
// doc must be one of the following:
//
//   - *Document
//   - []byte
//   - io.Reader
func (d *Document) Concat(docs ...interface{}) error {
	for _, doc := range docs {
		if doc == nil {
			if d.IgnoreNilInsert {
				continue
			}

			return ErrNilDocument
		}

		switch doc := doc.(type) {
		case *Document:
			if doc == nil {
				if d.IgnoreNilInsert {
					continue
				}

				return ErrNilDocument
			}
			d.Append(doc.elems...)
		case []byte:
			if err := d.concatReader(Reader(doc)); err != nil {
				return err
			}
		case Reader:
			if err := d.concatReader(doc); err != nil {
				return err
			}
		default:
			return ErrInvalidDocumentType
		}
	}

	return nil
}

func (d *Document) concatReader(r Reader) error {
	_, err := r.readElements(func(e *Element) error {
		d.Append(e)

		return nil
	})

	return err
}

// Reset clears a document so it can be reused. This method clears references
// to the underlying pointers to elements so they can be garbage collected.
func (d *Document) Reset() {
	for idx := range d.elems {
		d.elems[idx] = nil
	}
	d.elems = d.elems[:0]
	d.index = d.index[:0]
}

// Validate validates the document and returns its total size.
func (d *Document) Validate() (uint32, error) {
	// Header and Footer
	var size uint32 = 4 + 1
	for _, elem := range d.elems {
		n, err := elem.Validate()
		if err != nil {
			return 0, err
		}
		size += n
	}
	return size, nil
}

// validates the document and returns its total size. This method has
// bookkeeping parameters to prevent a stack overflow.
func (d *Document) validate(currentDepth, maxDepth uint32) (uint32, error) {
	return 0, nil
}

// WriteTo implements the io.WriterTo interface.
//
// TODO(skriptble): We can optimize this by having creating implementations of
// writeByteSlice that write directly to an io.Writer instead.
func (d *Document) WriteTo(w io.Writer) (int64, error) {
	b, err := d.MarshalBSON()
	if err != nil {
		return 0, err
	}
	n, err := w.Write(b)
	return int64(n), err
}

// WriteDocument will serialize this document to the provided writer beginning
// at the provided start position.
func (d *Document) WriteDocument(start uint, writer interface{}) (int64, error) {
	var total int64
	var pos = start
	size, err := d.Validate()
	if err != nil {
		return total, err
	}
	switch w := writer.(type) {
	case []byte:
		n, err := d.writeByteSlice(pos, size, w)
		total += n
		pos += uint(n)
		if err != nil {
			return total, err
		}
	default:
		return 0, ErrInvalidWriter
	}
	return total, nil
}

// writeByteSlice handles serializing this document to a slice of bytes starting
// at the given start position.
func (d *Document) writeByteSlice(start uint, size uint32, b []byte) (int64, error) {
	var total int64
	var pos = start
	if len(b) < int(start)+int(size) {
		return 0, ErrTooSmall
	}
	n, err := elements.Int32.Encode(start, b, int32(size))
	total += int64(n)
	pos += uint(n)
	if err != nil {
		return total, err
	}
	for _, elem := range d.elems {
		n, err := elem.writeElement(true, pos, b)
		total += int64(n)
		pos += uint(n)
		if err != nil {
			return total, err
		}
	}

	n, err = elements.Byte.Encode(pos, b, '\x00')
	total += int64(n)
	pos += uint(n)
	if err != nil {
		return total, err
	}
	return total, nil
}

// MarshalBSON implements the Marshaler interface.
func (d *Document) MarshalBSON() ([]byte, error) {
	size, err := d.Validate()

	if err != nil {
		return nil, err
	}
	b := make([]byte, size)
	_, err = d.writeByteSlice(0, size, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// UnmarshalBSON implements the Unmarshaler interface.
func (d *Document) UnmarshalBSON(b []byte) error {
	// Read byte array
	//   - Create an Element for each element found
	//   - Update the index with the key of the element
	//   TODO: Maybe do 2 pass and alloc the elems and index once?
	// 		   We should benchmark 2 pass vs multiple allocs for growing the slice
	_, err := Reader(b).readElements(func(elem *Element) error {
		d.elems = append(d.elems, elem)
		i := sort.Search(len(d.index), func(i int) bool {
			return bytes.Compare(
				d.keyFromIndex(i), elem.value.data[elem.value.start+1:elem.value.offset]) >= 0
		})
		if i < len(d.index) {
			d.index = append(d.index, 0)
			copy(d.index[i+1:], d.index[i:])
			d.index[i] = uint32(len(d.elems) - 1)
		} else {
			d.index = append(d.index, uint32(len(d.elems)-1))
		}
		return nil
	})
	return err
}

// ReadFrom will read one BSON document from the given io.Reader.
func (d *Document) ReadFrom(r io.Reader) (int64, error) {
	var total int64
	sizeBuf := make([]byte, 4)
	n, err := io.ReadFull(r, sizeBuf)
	total += int64(n)
	if err != nil {
		return total, err
	}
	givenLength := readi32(sizeBuf)
	b := make([]byte, givenLength)
	copy(b[0:4], sizeBuf)
	n, err = io.ReadFull(r, b[4:])
	total += int64(n)
	if err != nil {
		return total, err
	}
	err = d.UnmarshalBSON(b)
	return total, err
}

// keyFromIndex returns the key for the element. The idx parameter is the
// position in the index property, not the elems property. This method is
// mainly used when calling sort.Search.
func (d *Document) keyFromIndex(idx int) []byte {
	haystack := d.elems[d.index[idx]]
	return haystack.value.data[haystack.value.start+1 : haystack.value.offset]
}
