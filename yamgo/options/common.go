package options

// CollationOptions allows users to specify language-specific rules for string comparison, such as
// rules for lettercase and accent marks.
type CollationOptions struct {
	Locale          string ",omitempty"
	CaseLevel       bool   ",omitempty"
	CaseFirst       string ",omitempty"
	Strength        int    ",omitempty"
	NumericOrdering bool   ",omitempty"
	Alternate       string ",omitempty"
	MaxVariable     string ",omitempty"
	Backwards       bool   ",omitempty"
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
