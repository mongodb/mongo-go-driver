package main

import (
	"go/types"
	"sort"
)

type Field struct {
	ftype         TagType
	field         *types.Var
	fnname        string
	name          string
	doc           []string
	pointerExempt bool
	variadic      bool
	knownType     KnownType
}

type FieldSlice []Field

var _ sort.Interface = (FieldSlice)(nil)

// fieldSliceFromMap converts a map into a FieldSlice. The returned FieldSlice is sorted by Field.name.
func fieldSliceFromMap(m map[string]Field) FieldSlice {
	fs := make(FieldSlice, 0, len(m))
	for _, field := range m {
		fs = append(fs, field)
	}
	sort.Sort(fs)
	return fs
}

func (f FieldSlice) Len() int           { return len(f) }
func (f FieldSlice) Less(i, j int) bool { return f[i].name < f[j].name }
func (f FieldSlice) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }
