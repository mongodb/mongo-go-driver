package mongoopt

import "github.com/mongodb/mongo-go-driver/core/option"

// Collation allows users to specify language-specific rules for string comparison, such as
// rules for lettercase and accent marks.
type Collation struct {
	Locale          string `bson:",omitempty"`
	CaseLevel       bool   `bson:",omitempty"`
	CaseFirst       string `bson:",omitempty"`
	Strength        int    `bson:",omitempty"`
	NumericOrdering bool   `bson:",omitempty"`
	Alternate       string `bson:",omitempty"`
	MaxVariable     string `bson:",omitempty"`
	Backwards       bool   `bson:",omitempty"`
}

// Convert changes a Collation instance into a core options Collation.
func (c *Collation) Convert() *option.Collation {
	return &option.Collation{
		Locale:          c.Locale,
		CaseLevel:       c.CaseLevel,
		CaseFirst:       c.CaseFirst,
		Strength:        c.Strength,
		NumericOrdering: c.NumericOrdering,
		Alternate:       c.Alternate,
		MaxVariable:     c.MaxVariable,
		Backwards:       c.Backwards,
	}
}

// CursorType specifies whether a cursor should close when the last data is retrieved. See
// NonTailable, Tailable, and TailableAwait.
type CursorType int8

const (
	// NonTailable specifies that a cursor should close after retrieving the last data.
	NonTailable CursorType = iota
	// Tailable specifies that a cursor should not close when the last data is retrieved.
	Tailable
	// TailableAwait specifies that a cursor should not close when the last data is retrieved and
	// that it should block for a certain amount of time for new data before returning no data.
	TailableAwait
)

// ReturnDocument specifies whether a findAndUpdate operation should return the document as it was
// before the update or as it is after the update.
type ReturnDocument int8

const (
	// Before specifies that findAndUpdate should return the document as it was before the update.
	Before ReturnDocument = iota
	// After specifies that findAndUpdate should return the document as it is after the update.
	After
)

// FullDocument specifies whether a change stream should include a copy of the entire document that was changed from
// some time after the change occurred.
type FullDocument string

const (
	// Default does not include a document copy
	Default FullDocument = "default"
	// UpdateLookup includes a delta describing the changes to the document and a copy of the entire document that
	// was changed
	UpdateLookup FullDocument = "updateLookup"
)
