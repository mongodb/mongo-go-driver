// Copied from https://github.com/stretchr/testify/blob/1333b5d3bda8cf5aedcf3e1aaa95cac28aaab892/assert/assertions_test.go

// Copyright 2020 Mat Ryer, Tyler Bunnell and all contributors. All rights reserved.
// Use of this source code is governed by an MIT-style license that can be found in
// the THIRD-PARTY-NOTICES file.

package assert

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"math"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

// AssertionTesterInterface defines an interface to be used for testing assertion methods
type AssertionTesterInterface interface {
	TestMethod()
}

// AssertionTesterConformingObject is an object that conforms to the AssertionTesterInterface interface
type AssertionTesterConformingObject struct {
}

func (a *AssertionTesterConformingObject) TestMethod() {
}

// AssertionTesterNonConformingObject is an object that does not conform to the AssertionTesterInterface interface
type AssertionTesterNonConformingObject struct {
}

func TestObjectsAreEqual(t *testing.T) {
	cases := []struct {
		expected interface{}
		actual   interface{}
		result   bool
	}{
		// cases that are expected to be equal
		{"Hello World", "Hello World", true},
		{123, 123, true},
		{123.5, 123.5, true},
		{[]byte("Hello World"), []byte("Hello World"), true},
		{nil, nil, true},

		// cases that are expected not to be equal
		{map[int]int{5: 10}, map[int]int{10: 20}, false},
		{'x', "x", false},
		{"x", 'x', false},
		{0, 0.1, false},
		{0.1, 0, false},
		{time.Now, time.Now, false},
		{func() {}, func() {}, false},
		{uint32(10), int32(10), false},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("ObjectsAreEqual(%#v, %#v)", c.expected, c.actual), func(t *testing.T) {
			res := ObjectsAreEqual(c.expected, c.actual)

			if res != c.result {
				t.Errorf("ObjectsAreEqual(%#v, %#v) should return %#v", c.expected, c.actual, c.result)
			}

		})
	}

	// Cases where type differ but values are equal
	if !ObjectsAreEqualValues(uint32(10), int32(10)) {
		t.Error("ObjectsAreEqualValues should return true")
	}
	if ObjectsAreEqualValues(0, nil) {
		t.Fail()
	}
	if ObjectsAreEqualValues(nil, 0) {
		t.Fail()
	}

}

func TestIsType(t *testing.T) {

	mockT := new(testing.T)

	if !IsType(mockT, new(AssertionTesterConformingObject), new(AssertionTesterConformingObject)) {
		t.Error("IsType should return true: AssertionTesterConformingObject is the same type as AssertionTesterConformingObject")
	}
	if IsType(mockT, new(AssertionTesterConformingObject), new(AssertionTesterNonConformingObject)) {
		t.Error("IsType should return false: AssertionTesterConformingObject is not the same type as AssertionTesterNonConformingObject")
	}

}

func TestEqual(t *testing.T) {
	type myType string

	mockT := new(testing.T)
	var m map[string]interface{}

	cases := []struct {
		expected interface{}
		actual   interface{}
		result   bool
		remark   string
	}{
		{"Hello World", "Hello World", true, ""},
		{123, 123, true, ""},
		{123.5, 123.5, true, ""},
		{[]byte("Hello World"), []byte("Hello World"), true, ""},
		{nil, nil, true, ""},
		{int32(123), int32(123), true, ""},
		{uint64(123), uint64(123), true, ""},
		{myType("1"), myType("1"), true, ""},
		{&struct{}{}, &struct{}{}, true, "pointer equality is based on equality of underlying value"},

		// Not expected to be equal
		{m["bar"], "something", false, ""},
		{myType("1"), myType("2"), false, ""},

		// A case that might be confusing, especially with numeric literals
		{10, uint(10), false, ""},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("Equal(%#v, %#v)", c.expected, c.actual), func(t *testing.T) {
			res := Equal(mockT, c.expected, c.actual)

			if res != c.result {
				t.Errorf("Equal(%#v, %#v) should return %#v: %s", c.expected, c.actual, c.result, c.remark)
			}
		})
	}
}

// bufferT implements TestingT. Its implementation of Errorf writes the output that would be produced by
// testing.T.Errorf to an internal bytes.Buffer.
type bufferT struct {
	buf bytes.Buffer
}

