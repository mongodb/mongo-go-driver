package benchmark

import "testing"

func BenchmarkMultiFindMany(b *testing.B)            { WrapCase(MultiFindMany)(b) }
func BenchmarkMultiInsertSmallDocument(b *testing.B) { WrapCase(MultiInsertSmallDocument)(b) }
func BenchmarkMultiInsertLargeDocument(b *testing.B) { WrapCase(MultiInsertLargeDocument)(b) }
