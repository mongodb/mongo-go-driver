// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package binaryutil provides functions for reading binary primitives from
// byte slices. It is used internally for BSON parsing and wire protocol
// operations.
//
// The functions in this package are designed for use in BSON operations.
// Signed integer functions (ReadI32, ReadI64) use manual bit-shifting rather
// than encoding/binary to avoid unsafe signed/unsigned conversions and comply
// with gosec G115. Bounds-check elimination (BCE) hints help the compiler
// inline these functions.
//
// Benchmarking across different ARM64 architectures (Apple M-series)
// revealed non-deterministic performance discrepancies between using the
// "encoding/binary" standard library and manual bit-shifting ("straight-lining").
//
// Without Loss of Generality (WLOG), benchmarking observed that:
//   - On Apple M1 Pro: Standard library (ReadU32) outperformed manual
//     bit-shifting (ReadI32) by ~2x (~0.08ns vs ~0.16ns).
//   - On Apple M4 Max: Manual bit-shifting (ReadI32) outperformed the
//     standard library (ReadU32) by ~1.6x (~0.03ns vs ~0.05ns).
//
// Further testing showed that "straight-lining" the ReadU32 implementation
// to match ReadI32 normalized performance to ~0.03ns on the M4 Max, even
// though the generated assembly for both approaches is virtually equivalent.
//
// The generated assembly is nearly identical for both approaches. These
// sub-nanosecond variations likely stem from microarchitecture differences
// (instruction caching, branch prediction) rather than the code itself.
//
// Since network I/O dominates driver latency, these differences do not have a
// significant impact on driver performance. The implementation favors security
// compliance and readability over hardware-specific tuning.
package binaryutil
