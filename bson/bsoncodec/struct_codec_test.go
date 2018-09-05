package bsoncodec

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZeoerInterfaceUsedByDecoder(t *testing.T) {
	enc := &StructCodec{}

	// cases that are zero, because they are known types or pointers
	var st *nonZeroer
	assert.True(t, enc.isZero(st))
	assert.True(t, enc.isZero(0))
	assert.True(t, enc.isZero(false))

	// cases that shouldn't be zero
	st = &nonZeroer{value: false}
	assert.False(t, enc.isZero(struct{ val bool }{val: true}))
	assert.False(t, enc.isZero(struct{ val bool }{val: false}))
	assert.False(t, enc.isZero(st))
	st.value = true
	assert.False(t, enc.isZero(st))

	// a test to see if the interface impacts the outcome
	z := zeroTest{}
	assert.False(t, enc.isZero(z))

	z.reportZero = true
	assert.True(t, enc.isZero(z))
}
