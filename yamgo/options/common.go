package options

// OptArrayFilters is for internal use.
type OptArrayFilters []interface{}

// OptBypassDocumentValidation is for internal use.
type OptBypassDocumentValidation bool

// OptCollation is for internal use.
type OptCollation struct{ Collation *CollationOptions }

// OptOrdered is for internal use.
type OptOrdered bool

// OptUpsert is for internal use.
type OptUpsert bool

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
