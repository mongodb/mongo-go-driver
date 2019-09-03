package driver

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"go.mongodb.org/mongo-driver/x/mongo/driver/wiremessage"
)

func TestCompression(t *testing.T) {
	compressors := []wiremessage.CompressorID{
		wiremessage.CompressorNoOp,
		wiremessage.CompressorSnappy,
		wiremessage.CompressorZLib,
		wiremessage.CompressorZstd,
	}

	for _, compressor := range compressors {
		t.Run(strconv.Itoa(int(compressor)), func(t *testing.T) {
			payload := "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt"
			opts := CompressionOpts{
				Compressor:       compressor,
				ZlibLevel:        wiremessage.DefaultZlibLevel,
				ZstdLevel:        wiremessage.DefaultZstdLevel,
				UncompressedSize: int32(len(payload)),
			}
			compressed, err := CompressPlayoad([]byte(payload), opts)
			if err != nil {
				require.Contains(t, err.Error(), "support requires cgo")
				return
			}
			assert.NotEqual(t, 0, len(compressed))
			decompressed, err := DecompressPayload(compressed, opts)
			if err != nil {
				require.Contains(t, err.Error(), "support requires cgo")
				return
			}
			assert.EqualValues(t, payload, decompressed)
		})
	}
}
