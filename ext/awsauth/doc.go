// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// This package allows credential providers in AWS SDK v2 to be supplied to the
// MongoDB Go Driver for use with the MONGODB-AWS authentication mechanism.
//
// # Pre-release and unstable
//
// This module is pre-release software. Its API is unstable and may change in
// backward-incompatible ways, or be removed entirely, without notice. It is not
// yet covered by the MongoDB Go Driver's semantic-versioning guarantees. Pin an
// exact version and review the changelog before upgrading.
//
// # No dependency on the Go Driver
//
// This module is a separate Go module that intentionally does not take a
// dependency on the MongoDB Go Driver. Rather than importing the driver, it
// adapts to the driver's options.AWSCredentialsProvider interface structurally:
// the exported types in this package are defined to match that interface, so the
// adapter satisfies it without an import.
//
// This decoupling exists for two reasons:
//
//   - It keeps the AWS SDK out of the main driver's dependency graph. Users who
//     don't need AWS authentication never pull in the AWS SDK transitively.
//   - It lets the two modules version independently and avoids a circular
//     dependency between the driver and this adapter.
//
// NewCredentialsProvider() adapts an AWS CredentialsProvider to be used in:
//
//	ClientOptions
//	ClientEncryptionOptions
//	AutoEncryptionOptions
package awsauth