func (t *bufferT) Errorf(format string, args ...interface{}) {
	// implementation of decorate is copied from testing.T
	decorate := func(s string) string {
		_, file, line, ok := runtime.Caller(3) // decorate + log + public function.
		if ok {
			// Truncate file name at last file name separator.
			if index := strings.LastIndex(file, "/"); index >= 0 {
				file = file[index+1:]
			} else if index = strings.LastIndex(file, "\\"); index >= 0 {
				file = file[index+1:]
			}
		} else {
			file = "???"
			line = 1
		}
		buf := new(bytes.Buffer)
		// Every line is indented at least one tab.
		buf.WriteByte('\t')
		fmt.Fprintf(buf, "%s:%d: ", file, line)
		lines := strings.Split(s, "\n")
		if l := len(lines); l > 1 && lines[l-1] == "" {
			lines = lines[:l-1]
		}
		for i, line := range lines {
			if i > 0 {
				// Second and subsequent lines are indented an extra tab.
				buf.WriteString("\n\t\t")
			}
			buf.WriteString(line)
		}
		buf.WriteByte('\n')
		return buf.String()
	}
	t.buf.WriteString(decorate(fmt.Sprintf(format, args...)))
}

func TestStringEqual(t *testing.T) {
	for _, currCase := range []struct {
		equalWant  string
		equalGot   string
		msgAndArgs []interface{}
		want       string
	}{
		{equalWant: "hi, \nmy name is", equalGot: "what,\nmy name is", want: "\tassertions.go:\\d+: \n\t+Error Trace:\t\n\t+Error:\\s+Not equal:\\s+\n\\s+expected: \"hi, \\\\nmy name is\"\n\\s+actual\\s+: \"what,\\\\nmy name is\"\n\\s+Diff:\n\\s+-+ Expected\n\\s+\\++ Actual\n\\s+@@ -1,2 \\+1,2 @@\n\\s+-hi, \n\\s+\\+what,\n\\s+my name is"},
	} {
		mockT := &bufferT{}
		Equal(mockT, currCase.equalWant, currCase.equalGot, currCase.msgAndArgs...)
	}
}

func TestEqualFormatting(t *testing.T) {
	for _, currCase := range []struct {
		equalWant  string
		equalGot   string
		msgAndArgs []interface{}
		want       string
	}{
		{equalWant: "want", equalGot: "got", want: "\tassertions.go:\\d+: \n\t+Error Trace:\t\n\t+Error:\\s+Not equal:\\s+\n\\s+expected: \"want\"\n\\s+actual\\s+: \"got\"\n\\s+Diff:\n\\s+-+ Expected\n\\s+\\++ Actual\n\\s+@@ -1 \\+1 @@\n\\s+-want\n\\s+\\+got\n"},
		{equalWant: "want", equalGot: "got", msgAndArgs: []interface{}{"hello, %v!", "world"}, want: "\tassertions.go:[0-9]+: \n\t+Error Trace:\t\n\t+Error:\\s+Not equal:\\s+\n\\s+expected: \"want\"\n\\s+actual\\s+: \"got\"\n\\s+Diff:\n\\s+-+ Expected\n\\s+\\++ Actual\n\\s+@@ -1 \\+1 @@\n\\s+-want\n\\s+\\+got\n\\s+Messages:\\s+hello, world!\n"},
		{equalWant: "want", equalGot: "got", msgAndArgs: []interface{}{123}, want: "\tassertions.go:[0-9]+: \n\t+Error Trace:\t\n\t+Error:\\s+Not equal:\\s+\n\\s+expected: \"want\"\n\\s+actual\\s+: \"got\"\n\\s+Diff:\n\\s+-+ Expected\n\\s+\\++ Actual\n\\s+@@ -1 \\+1 @@\n\\s+-want\n\\s+\\+got\n\\s+Messages:\\s+123\n"},
		{equalWant: "want", equalGot: "got", msgAndArgs: []interface{}{struct{ a string }{"hello"}}, want: "\tassertions.go:[0-9]+: \n\t+Error Trace:\t\n\t+Error:\\s+Not equal:\\s+\n\\s+expected: \"want\"\n\\s+actual\\s+: \"got\"\n\\s+Diff:\n\\s+-+ Expected\n\\s+\\++ Actual\n\\s+@@ -1 \\+1 @@\n\\s+-want\n\\s+\\+got\n\\s+Messages:\\s+{a:hello}\n"},
	} {
		mockT := &bufferT{}
		Equal(mockT, currCase.equalWant, currCase.equalGot, currCase.msgAndArgs...)
	}
}

func TestFormatUnequalValues(t *testing.T) {
	expected, actual := formatUnequalValues("foo", "bar")
	Equal(t, `"foo"`, expected, "value should not include type")
	Equal(t, `"bar"`, actual, "value should not include type")

	expected, actual = formatUnequalValues(123, 123)
	Equal(t, `123`, expected, "value should not include type")
	Equal(t, `123`, actual, "value should not include type")

	expected, actual = formatUnequalValues(int64(123), int32(123))
	Equal(t, `int64(123)`, expected, "value should include type")
	Equal(t, `int32(123)`, actual, "value should include type")

	expected, actual = formatUnequalValues(int64(123), nil)
	Equal(t, `int64(123)`, expected, "value should include type")
	Equal(t, `<nil>(<nil>)`, actual, "value should include type")

	type testStructType struct {
		Val string
	}

	expected, actual = formatUnequalValues(&testStructType{Val: "test"}, &testStructType{Val: "test"})
	Equal(t, `&assert.testStructType{Val:"test"}`, expected, "value should not include type annotation")
	Equal(t, `&assert.testStructType{Val:"test"}`, actual, "value should not include type annotation")
}

