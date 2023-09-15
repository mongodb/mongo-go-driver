# Contributing to the MongoDB Go Driver

Thank you for your interest in contributing to the MongoDB Go Driver!

We are building this software together and strongly encourage contributions from the community that are within the guidelines set forth below.

## Requirements

Go 1.20 or higher is required to run the driver test suite.

## Bug Fixes and New Features

Before starting to write code, look for existing [tickets](https://jira.mongodb.org/browse/GODRIVER) or [create one](https://jira.mongodb.org/secure/CreateIssue!default.jspa) for your bug, issue, or feature request. This helps the community avoid working on something that might not be of interest or which has already been addressed.

## Pull Requests & Patches

The Go Driver team uses GitHub to manage and review all code changes. Patches should generally be made against the master (default) branch and include relevant tests, if
applicable.

Code should compile and tests should pass under all Go versions which the driver currently supports. Currently the Go Driver supports a minimum version of Go 1.13 and requires Go 1.20 for development. Please run the following Make targets to validate your changes:
- `make fmt`
- `make lint` (requires [golangci-lint](https://github.com/golangci/golangci-lint) and [lll](https://github.com/walle/lll) to be installed and available in the `PATH`)
- `make test`
- `make test-race`

**Running the tests requires that you have a `mongod` server running on localhost, listening on the default port (27017). At minimum, please test against the latest release version of the MongoDB server.**

If any tests do not pass, or relevant tests are not included, the patch will not be considered.

If you are working on a bug or feature listed in Jira, please include the ticket number prefixed with GODRIVER in the commit message and GitHub pull request title, (e.g. GODRIVER-123). For the patch commit message itself, please follow the [How to Write a Git Commit Message](https://chris.beams.io/posts/git-commit/) guide.

## Testing / Development

The driver tests can be run against several database configurations. The most simple configuration is a standalone mongod with no auth, no ssl, and no compression. To run these basic driver tests, make sure a standalone MongoDB server instance is running at localhost:27017. To run the tests, you can run `make` (on Windows, run `nmake`). This will run coverage, run go-lint, run go-vet, and build the examples.

### Testing Different Topologies

To test a **replica set** or **sharded cluster**, set `MONGODB_URI="<connection-string>"` for the `make` command.
For example, for a local replica set named `rs1` comprised of three nodes on ports 27017, 27018, and 27019:

```
MONGODB_URI="mongodb://localhost:27017,localhost:27018,localhost:27019/?replicaSet=rs1" make
```

### Testing Auth and TLS

To test authentication and TLS, first set up a MongoDB cluster with auth and TLS configured. Testing authentication requires a user with the `root` role on the `admin` database. Here is an example command that would run a mongod with TLS correctly configured for tests. Either set or replace PATH_TO_SERVER_KEY_FILE and PATH_TO_CA_FILE with paths to their respective files:

```
mongod \
--auth \
--tlsMode requireTLS \
--tlsCertificateKeyFile $PATH_TO_SERVER_KEY_FILE \
--tlsCAFile $PATH_TO_CA_FILE \
--tlsAllowInvalidCertificates
```

To run the tests with `make`, set:
- `MONGO_GO_DRIVER_CA_FILE` to the location of the CA file used by the database
- `MONGO_GO_DRIVER_KEY_FILE` to the location of the client key file
- `MONGO_GO_DRIVER_PKCS8_ENCRYPTED_KEY_FILE` to the location of the pkcs8 client key file encrypted with the password string: `password`
- `MONGO_GO_DRIVER_PKCS8_UNENCRYPTED_KEY_FILE` to the location of the unencrypted pkcs8 key file
- `MONGODB_URI` to the connection string of the server
- `AUTH=auth`
- `SSL=ssl`

For example:

```
AUTH=auth SSL=ssl \
MONGO_GO_DRIVER_CA_FILE=$PATH_TO_CA_FILE \
MONGO_GO_DRIVER_KEY_FILE=$PATH_TO_CLIENT_KEY_FILE \
MONGO_GO_DRIVER_PKCS8_ENCRYPTED_KEY_FILE=$PATH_TO_ENCRYPTED_KEY_FILE \
MONGO_GO_DRIVER_PKCS8_UNENCRYPTED_KEY_FILE=$PATH_TO_UNENCRYPTED_KEY_FILE \
MONGODB_URI="mongodb://user:password@localhost:27017/?authSource=admin" \
make
```

Notes:
- The `--tlsAllowInvalidCertificates` flag is required on the server for the test suite to work correctly.
- The test suite requires the auth database to be set with `?authSource=admin`, not `/admin`.

### Testing Compression

The MongoDB Go Driver supports wire protocol compression using Snappy, zLib, or zstd. To run tests with wire protocol compression, set `MONGO_GO_DRIVER_COMPRESSOR` to `snappy`, `zlib`, or `zstd`.  For example:

```
MONGO_GO_DRIVER_COMPRESSOR=snappy make
```

Ensure the [`--networkMessageCompressors` flag](https://www.mongodb.com/docs/manual/reference/program/mongod/#cmdoption-mongod-networkmessagecompressors) on mongod or mongos includes `zlib` if testing zLib compression.

### Testing FaaS

The requirements for testing FaaS implementations in the Go Driver vary depending on the specific runtime.

#### AWS Lambda

The following are the requirements for running the AWS Lambda tests locally:

1. [AWS SAM CLI](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/install-sam-cli.html)
2. [Docker](https://www.docker.com/products/docker-desktop/)

Local testing requires exporting the `MONGODB_URI` environment variables. To build the AWS Lambda image and invoke the `MongoDBFunction` lambda function use the `build-faas-awslambda` make target:

```bash
MONGODB_URI="mongodb://host.docker.internal:27017" make build-faas-awslambda
```

The usage of host.docker.internal comes from the [Docker networking documentation](https://docs.docker.com/desktop/networking/#i-want-to-connect-from-a-container-to-a-service-on-the-host).

There is currently no arm64 support for the go1.x runtime, see [here](https://docs.aws.amazon.com/lambda/latest/dg/lambda-runtimes.html). Known issues running on linux/arm64 include the inability to network with the localhost from the public.ecr.aws/lambda/go Docker image.

## Talk To Us

If you want to work on the driver, write documentation, or have questions/complaints, please reach out to us either via [MongoDB Community Forums](https://community.mongodb.com/tags/c/drivers-odms-connectors/7/go-driver) or by creating a Question issue in [Jira](https://jira.mongodb.org/secure/CreateIssue!default.jspa).
