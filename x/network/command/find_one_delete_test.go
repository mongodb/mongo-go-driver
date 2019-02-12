package command

import (
	"testing"

	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/network/description"
)

func TestFindOneAndDelete(t *testing.T) {
	t.Run("Encode Write Concern for MaxWireVersion >= 4", func(t *testing.T) {
		desc := description.SelectedServer{
			Server: description.Server{
				WireVersion: &description.VersionRange{Min: 0, Max: 4},
			},
		}
		wc := writeconcern.New(writeconcern.WMajority())
		cmd := FindOneAndDelete{NS: Namespace{DB: "foo", Collection: "bar"}, WriteConcern: wc}
		write, err := cmd.encode(desc)
		noerr(t, err)
		if write.WriteConcern != wc {
			t.Error("write concern should be added to write command, but is missing")
		}
	})
	t.Run("Omit Write Concern for MaxWireVersion < 4", func(t *testing.T) {
		desc := description.SelectedServer{
			Server: description.Server{
				WireVersion: &description.VersionRange{Min: 0, Max: 3},
			},
		}
		wc := writeconcern.New(writeconcern.WMajority())
		cmd := FindOneAndDelete{NS: Namespace{DB: "foo", Collection: "bar"}, WriteConcern: wc}
		write, err := cmd.encode(desc)
		noerr(t, err)
		if write.WriteConcern != nil {
			t.Error("write concern should be omitted from write command, but is present")
		}
	})
}
