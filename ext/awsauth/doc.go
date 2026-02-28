// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// This package allows credential providers and signers in AWS SDK v2 to be
// supplied to the MongoDB Go Driver for use with the MONGODB-AWS authentication
// mechanism.
//
// This package is a separate module to avoid adding the AWS SDK as a dependency
// of the main driver. Users who don't need AWS authentication won't need to
// import the AWS SDK.
//
// NewCredentialsProvider() adapts an AWS CredentialsProvider to be used in:
//
//	ClientOptions
//	ClientEncryptionOptions
//	AutoEncryptionOptions
//
// NewSigner() adapts an AWS HTTPSigner to be used in:
//
//	ClientOptions
package awsauth
