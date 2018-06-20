package integration

import (
	"testing"

	"context"
	"os"

	"math"
	"strconv"
	"strings"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
	"github.com/mongodb/mongo-go-driver/internal/testutil"
	"github.com/stretchr/testify/require"
)

// compareVersions compares two version number strings (i.e. positive integers separated by
// periods). Comparisons are done to the lesser precision of the two versions. For example, 3.2 is
// considered equal to 3.2.11, whereas 3.2.0 is considered less than 3.2.11.
//
// Returns a positive int if version1 is greater than version2, a negative int if version1 is less
// than version2, and 0 if version1 is equal to version2.
func compareVersions(t *testing.T, v1 string, v2 string) int {
	n1 := strings.Split(v1, ".")
	n2 := strings.Split(v2, ".")

	for i := 0; i < int(math.Min(float64(len(n1)), float64(len(n2)))); i++ {
		i1, err := strconv.Atoi(n1[i])
		require.NoError(t, err)

		i2, err := strconv.Atoi(n2[i])
		require.NoError(t, err)

		difference := i1 - i2
		if difference != 0 {
			return difference
		}
	}

	return 0
}

func TestCompression(t *testing.T) {
	comp := os.Getenv("MONGO_GO_DRIVER_COMPRESSOR")
	if len(comp) == 0 {
		t.Skip("Skipping because no compressor specified")
	}

	server, err := testutil.Topology(t).SelectServer(context.Background(), description.WriteSelector())
	noerr(t, err)

	wc := writeconcern.New(writeconcern.WMajority())
	collOne := testutil.ColName(t)

	testutil.DropCollection(t, testutil.DBName(t), collOne)
	testutil.InsertDocs(t, testutil.DBName(t), collOne, wc, bson.NewDocument(bson.EC.String("name", "compression_test")))

	cmd := &command.Read{
		DB:      testutil.DBName(t),
		Command: bson.NewDocument(bson.EC.Int32("serverStatus", 1)),
	}

	ctx := context.Background()
	rw, err := server.Connection(ctx)
	noerr(t, err)

	rdr, err := cmd.RoundTrip(ctx, server.SelectedDescription(), rw)
	noerr(t, err)

	result, err := bson.ReadDocument(rdr)
	noerr(t, err)

	serverVersion, err := result.LookupErr("version")
	noerr(t, err)

	if compareVersions(t, serverVersion.StringValue(), "3.4") < 0 {
		t.Skip("skipping compression test for version < 3.4")
	}

	networkVal, err := result.LookupErr("network")
	noerr(t, err)

	require.Equal(t, networkVal.Type(), bson.TypeEmbeddedDocument)

	compressionVal, err := networkVal.MutableDocument().LookupErr("compression")
	noerr(t, err)

	snappy, err := compressionVal.MutableDocument().LookupErr("snappy")
	noerr(t, err)

	compressorKey := "compressor"
	compareTo36 := compareVersions(t, serverVersion.StringValue(), "3.6")
	if compareTo36 < 0 {
		compressorKey = "compressed"
	}
	compressor, err := snappy.MutableDocument().LookupErr(compressorKey)
	noerr(t, err)

	bytesIn, err := compressor.MutableDocument().LookupErr("bytesIn")
	noerr(t, err)

	require.True(t, bytesIn.IsNumber())
	require.True(t, bytesIn.Int64() > 0)
}
