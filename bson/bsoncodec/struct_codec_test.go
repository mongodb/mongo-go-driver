package bsoncodec

import (
	"testing"

	"github.com/mongodb/mongo-go-driver/bson/bsoncore"
	"github.com/mongodb/mongo-go-driver/bson/bsonrw"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
	"github.com/stretchr/testify/assert"
)

func TestDecodeNullValue(t *testing.T) {

	ind, doc := bsoncore.AppendDocumentStart(nil)
	doc = bsoncore.AppendStringElement(doc, "b", "foo")
	doc = bsoncore.AppendNullElement(doc, "blank")
	doc, _ = bsoncore.AppendDocumentEnd(doc, ind)

	type teststruct struct {
		Blank *objectid.ObjectID
		B     string
	}
	var got *teststruct
	dc := DecodeContext{Registry: buildDefaultRegistry()}
	parser, err := NewStructCodec(DefaultStructTagParser)
	noerr(t, err)
	vr := bsonrw.NewBSONValueReader(doc)
	err = parser.DecodeValue(dc, vr, &got)
	noerr(t, err)
	want := teststruct{nil, "foo"}
	if got.Blank != want.Blank || got.B != want.B {
		t.Error("Documents do not match")
		t.Errorf("\ngot :%v\nwant:%v", got, want)
	}
}

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
