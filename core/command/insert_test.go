package command

import (
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/stretchr/testify/assert"
)

func TestInsertCommandSplitting(t *testing.T) {
	const (
		megabyte = 10 * 10 * 10 * 10 * 10 * 10
		kilobyte = 10 * 10 * 10
	)

	ss := description.SelectedServer{}
	t.Run("split_smoke_test", func(t *testing.T) {
		i := &Insert{}
		for n := 0; n < 100; n++ {
			i.Docs = append(i.Docs, bson.NewDocument(bson.EC.Int32("a", int32(n))))
		}

		batches, err := i.split(10, kilobyte) // 1kb
		assert.NoError(t, err)
		assert.Len(t, batches, 10)
		for _, b := range batches {
			assert.Len(t, b, 10)
			cmd, err := i.encodeBatch(b, ss)
			assert.NoError(t, err)

			wm, err := cmd.Encode(ss)
			assert.NoError(t, err)

			assert.True(t, wm.Len() < 16*megabyte)
		}
	})
	t.Run("split_with_small_target_Size", func(t *testing.T) {
		i := &Insert{}
		for n := 0; n < 100; n++ {
			i.Docs = append(i.Docs, bson.NewDocument(bson.EC.Int32("a", int32(n))))
		}

		batches, err := i.split(100, 32) // 32 bytes?
		assert.NoError(t, err)
		assert.Len(t, batches, 50)
		for _, b := range batches {
			assert.Len(t, b, 2)
			cmd, err := i.encodeBatch(b, ss)
			assert.NoError(t, err)

			wm, err := cmd.Encode(ss)
			assert.NoError(t, err)

			assert.True(t, wm.Len() < 16*megabyte)
		}
	})
	t.Run("invalid_max_counts", func(t *testing.T) {
		i := &Insert{}
		for n := 0; n < 100; n++ {
			i.Docs = append(i.Docs, bson.NewDocument(bson.EC.Int32("a", int32(n))))
		}

		for _, ct := range []int{-1, 0, -1000} {
			batches, err := i.split(ct, 100*megabyte)
			assert.NoError(t, err)
			assert.Len(t, batches, 100)
			for _, b := range batches {
				assert.Len(t, b, 1)
				cmd, err := i.encodeBatch(b, ss)
				assert.NoError(t, err)

				wm, err := cmd.Encode(ss)
				assert.NoError(t, err)

				assert.True(t, wm.Len() < 16*megabyte)
			}
		}

	})
}
