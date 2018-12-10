package driver

import (
	"github.com/mongodb/mongo-go-driver/x/bsonx"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestListCollections(t *testing.T) {
	dbName := "db"
	noNameFilter := bsonx.Doc{
		{"foo", bsonx.String("bar")},
	}
	nonStringFilter := bsonx.Doc{
		{"name", bsonx.Int32(1)},
	}
	nameFilter := bsonx.Doc{
		{"name", bsonx.String("coll")},
	}
	modifiedFilter := bsonx.Doc{
		{"name", bsonx.String(dbName + ".coll")},
	}

	t.Run("TestTransformFilter", func(t *testing.T) {
		testCases := []struct {
			name           string
			filter         bsonx.Doc
			expectedFilter bsonx.Doc
			err            error
		}{
			{"TestNilFilter", nil, nil, nil},
			{"TestNoName", noNameFilter, noNameFilter, nil},
			{"TestNonStringName", nonStringFilter, nil, ErrFilterType},
			{"TestName", nameFilter, modifiedFilter, nil},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				newFilter, err := transformFilter(tc.filter, dbName)
				require.Equal(t, tc.err, err)
				require.Equal(t, tc.expectedFilter, newFilter)
			})
		}
	})

	t.Run("TestCreateQueryDocument", func(t *testing.T) {
		// nil filter -> just regexDoc

		regexDoc := bsonx.Doc{
			{"name", bsonx.Regex("^[^$]*$", "")},
		}

		queryArr := bsonx.Arr{
			bsonx.Document(regexDoc),
			bsonx.Document(modifiedFilter),
		}
		nameQueryDoc := bsonx.Doc{
			{"$and", bsonx.Array(queryArr)},
		}

		testCases := []struct {
			name        string
			filter      bsonx.Doc
			expectedDoc bsonx.Doc
			err         error
		}{
			{"TestNilFilter", nil, regexDoc, nil},
			{"TestEmptyFilter", bsonx.Doc{}, regexDoc, nil},
			{"TestNonEmptyFilter", nameFilter, nameQueryDoc, nil},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				doc, err := createQueryDocument(tc.filter, dbName)
				require.Equal(t, tc.err, err)
				require.Equal(t, tc.expectedDoc, doc)
			})
		}
	})
}