func TestNotNil(t *testing.T) {

	mockT := new(testing.T)

	if !NotNil(mockT, new(AssertionTesterConformingObject)) {
		t.Error("NotNil should return true: object is not nil")
	}
	if NotNil(mockT, nil) {
		t.Error("NotNil should return false: object is nil")
	}
	if NotNil(mockT, (*struct{})(nil)) {
		t.Error("NotNil should return false: object is (*struct{})(nil)")
	}

}

func TestNil(t *testing.T) {

	mockT := new(testing.T)

	if !Nil(mockT, nil) {
		t.Error("Nil should return true: object is nil")
	}
	if !Nil(mockT, (*struct{})(nil)) {
		t.Error("Nil should return true: object is (*struct{})(nil)")
	}
	if Nil(mockT, new(AssertionTesterConformingObject)) {
		t.Error("Nil should return false: object is not nil")
	}

}

func TestTrue(t *testing.T) {

	mockT := new(testing.T)

	if !True(mockT, true) {
		t.Error("True should return true")
	}
	if True(mockT, false) {
		t.Error("True should return false")
	}

}

func TestFalse(t *testing.T) {

	mockT := new(testing.T)

	if !False(mockT, false) {
		t.Error("False should return true")
	}
	if False(mockT, true) {
		t.Error("False should return false")
	}

}

func TestNotEqual(t *testing.T) {

	mockT := new(testing.T)

	cases := []struct {
		expected interface{}
		actual   interface{}
		result   bool
	}{
		// cases that are expected not to match
		{"Hello World", "Hello World!", true},
		{123, 1234, true},
		{123.5, 123.55, true},
		{[]byte("Hello World"), []byte("Hello World!"), true},
		{nil, new(AssertionTesterConformingObject), true},

		// cases that are expected to match
		{nil, nil, false},
		{"Hello World", "Hello World", false},
		{123, 123, false},
		{123.5, 123.5, false},
		{[]byte("Hello World"), []byte("Hello World"), false},
		{new(AssertionTesterConformingObject), new(AssertionTesterConformingObject), false},
		{&struct{}{}, &struct{}{}, false},
		{func() int { return 23 }, func() int { return 24 }, false},
		// A case that might be confusing, especially with numeric literals
		{int(10), uint(10), true},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("NotEqual(%#v, %#v)", c.expected, c.actual), func(t *testing.T) {
			res := NotEqual(mockT, c.expected, c.actual)

			if res != c.result {
				t.Errorf("NotEqual(%#v, %#v) should return %#v", c.expected, c.actual, c.result)
			}
		})
	}
}

func TestNotEqualValues(t *testing.T) {
	mockT := new(testing.T)

	cases := []struct {
		expected interface{}
		actual   interface{}
		result   bool
	}{
		// cases that are expected not to match
		{"Hello World", "Hello World!", true},
		{123, 1234, true},
		{123.5, 123.55, true},
		{[]byte("Hello World"), []byte("Hello World!"), true},
		{nil, new(AssertionTesterConformingObject), true},

		// cases that are expected to match
		{nil, nil, false},
		{"Hello World", "Hello World", false},
		{123, 123, false},
		{123.5, 123.5, false},
		{[]byte("Hello World"), []byte("Hello World"), false},
		{new(AssertionTesterConformingObject), new(AssertionTesterConformingObject), false},
		{&struct{}{}, &struct{}{}, false},

		// Different behaviour from NotEqual()
		{func() int { return 23 }, func() int { return 24 }, true},
		{int(10), int(11), true},
		{int(10), uint(10), false},

		{struct{}{}, struct{}{}, false},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("NotEqualValues(%#v, %#v)", c.expected, c.actual), func(t *testing.T) {
			res := NotEqualValues(mockT, c.expected, c.actual)

			if res != c.result {
				t.Errorf("NotEqualValues(%#v, %#v) should return %#v", c.expected, c.actual, c.result)
			}
		})
	}
}

