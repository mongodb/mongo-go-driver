package options

import (
	"github.com/mongodb/mongo-go-driver/x/bsonx"
	"reflect"
	"testing"
)

func TestCollation(t *testing.T) {
	t.Run("TestCollationToDocument", func(t *testing.T) {
		c := &Collation{
			Locale:          "locale",
			CaseLevel:       true,
			CaseFirst:       "first",
			Strength:        1,
			NumericOrdering: true,
			Alternate:       "alternate",
			MaxVariable:     "maxVariable",
			Normalization:   true,
			Backwards:       true,
		}

		doc := c.ToDocument()
		expected := bsonx.Doc{
			{"locale", bsonx.String("locale")},
			{"caseLevel", bsonx.Boolean(true)},
			{"caseFirst", bsonx.String("first")},
			{"strength", bsonx.Int32(1)},
			{"numericOrdering", bsonx.Boolean(true)},
			{"alternate", bsonx.String("alternate")},
			{"maxVariable", bsonx.String("maxVariable")},
			{"normalization", bsonx.Boolean(true)},
			{"backwards", bsonx.Boolean(true)},
		}

		if !reflect.DeepEqual(doc, expected) {
			t.Fatalf("collation did not match expected. got %v; wanted %v", doc, expected)
		}
	})
}
