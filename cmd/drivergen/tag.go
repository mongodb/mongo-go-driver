package main

import (
	"reflect"
	"strings"
)

type TagType uint

const (
	Setter TagType = iota
	CommandName
	DocumentSequence
)

type Tag struct {
	Type           TagType
	Skip           bool
	Name           string
	PointerExempt  bool
	Variadic       bool
	ConstructorArg bool
	Retryable      bool
}

func parseTag(str string) Tag {
	tag, ok := reflect.StructTag(str).Lookup("drivergen")
	if !ok && !strings.Contains(string(str), ":") && len(str) > 0 {
		tag = str
	}
	var t Tag
	if tag == "-" {
		t.Skip = true
		return t
	}

	for idx, s := range strings.Split(tag, ",") {
		if idx == 0 && s != "" {
			t.Name = s
		}
		switch s {
		case "pointerExempt":
			t.PointerExempt = true
		case "variadic":
			t.Variadic = true
		case "commandName":
			t.Type = CommandName
		case "msgDocSeq":
			t.Type = DocumentSequence
		case "constructorArg":
			t.ConstructorArg = true
		case "retryable":
			t.Retryable = true
		}
	}

	return t
}
