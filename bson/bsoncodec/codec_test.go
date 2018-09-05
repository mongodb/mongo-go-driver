package bsoncodec

import (
	"reflect"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
)

func compareValues(v1, v2 *bson.Value) bool     { return v1.Equal(v2) }
func compareElements(e1, e2 *bson.Element) bool { return e1.Equal(e2) }
func compareStrings(s1, s2 string) bool         { return s1 == s2 }

type noPrivateFields struct {
	a string
}

func compareNoPrivateFields(npf1, npf2 noPrivateFields) bool {
	return npf1.a != npf2.a // We don't want these to be equal
}

func docToBytes(d *bson.Document) []byte {
	b, err := d.MarshalBSON()
	if err != nil {
		panic(err)
	}
	return b
}

type zeroTest struct {
	reportZero bool
}

func (z zeroTest) IsZero() bool { return z.reportZero }

func compareZeroTest(_, _ zeroTest) bool { return true }

type nonZeroer struct {
	value bool
}

type llCodec struct {
	t         *testing.T
	decodeval interface{}
	encodeval interface{}
	err       error
}

func (llc *llCodec) EncodeValue(_ EncodeContext, _ ValueWriter, i interface{}) error {
	if llc.err != nil {
		return llc.err
	}

	llc.encodeval = i
	return nil
}

func (llc *llCodec) DecodeValue(_ DecodeContext, _ ValueReader, i interface{}) error {
	if llc.err != nil {
		return llc.err
	}

	val := reflect.ValueOf(i)
	if val.Type().Kind() != reflect.Ptr {
		llc.t.Errorf("Value provided to DecodeValue must be a pointer, but got %T", i)
		return nil
	}

	switch val.Type() {
	case tDocument:
		decodeval, ok := llc.decodeval.(*bson.Document)
		if !ok {
			llc.t.Errorf("decodeval must be a *Document if the i is a *Document. decodeval %T", llc.decodeval)
			return nil
		}

		doc := i.(*bson.Document)
		doc.Reset()
		err := doc.Concat(decodeval)
		if err != nil {
			llc.t.Errorf("could not concatenate the decoded val to doc: %v", err)
			return err
		}

		return nil
	case tArray:
		decodeval, ok := llc.decodeval.(*bson.Array)
		if !ok {
			llc.t.Errorf("decodeval must be a *Array if the i is a *Array. decodeval %T", llc.decodeval)
			return nil
		}

		arr := i.(*bson.Array)
		arr.Reset()
		err := arr.Concat(decodeval)
		if err != nil {
			llc.t.Errorf("could not concatenate the decoded val to array: %v", err)
			return err
		}

		return nil
	}

	if !reflect.TypeOf(llc.decodeval).AssignableTo(val.Type().Elem()) {
		llc.t.Errorf("decodeval must be assignable to i provided to DecodeValue, but is not. decodeval %T; i %T", llc.decodeval, i)
		return nil
	}

	val.Elem().Set(reflect.ValueOf(llc.decodeval))
	return nil
}
