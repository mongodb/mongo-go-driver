package option

import "github.com/mongodb/mongo-go-driver/options-design/bson"

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

// MarshalBSONDocument implements the bson.DocumentMarshaler interface.
func (co *Collation) MarshalBSONDocument() (*bson.Document, error) {
	return nil, nil
}
