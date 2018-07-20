package bson

import (
	"bytes"
	"fmt"
	"testing"
)

func bytesFromDoc(doc *Document) []byte {
	b, err := doc.MarshalBSON()
	if err != nil {
		panic(fmt.Errorf("Couldn't marshal BSON document: %v", err))
	}
	return b
}

func TestValueWriter(t *testing.T) {
	testCases := []struct {
		name string
		fn   func(*testing.T, *valueWriter)
		want []byte
	}{
		{
			"simple document",
			vwBasicDoc,
			bytesFromDoc(NewDocument(EC.Boolean("foo", true))),
		},
		{
			"nested document",
			vwNestedDoc,
			bytesFromDoc(NewDocument(EC.SubDocumentFromElements("foo", EC.Boolean("bar", true)))),
		},
		{
			"simple array",
			vwBasicArray,
			bytesFromDoc(NewDocument(EC.ArrayFromElements("foo", VC.Boolean(true)))),
		},
		{
			"code with scope",
			vwCodeWithScopeNoNested,
			bytesFromDoc(NewDocument(EC.CodeWithScope("foo", "var hello = world;", NewDocument(EC.Boolean("bar", false))))),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := make(writer, 0, 1024)
			vw := newValueWriter(&got)
			tc.fn(t, vw)
			if !bytes.Equal(got, tc.want) {
				t.Errorf("Documents are not equal. got %v; want %v", Reader(got), Reader(tc.want))
				t.Errorf("Bytes:\n%v\n%v", got, tc.want)
			}
		})
	}
}

func vwBasicDoc(t *testing.T, vw *valueWriter) {
	dw, err := vw.WriteDocument()
	noerr(t, err)
	vw2, err := dw.WriteDocumentElement("foo")
	noerr(t, err)
	err = vw2.WriteBoolean(true)
	noerr(t, err)
	err = dw.WriteDocumentEnd()
	noerr(t, err)

	return
}

func vwBasicArray(t *testing.T, vw *valueWriter) {
	dw, err := vw.WriteDocument()
	noerr(t, err)
	vw2, err := dw.WriteDocumentElement("foo")
	noerr(t, err)
	aw, err := vw2.WriteArray()
	noerr(t, err)
	vw2, err = aw.WriteArrayElement()
	noerr(t, err)
	err = vw2.WriteBoolean(true)
	noerr(t, err)
	err = aw.WriteArrayEnd()
	noerr(t, err)
	err = dw.WriteDocumentEnd()
	noerr(t, err)

	return
}

func vwNestedDoc(t *testing.T, vw *valueWriter) {
	dw, err := vw.WriteDocument()
	noerr(t, err)
	vw2, err := dw.WriteDocumentElement("foo")
	noerr(t, err)
	dw2, err := vw2.WriteDocument()
	noerr(t, err)
	vw3, err := dw2.WriteDocumentElement("bar")
	noerr(t, err)
	err = vw3.WriteBoolean(true)
	noerr(t, err)
	err = dw2.WriteDocumentEnd()
	noerr(t, err)
	err = dw.WriteDocumentEnd()
	noerr(t, err)

	return
}

func vwCodeWithScopeNoNested(t *testing.T, vw *valueWriter) {
	dw, err := vw.WriteDocument()
	noerr(t, err)
	vw2, err := dw.WriteDocumentElement("foo")
	noerr(t, err)
	dw2, err := vw2.WriteCodeWithScope("var hello = world;")
	noerr(t, err)
	vw2, err = dw2.WriteDocumentElement("bar")
	noerr(t, err)
	err = vw2.WriteBoolean(false)
	noerr(t, err)
	err = dw2.WriteDocumentEnd()
	noerr(t, err)
	err = dw.WriteDocumentEnd()
	noerr(t, err)

	return
}