func TestContainsNotContains(t *testing.T) {

	type A struct {
		Name, Value string
	}
	list := []string{"Foo", "Bar"}

	complexList := []*A{
		{"b", "c"},
		{"d", "e"},
		{"g", "h"},
		{"j", "k"},
	}
	simpleMap := map[interface{}]interface{}{"Foo": "Bar"}
	var zeroMap map[interface{}]interface{}

	cases := []struct {
		expected interface{}
		actual   interface{}
		result   bool
	}{
		{"Hello World", "Hello", true},
		{"Hello World", "Salut", false},
		{list, "Bar", true},
		{list, "Salut", false},
		{complexList, &A{"g", "h"}, true},
		{complexList, &A{"g", "e"}, false},
		{simpleMap, "Foo", true},
		{simpleMap, "Bar", false},
		{zeroMap, "Bar", false},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("Contains(%#v, %#v)", c.expected, c.actual), func(t *testing.T) {
			mockT := new(testing.T)
			res := Contains(mockT, c.expected, c.actual)

			if res != c.result {
				if res {
					t.Errorf("Contains(%#v, %#v) should return true:\n\t%#v contains %#v", c.expected, c.actual, c.expected, c.actual)
				} else {
					t.Errorf("Contains(%#v, %#v) should return false:\n\t%#v does not contain %#v", c.expected, c.actual, c.expected, c.actual)
				}
			}
		})
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("NotContains(%#v, %#v)", c.expected, c.actual), func(t *testing.T) {
			mockT := new(testing.T)
			res := NotContains(mockT, c.expected, c.actual)

			// NotContains should be inverse of Contains. If it's not, something is wrong
			if res == Contains(mockT, c.expected, c.actual) {
				if res {
					t.Errorf("NotContains(%#v, %#v) should return true:\n\t%#v does not contains %#v", c.expected, c.actual, c.expected, c.actual)
				} else {
					t.Errorf("NotContains(%#v, %#v) should return false:\n\t%#v contains %#v", c.expected, c.actual, c.expected, c.actual)
				}
			}
		})
	}
}

func TestContainsFailMessage(t *testing.T) {

	mockT := new(mockTestingT)

	Contains(mockT, "Hello World", errors.New("Hello"))
	expectedFail := "\"Hello World\" does not contain &errors.errorString{s:\"Hello\"}"
	actualFail := mockT.errorString()
	if !strings.Contains(actualFail, expectedFail) {
		t.Errorf("Contains failure should include %q but was %q", expectedFail, actualFail)
	}
}

func TestContainsNotContainsOnNilValue(t *testing.T) {
	mockT := new(mockTestingT)

	Contains(mockT, nil, "key")
	expectedFail := "<nil> could not be applied builtin len()"
	actualFail := mockT.errorString()
	if !strings.Contains(actualFail, expectedFail) {
		t.Errorf("Contains failure should include %q but was %q", expectedFail, actualFail)
	}

	NotContains(mockT, nil, "key")
	if !strings.Contains(actualFail, expectedFail) {
		t.Errorf("Contains failure should include %q but was %q", expectedFail, actualFail)
	}
}

func Test_containsElement(t *testing.T) {

	list1 := []string{"Foo", "Bar"}
	list2 := []int{1, 2}
	simpleMap := map[interface{}]interface{}{"Foo": "Bar"}

	ok, found := containsElement("Hello World", "World")
	True(t, ok)
	True(t, found)

	ok, found = containsElement(list1, "Foo")
	True(t, ok)
	True(t, found)

	ok, found = containsElement(list1, "Bar")
	True(t, ok)
	True(t, found)

	ok, found = containsElement(list2, 1)
	True(t, ok)
	True(t, found)

	ok, found = containsElement(list2, 2)
	True(t, ok)
	True(t, found)

	ok, found = containsElement(list1, "Foo!")
	True(t, ok)
	False(t, found)

	ok, found = containsElement(list2, 3)
	True(t, ok)
	False(t, found)

	ok, found = containsElement(list2, "1")
	True(t, ok)
	False(t, found)

	ok, found = containsElement(simpleMap, "Foo")
	True(t, ok)
	True(t, found)

	ok, found = containsElement(simpleMap, "Bar")
	True(t, ok)
	False(t, found)

	ok, found = containsElement(1433, "1")
	False(t, ok)
	False(t, found)
}

func TestElementsMatch(t *testing.T) {
	mockT := new(testing.T)

	cases := []struct {
		expected interface{}
		actual   interface{}
		result   bool
	}{
		// matching
		{nil, nil, true},

		{nil, nil, true},
		{[]int{}, []int{}, true},
		{[]int{1}, []int{1}, true},
		{[]int{1, 1}, []int{1, 1}, true},
		{[]int{1, 2}, []int{1, 2}, true},
		{[]int{1, 2}, []int{2, 1}, true},
		{[2]int{1, 2}, [2]int{2, 1}, true},
		{[]string{"hello", "world"}, []string{"world", "hello"}, true},
		{[]string{"hello", "hello"}, []string{"hello", "hello"}, true},
		{[]string{"hello", "hello", "world"}, []string{"hello", "world", "hello"}, true},
		{[3]string{"hello", "hello", "world"}, [3]string{"hello", "world", "hello"}, true},
		{[]int{}, nil, true},

		// not matching
		{[]int{1}, []int{1, 1}, false},
		{[]int{1, 2}, []int{2, 2}, false},
		{[]string{"hello", "hello"}, []string{"hello"}, false},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("ElementsMatch(%#v, %#v)", c.expected, c.actual), func(t *testing.T) {
			res := ElementsMatch(mockT, c.actual, c.expected)

			if res != c.result {
				t.Errorf("ElementsMatch(%#v, %#v) should return %v", c.actual, c.expected, c.result)
			}
		})
	}
}

