package command

import (
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
	"github.com/mongodb/mongo-go-driver/mongo/writeconcern"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
	"github.com/mongodb/mongo-go-driver/x/network/description"
	"testing"
)

func TestAggregate(t *testing.T) {
	ns := Namespace{
		DB:         "db",
		Collection: "coll",
	}
	wc := writeconcern.New(writeconcern.W(10))

	// Write concern should not be encoded on an aggregate without a $out stage
	t.Run("TestNoWriteConcern", func(t *testing.T) {
		cmd := Aggregate{
			NS:           ns,
			Pipeline:     bsonx.Arr{},
			WriteConcern: wc,
		}

		readCmd, err := cmd.encode(description.SelectedServer{})
		testhelpers.RequireNil(t, err, "non-nil error from encode: %s", err)

		_, err = readCmd.Command.LookupErr("writeConcern")
		testhelpers.RequireNotNil(t, err, "write concern found on agg command with no $out")
	})

	// Write concern should be encoded on an aggregate with an $out stage.
	t.Run("TestWriteConcern", func(t *testing.T) {
		outDoc := bsonx.Doc{
			{"$out", bsonx.Int32(1)},
		}
		cmd := Aggregate{
			NS: ns,
			Pipeline: bsonx.Arr{
				bsonx.Document(outDoc),
			},
			WriteConcern: wc,
		}

		readCmd, err := cmd.encode(description.SelectedServer{})
		testhelpers.RequireNil(t, err, "non-nil error from encode: %s", err)

		_, err = readCmd.Command.LookupErr("writeConcern")
		testhelpers.RequireNil(t, err, "write concern not found on agg command with $out")
	})
}
