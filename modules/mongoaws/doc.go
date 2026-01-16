// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// This package allows credential providers and signers in v2 AWS SDK to be
// supplied to the MongoDB Go Driver for use with the MONGODB-AWS authentication
// mechanism.
//
// NewCredentialsProvider() adapts an AWS CredentialsProvider to be used in:
//
//	ClientOptions
//	ClinetEncryptionOptions
//	AutoEncryptionOptions
//
// NewSigner() adapts an AWS HTTPSigner to be used in:
//
//	ClientOptions
package mongoaws