func TestDiffLists(t *testing.T) {
	tests := []struct {
		name   string
		listA  interface{}
		listB  interface{}
		extraA []interface{}
		extraB []interface{}
	}{
		{
			name:   "equal empty",
			listA:  []string{},
			listB:  []string{},
			extraA: nil,
			extraB: nil,
		},
		{
			name:   "equal same order",
			listA:  []string{"hello", "world"},
			listB:  []string{"hello", "world"},
			extraA: nil,
			extraB: nil,
		},
		{
			name:   "equal different order",
			listA:  []string{"hello", "world"},
			listB:  []string{"world", "hello"},
			extraA: nil,
			extraB: nil,
		},
		{
			name:   "extra A",
			listA:  []string{"hello", "hello", "world"},
			listB:  []string{"hello", "world"},
			extraA: []interface{}{"hello"},
			extraB: nil,
		},
		{
			name:   "extra A twice",
			listA:  []string{"hello", "hello", "hello", "world"},
			listB:  []string{"hello", "world"},
			extraA: []interface{}{"hello", "hello"},
			extraB: nil,
		},
		{
			name:   "extra B",
			listA:  []string{"hello", "world"},
			listB:  []string{"hello", "hello", "world"},
			extraA: nil,
			extraB: []interface{}{"hello"},
		},
		{
			name:   "extra B twice",
			listA:  []string{"hello", "world"},
			listB:  []string{"hello", "hello", "world", "hello"},
			extraA: nil,
			extraB: []interface{}{"hello", "hello"},
		},
		{
			name:   "integers 1",
			listA:  []int{1, 2, 3, 4, 5},
			listB:  []int{5, 4, 3, 2, 1},
			extraA: nil,
			extraB: nil,
		},
		{
			name:   "integers 2",
			listA:  []int{1, 2, 1, 2, 1},
			listB:  []int{2, 1, 2, 1, 2},
			extraA: []interface{}{1},
			extraB: []interface{}{2},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			actualExtraA, actualExtraB := diffLists(test.listA, test.listB)
			Equal(t, test.extraA, actualExtraA, "extra A does not match for listA=%v listB=%v",
				test.listA, test.listB)
			Equal(t, test.extraB, actualExtraB, "extra B does not match for listA=%v listB=%v",
				test.listA, test.listB)
		})
	}
}

func TestNoError(t *testing.T) {

	mockT := new(testing.T)

	// start with a nil error
	var err error

	True(t, NoError(mockT, err), "NoError should return True for nil arg")

	// now set an error
	err = errors.New("some error")

	False(t, NoError(mockT, err), "NoError with error should return False")

	// returning an empty error interface
	err = func() error {
		var err *customError
		return err
	}()

	if err == nil { // err is not nil here!
		t.Errorf("Error should be nil due to empty interface: %s", err)
	}

	False(t, NoError(mockT, err), "NoError should fail with empty error interface")
}

type customError struct{}

func (*customError) Error() string { return "fail" }

func TestError(t *testing.T) {

	mockT := new(testing.T)

	// start with a nil error
	var err error

	False(t, Error(mockT, err), "Error should return False for nil arg")

	// now set an error
	err = errors.New("some error")

	True(t, Error(mockT, err), "Error with error should return True")

	// go vet check
	True(t, Errorf(mockT, err, "example with %s", "formatted message"), "Errorf with error should rturn True")

	// returning an empty error interface
	err = func() error {
		var err *customError
		return err
	}()

	if err == nil { // err is not nil here!
		t.Errorf("Error should be nil due to empty interface: %s", err)
	}

	True(t, Error(mockT, err), "Error should pass with empty error interface")
}

func TestEqualError(t *testing.T) {
	mockT := new(testing.T)

	// start with a nil error
	var err error
	False(t, EqualError(mockT, err, ""),
		"EqualError should return false for nil arg")

	// now set an error
	err = errors.New("some error")
	False(t, EqualError(mockT, err, "Not some error"),
		"EqualError should return false for different error string")
	True(t, EqualError(mockT, err, "some error"),
		"EqualError should return true")
}

func TestErrorContains(t *testing.T) {
	mockT := new(testing.T)

	// start with a nil error
	var err error
	False(t, ErrorContains(mockT, err, ""),
		"ErrorContains should return false for nil arg")

	// now set an error
	err = errors.New("some error: another error")
	False(t, ErrorContains(mockT, err, "bad error"),
		"ErrorContains should return false for different error string")
	True(t, ErrorContains(mockT, err, "some error"),
		"ErrorContains should return true")
	True(t, ErrorContains(mockT, err, "another error"),
		"ErrorContains should return true")
}

