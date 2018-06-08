package benchmark

import "testing"

func BenchmarkCanaryInc(b *testing.B)       { WrapCase(CanaryIncCase)(b) }
func BenchmarkGlobalCanaryInc(b *testing.B) { WrapCase(GlobalCanaryIncCase)(b) }
