package bsoncodec

import (
	"reflect"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson/bsonrw"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
)

func noerr(t *testing.T, err error) {
	if err != nil {
		t.Helper()
		t.Errorf("Unexpected error: (%T)%v", err, err)
		t.FailNow()
	}
}

func compareErrors(err1, err2 error) bool {
	if err1 == nil && err2 == nil {
		return true
	}

	if err1 == nil || err2 == nil {
		return false
	}

	if err1.Error() != err2.Error() {
		return false
	}

	return true
}

func compareDecimal128(d1, d2 decimal.Decimal128) bool {
	d1H, d1L := d1.GetBytes()
	d2H, d2L := d2.GetBytes()

	if d1H != d2H {
		return false
	}

	if d1L != d2L {
		return false
	}

	return true
}

func compareStrings(s1, s2 string) bool { return s1 == s2 }

type noPrivateFields struct {
	a string
}

func compareNoPrivateFields(npf1, npf2 noPrivateFields) bool {
	return npf1.a != npf2.a // We don't want these to be equal
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

func (llc *llCodec) EncodeValue(_ EncodeContext, _ bsonrw.ValueWriter, i interface{}) error {
	if llc.err != nil {
		return llc.err
	}

	llc.encodeval = i
	return nil
}

func (llc *llCodec) DecodeValue(_ DecodeContext, _ bsonrw.ValueReader, i interface{}) error {
	if llc.err != nil {
		return llc.err
	}

	val := reflect.ValueOf(i)
	if val.Type().Kind() != reflect.Ptr {
		llc.t.Errorf("Value provided to DecodeValue must be a pointer, but got %T", i)
		return nil
	}

	if !reflect.TypeOf(llc.decodeval).AssignableTo(val.Type().Elem()) {
		llc.t.Errorf("decodeval must be assignable to i provided to DecodeValue, but is not. decodeval %T; i %T", llc.decodeval, i)
		return nil
	}

	val.Elem().Set(reflect.ValueOf(llc.decodeval))
	return nil
}