func Test_isEmpty(t *testing.T) {

	chWithValue := make(chan struct{}, 1)
	chWithValue <- struct{}{}

	True(t, isEmpty(""))
	True(t, isEmpty(nil))
	True(t, isEmpty([]string{}))
	True(t, isEmpty(0))
	True(t, isEmpty(int32(0)))
	True(t, isEmpty(int64(0)))
	True(t, isEmpty(false))
	True(t, isEmpty(map[string]string{}))
	True(t, isEmpty(new(time.Time)))
	True(t, isEmpty(time.Time{}))
	True(t, isEmpty(make(chan struct{})))
	True(t, isEmpty([1]int{}))
	False(t, isEmpty("something"))
	False(t, isEmpty(errors.New("something")))
	False(t, isEmpty([]string{"something"}))
	False(t, isEmpty(1))
	False(t, isEmpty(true))
	False(t, isEmpty(map[string]string{"Hello": "World"}))
	False(t, isEmpty(chWithValue))
	False(t, isEmpty([1]int{42}))
}

func Test_getLen(t *testing.T) {
	falseCases := []interface{}{
		nil,
		0,
		true,
		false,
		'A',
		struct{}{},
	}
	for _, v := range falseCases {
		ok, l := getLen(v)
		False(t, ok, "Expected getLen fail to get length of %#v", v)
		Equal(t, 0, l, "getLen should return 0 for %#v", v)
	}

	ch := make(chan int, 5)
	ch <- 1
	ch <- 2
	ch <- 3
	trueCases := []struct {
		v interface{}
		l int
	}{
		{[]int{1, 2, 3}, 3},
		{[...]int{1, 2, 3}, 3},
		{"ABC", 3},
		{map[int]int{1: 2, 2: 4, 3: 6}, 3},
		{ch, 3},

		{[]int{}, 0},
		{map[int]int{}, 0},
		{make(chan int), 0},

		{[]int(nil), 0},
		{map[int]int(nil), 0},
		{(chan int)(nil), 0},
	}

	for _, c := range trueCases {
		ok, l := getLen(c.v)
		True(t, ok, "Expected getLen success to get length of %#v", c.v)
		Equal(t, c.l, l)
	}
}

func TestLen(t *testing.T) {
	mockT := new(testing.T)

	False(t, Len(mockT, nil, 0), "nil does not have length")
	False(t, Len(mockT, 0, 0), "int does not have length")
	False(t, Len(mockT, true, 0), "true does not have length")
	False(t, Len(mockT, false, 0), "false does not have length")
	False(t, Len(mockT, 'A', 0), "Rune does not have length")
	False(t, Len(mockT, struct{}{}, 0), "Struct does not have length")

	ch := make(chan int, 5)
	ch <- 1
	ch <- 2
	ch <- 3

	cases := []struct {
		v interface{}
		l int
	}{
		{[]int{1, 2, 3}, 3},
		{[...]int{1, 2, 3}, 3},
		{"ABC", 3},
		{map[int]int{1: 2, 2: 4, 3: 6}, 3},
		{ch, 3},

		{[]int{}, 0},
		{map[int]int{}, 0},
		{make(chan int), 0},

		{[]int(nil), 0},
		{map[int]int(nil), 0},
		{(chan int)(nil), 0},
	}

	for _, c := range cases {
		True(t, Len(mockT, c.v, c.l), "%#v have %d items", c.v, c.l)
	}

	cases = []struct {
		v interface{}
		l int
	}{
		{[]int{1, 2, 3}, 4},
		{[...]int{1, 2, 3}, 2},
		{"ABC", 2},
		{map[int]int{1: 2, 2: 4, 3: 6}, 4},
		{ch, 2},

		{[]int{}, 1},
		{map[int]int{}, 1},
		{make(chan int), 1},

		{[]int(nil), 1},
		{map[int]int(nil), 1},
		{(chan int)(nil), 1},
	}

	for _, c := range cases {
		False(t, Len(mockT, c.v, c.l), "%#v have %d items", c.v, c.l)
	}
}

func TestWithinDuration(t *testing.T) {

	mockT := new(testing.T)
	a := time.Now()
	b := a.Add(10 * time.Second)

	True(t, WithinDuration(mockT, a, b, 10*time.Second), "A 10s difference is within a 10s time difference")
	True(t, WithinDuration(mockT, b, a, 10*time.Second), "A 10s difference is within a 10s time difference")

	False(t, WithinDuration(mockT, a, b, 9*time.Second), "A 10s difference is not within a 9s time difference")
	False(t, WithinDuration(mockT, b, a, 9*time.Second), "A 10s difference is not within a 9s time difference")

	False(t, WithinDuration(mockT, a, b, -9*time.Second), "A 10s difference is not within a 9s time difference")
	False(t, WithinDuration(mockT, b, a, -9*time.Second), "A 10s difference is not within a 9s time difference")

	False(t, WithinDuration(mockT, a, b, -11*time.Second), "A 10s difference is not within a 9s time difference")
	False(t, WithinDuration(mockT, b, a, -11*time.Second), "A 10s difference is not within a 9s time difference")
}

