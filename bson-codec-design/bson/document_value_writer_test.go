package bson

import (
	"testing"
)

func noerr(t *testing.T, err error) {
	if err != nil {
		t.Helper()
		t.Errorf("Unepexted error: %v", err)
		t.FailNow()
	}
}

func TestDocumentValueWriter(t *testing.T) {
	testCases := []struct {
		name string
		fn   func(*testing.T, *documentValueWriter)
		want *Document
	}{
		{
			"simple document",
			dvwBasicDoc,
			NewDocument(EC.Boolean("foo", true)),
		},
		{
			"nested document",
			dvwNestedDoc,
			NewDocument(EC.SubDocumentFromElements("foo", EC.Boolean("bar", true))),
		},
		{
			"simple array",
			dvwBasicArray,
			NewDocument(EC.ArrayFromElements("foo", VC.Boolean(true))),
		},
		{
			"code with scope",
			dvwCodeWithScopeNoNested,
			NewDocument(EC.CodeWithScope("foo", "var hello = world;", NewDocument(EC.Boolean("bar", false)))),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := NewDocument()
			dvw := newDocumentValueWriter(got)
			tc.fn(t, dvw)
			if !got.Equal(tc.want) {
				t.Errorf("Documents are not equal. got %v; want %v", got, tc.want)
			}
		})
	}
}

func dvwBasicDoc(t *testing.T, dvw *documentValueWriter) {
	dw, err := dvw.WriteDocument()
	noerr(t, err)
	vw, err := dw.WriteDocumentElement("foo")
	noerr(t, err)
	err = vw.WriteBoolean(true)
	noerr(t, err)
	err = dw.WriteDocumentEnd()
	noerr(t, err)

	return
}

func dvwBasicArray(t *testing.T, dvw *documentValueWriter) {
	dw, err := dvw.WriteDocument()
	noerr(t, err)
	vw, err := dw.WriteDocumentElement("foo")
	noerr(t, err)
	aw, err := vw.WriteArray()
	noerr(t, err)
	vw, err = aw.WriteArrayElement()
	noerr(t, err)
	err = vw.WriteBoolean(true)
	noerr(t, err)
	err = aw.WriteArrayEnd()
	noerr(t, err)
	err = dw.WriteDocumentEnd()
	noerr(t, err)

	return
}

func dvwNestedDoc(t *testing.T, dvw *documentValueWriter) {
	dw, err := dvw.WriteDocument()
	noerr(t, err)
	vw, err := dw.WriteDocumentElement("foo")
	noerr(t, err)
	dw2, err := vw.WriteDocument()
	noerr(t, err)
	vw2, err := dw2.WriteDocumentElement("bar")
	noerr(t, err)
	err = vw2.WriteBoolean(true)
	noerr(t, err)
	err = dw2.WriteDocumentEnd()
	noerr(t, err)
	err = dw.WriteDocumentEnd()
	noerr(t, err)

	return
}

func dvwCodeWithScopeNoNested(t *testing.T, dvw *documentValueWriter) {
	dw, err := dvw.WriteDocument()
	noerr(t, err)
	vw, err := dw.WriteDocumentElement("foo")
	noerr(t, err)
	dw2, err := vw.WriteCodeWithScope("var hello = world;")
	noerr(t, err)
	vw, err = dw2.WriteDocumentElement("bar")
	noerr(t, err)
	err = vw.WriteBoolean(false)
	noerr(t, err)
	err = dw2.WriteDocumentEnd()
	noerr(t, err)
	err = dw.WriteDocumentEnd()
	noerr(t, err)

	return
}
