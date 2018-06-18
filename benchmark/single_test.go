package benchmark

import "testing"

func BenchmarkSingleRunCommand(b *testing.B)          { WrapCase(SingleRunCommand)(b) }
func BenchmarkSingleFindOneByID(b *testing.B)         { WrapCase(SingleFindOneByID)(b) }
func BenchmarkSingleInsertSmallDocument(b *testing.B) { WrapCase(SingleInsertSmallDocument)(b) }
func BenchmarkSingleInsertLargeDocument(b *testing.B) { WrapCase(SingleInsertLargeDocument)(b) }