func TestInDelta(t *testing.T) {
	mockT := new(testing.T)

	True(t, InDelta(mockT, 1.001, 1, 0.01), "|1.001 - 1| <= 0.01")
	True(t, InDelta(mockT, 1, 1.001, 0.01), "|1 - 1.001| <= 0.01")
	True(t, InDelta(mockT, 1, 2, 1), "|1 - 2| <= 1")
	False(t, InDelta(mockT, 1, 2, 0.5), "Expected |1 - 2| <= 0.5 to fail")
	False(t, InDelta(mockT, 2, 1, 0.5), "Expected |2 - 1| <= 0.5 to fail")
	False(t, InDelta(mockT, "", nil, 1), "Expected non numerals to fail")
	False(t, InDelta(mockT, 42, math.NaN(), 0.01), "Expected NaN for actual to fail")
	False(t, InDelta(mockT, math.NaN(), 42, 0.01), "Expected NaN for expected to fail")
	True(t, InDelta(mockT, math.NaN(), math.NaN(), 0.01), "Expected NaN for both to pass")

	cases := []struct {
		a, b  interface{}
		delta float64
	}{
		{uint(2), uint(1), 1},
		{uint8(2), uint8(1), 1},
		{uint16(2), uint16(1), 1},
		{uint32(2), uint32(1), 1},
		{uint64(2), uint64(1), 1},

		{int(2), int(1), 1},
		{int8(2), int8(1), 1},
		{int16(2), int16(1), 1},
		{int32(2), int32(1), 1},
		{int64(2), int64(1), 1},

		{float32(2), float32(1), 1},
		{float64(2), float64(1), 1},
	}

	for _, tc := range cases {
		True(t, InDelta(mockT, tc.a, tc.b, tc.delta), "Expected |%V - %V| <= %v", tc.a, tc.b, tc.delta)
	}
}

type diffTestingStruct struct {
	A string
	B int
}

func (d *diffTestingStruct) String() string {
	return d.A
}

func TestDiff(t *testing.T) {
	expected := `

Diff:
--- Expected
+++ Actual
@@ -1,3 +1,3 @@
 (struct { foo string }) {
- foo: (string) (len=5) "hello"
+ foo: (string) (len=3) "bar"
 }
`
	actual := diff(
		struct{ foo string }{"hello"},
		struct{ foo string }{"bar"},
	)
	Equal(t, expected, actual)

	expected = `

Diff:
--- Expected
+++ Actual
@@ -2,5 +2,5 @@
  (int) 1,
- (int) 2,
  (int) 3,
- (int) 4
+ (int) 5,
+ (int) 7
 }
`
	actual = diff(
		[]int{1, 2, 3, 4},
		[]int{1, 3, 5, 7},
	)
	Equal(t, expected, actual)

	expected = `

Diff:
--- Expected
+++ Actual
@@ -2,4 +2,4 @@
  (int) 1,
- (int) 2,
- (int) 3
+ (int) 3,
+ (int) 5
 }
`
	actual = diff(
		[]int{1, 2, 3, 4}[0:3],
		[]int{1, 3, 5, 7}[0:3],
	)
	Equal(t, expected, actual)

	expected = `

Diff:
--- Expected
+++ Actual
@@ -1,6 +1,6 @@
 (map[string]int) (len=4) {
- (string) (len=4) "four": (int) 4,
+ (string) (len=4) "five": (int) 5,
  (string) (len=3) "one": (int) 1,
- (string) (len=5) "three": (int) 3,
- (string) (len=3) "two": (int) 2
+ (string) (len=5) "seven": (int) 7,
+ (string) (len=5) "three": (int) 3
 }
`

	actual = diff(
		map[string]int{"one": 1, "two": 2, "three": 3, "four": 4},
		map[string]int{"one": 1, "three": 3, "five": 5, "seven": 7},
	)
	Equal(t, expected, actual)

	expected = `

Diff:
--- Expected
+++ Actual
@@ -1,3 +1,3 @@
 (*errors.errorString)({
- s: (string) (len=19) "some expected error"
+ s: (string) (len=12) "actual error"
 })
`

	actual = diff(
		errors.New("some expected error"),
		errors.New("actual error"),
	)
	Equal(t, expected, actual)

	expected = `

Diff:
--- Expected
+++ Actual
@@ -2,3 +2,3 @@
  A: (string) (len=11) "some string",
- B: (int) 10
+ B: (int) 15
 }
`

	actual = diff(
		diffTestingStruct{A: "some string", B: 10},
		diffTestingStruct{A: "some string", B: 15},
	)
	Equal(t, expected, actual)

	expected = `

Diff:
--- Expected
+++ Actual
@@ -1,2 +1,2 @@
-(time.Time) 2020-09-24 00:00:00 +0000 UTC
+(time.Time) 2020-09-25 00:00:00 +0000 UTC
 
`

	actual = diff(
		time.Date(2020, 9, 24, 0, 0, 0, 0, time.UTC),
		time.Date(2020, 9, 25, 0, 0, 0, 0, time.UTC),
	)
	Equal(t, expected, actual)
}

func TestTimeEqualityErrorFormatting(t *testing.T) {
	mockT := new(mockTestingT)

	Equal(mockT, time.Second*2, time.Millisecond)
}

func TestDiffEmptyCases(t *testing.T) {
	Equal(t, "", diff(nil, nil))
	Equal(t, "", diff(struct{ foo string }{}, nil))
	Equal(t, "", diff(nil, struct{ foo string }{}))
	Equal(t, "", diff(1, 2))
	Equal(t, "", diff(1, 2))
	Equal(t, "", diff([]int{1}, []bool{true}))
}

// Ensure there are no data races
func TestDiffRace(t *testing.T) {
	t.Parallel()

	expected := map[string]string{
		"a": "A",
		"b": "B",
		"c": "C",
	}

	actual := map[string]string{
		"d": "D",
		"e": "E",
		"f": "F",
	}

	// run diffs in parallel simulating tests with t.Parallel()
	numRoutines := 10
	rChans := make([]chan string, numRoutines)
	for idx := range rChans {
		rChans[idx] = make(chan string)
		go func(ch chan string) {
			defer close(ch)
			ch <- diff(expected, actual)
		}(rChans[idx])
	}

	for _, ch := range rChans {
		for msg := range ch {
			NotEqual(t, msg, "") // dummy assert
		}
	}
}

type mockTestingT struct {
	errorFmt string
	args     []interface{}
}

func (m *mockTestingT) errorString() string {
	return fmt.Sprintf(m.errorFmt, m.args...)
}

func (m *mockTestingT) Errorf(format string, args ...interface{}) {
	m.errorFmt = format
	m.args = args
}

type mockFailNowTestingT struct {
}

func (m *mockFailNowTestingT) Errorf(format string, args ...interface{}) {}

func (m *mockFailNowTestingT) FailNow() {}

func TestBytesEqual(t *testing.T) {
	var cases = []struct {
		a, b []byte
	}{
		{make([]byte, 2), make([]byte, 2)},
		{make([]byte, 2), make([]byte, 2, 3)},
		{nil, make([]byte, 0)},
	}
	for i, c := range cases {
		Equal(t, reflect.DeepEqual(c.a, c.b), ObjectsAreEqual(c.a, c.b), "case %d failed", i+1)
	}
}

func BenchmarkBytesEqual(b *testing.B) {
	const size = 1024 * 8
	s := make([]byte, size)
	for i := range s {
		s[i] = byte(i % 255)
	}
	s2 := make([]byte, size)
	copy(s2, s)

	mockT := &mockFailNowTestingT{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Equal(mockT, s, s2)
	}
}

func BenchmarkNotNil(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NotNil(b, b)
	}
}

func TestEventuallyFalse(t *testing.T) {
	mockT := new(testing.T)

	condition := func() bool {
		return false
	}

	False(t, Eventually(mockT, condition, 100*time.Millisecond, 20*time.Millisecond))
}

func TestEventuallyTrue(t *testing.T) {
	state := 0
	condition := func() bool {
		defer func() {
			state++
		}()
		return state == 2
	}

	True(t, Eventually(t, condition, 100*time.Millisecond, 20*time.Millisecond))
}

func Test_validateEqualArgs(t *testing.T) {
	if validateEqualArgs(func() {}, func() {}) == nil {
		t.Error("non-nil functions should error")
	}

	if validateEqualArgs(func() {}, func() {}) == nil {
		t.Error("non-nil functions should error")
	}

	if validateEqualArgs(nil, nil) != nil {
		t.Error("nil functions are equal")
	}
}

func Test_truncatingFormat(t *testing.T) {

	original := strings.Repeat("a", bufio.MaxScanTokenSize-102)
	result := truncatingFormat(original)
	Equal(t, fmt.Sprintf("%#v", original), result, "string should not be truncated")

	original = original + "x"
	result = truncatingFormat(original)
	NotEqual(t, fmt.Sprintf("%#v", original), result, "string should have been truncated.")

	if !strings.HasSuffix(result, "<... truncated>") {
		t.Error("truncated string should have <... truncated> suffix")
	}
}
