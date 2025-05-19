# Contributing to the MongoDB Go Driver

Thank you for your interest in contributing to the MongoDB Go Driver!

We are building this software together and strongly encourage contributions from the community that are within the guidelines set forth below.

## Requirements

Go 1.23 or higher is required to run the driver test suite.  We use [task](https://taskfile.dev/) as our task runner.

## Bug Fixes and New Features

Before starting to write code, look for existing [tickets](https://jira.mongodb.org/browse/GODRIVER) or [create one](https://jira.mongodb.org/secure/CreateIssue!default.jspa) for your bug, issue, or feature request. This helps the community avoid working on something that might not be of interest or which has already been addressed.

## Pull Requests & Patches

The Go Driver team uses GitHub to manage and review all code changes. Patches should generally be made against the master (default) branch and include relevant tests, if
applicable.

Code should compile and tests should pass under all Go versions which the driver currently supports. Currently the Go Driver supports a minimum version of Go 1.19 and requires Go 1.23 for development. Please run the following `Taskfile` targets to validate your changes:

- `task fmt`
- `task lint`
- `task test`
- `task test-race`

**Running the tests requires that you have a `mongod` server running on localhost, listening on the default port (27017). At minimum, please test against the latest release version of the MongoDB server.**

If any tests do not pass, or relevant tests are not included, the patch will not be considered.

If you are working on a bug or feature listed in Jira, please include the ticket number prefixed with GODRIVER in the commit message and GitHub pull request title, (e.g. GODRIVER-123). For the patch commit message itself, please follow the [How to Write a Git Commit Message](https://chris.beams.io/posts/git-commit/) guide.

### Linting on commit

The Go team uses [pre-commit](https://pre-commit.com/#installation) to lint both source and text files.

To install locally, run:

```bash
brew install pre-commit
pre-commit install
```

After that, the checks will run on any changed files when committing.  To manually run the checks on all files, run:

```bash
pre-commit run --all-files
```

### Merge up GitHub Action

PR [#1962](https://github.com/mongodb/mongo-go-driver/pull/1962) added the "Merge up" GitHub Actions workflow to automatically roll changes from old branches into new ones. This section outlines how this process works.

#### Regression

If a regression is identified in an older branch, the fix should be applied directly to the latest
release branch. Once the pull request with the fix is merged into latest, the "Merge up" GitHub Action will
automatically create a pull request to merge these changes into the master branch. This ensures that all bug fixes are
incorporated into the latest codebase and actively supported versions.

For example, suppose we have four minor release branches: release/2.0, release/2.1, release/2.2, and release/2.3. If a
regression is found in the release/2.1 branch, you would create a pull request to fix the issue in the latest supported
branch, release/2.3. Once this pull request is merged, the "Merge up" GitHub Action will automatically create a pull
request to merge the changes from release/2.3 into the master branch. Then you can proceed to release release/2.3.latest+1.

```mermaid
gitGraph
   commit tag: "Initial main setup"

   branch release/2.0
   checkout release/2.0
   commit tag: "Initial release/2.0"

   checkout main
   branch release/2.1
   checkout release/2.1
   commit tag: "Bug introduced"

   checkout main
   branch release/2.2
   checkout release/2.2
   commit tag: "Initial release/2.2"

   checkout main
   branch release/2.3
   checkout release/2.3
   commit tag: "Initial release/2.3"

   checkout release/2.1
   commit tag: "Bug found in release/2.1"

   checkout release/2.3
   commit tag: "Bug fix applied in release/2.3 (Manual PR)"

   checkout main
   merge release/2.3 tag: "Merge fix from release/2.3 into master (GitHub Actions)"
   commit
```

If necessary, it is also possible to apply the fix to the older branch where the bug was originally found. In our example,
once the pull request is merged into release/2.1, the "Merge up" GitHub Action will initiate a series of pull requests
to roll the fix forward: first into release/2.2, then into release/2.3, and finally into master. This process makes sure
that the change cascades through every intermediate supported version.

```mermaid
gitGraph
   commit tag: "Initial main setup"

   branch release/2.0
   checkout release/2.0
   commit tag: "Initial release/2.0"

   checkout main
   branch release/2.1
   checkout release/2.1
   commit tag: "Bug introduced"

   checkout main
   branch release/2.2
   checkout release/2.2
   commit tag: "Initial release/2.2"

   checkout main
   branch release/2.3
   checkout release/2.3
   commit tag: "Initial release/2.3"

   checkout release/2.1
   commit tag: "Bug fix in release/2.1 (Manual PR)"

   checkout release/2.2
   merge release/2.1 tag: "Merge fix from release/2.1 (GitHub Actions)"
   commit

   checkout release/2.3
   merge release/2.2 tag: "Merge updates from release/2.2 (GitHub Actions)"
   commit

   checkout main
   merge release/2.3 tag: "Merge updates from release/2.3 (GitHub Actions)"
   commit
```

#### Pull Request Management

When the "Merge up" GitHub Action is enabled, multiple merge-up pull requests (such as PR1, PR2, and PR3) can be
automatically created at the same time for different bug fixes or features that all target, for example, the
release/2.x branch. At first, PR1, PR2, and PR3 exist side by side—each handling separate changes. When PR1 and PR2 are
closed, the Action automatically combines their changes into PR3. This final PR3 then contains all updates,
allowing you to merge everything into release/2.x+1 in a single, streamlined step.

```mermaid
flowchart LR
   A[PR1: Merge up from release/2.x] --> B[Close PR1]
   C[PR2: Merge up from release/2.x] --> D[Close PR2]

   B --> E[PR3: Consolidated Final Pull Request]
   D --> E
    E --> F[release/2.x+1]
    B[Close PR1]
   D[Close PR2]
   E[PR3: Includes changes from both PR1 and PR2]
```

#### Evergreen Config Merge Strategy

Changes to the testing workflow should persist through all releases in a major version.

### Cherry-picking between branches

#### Using the GitHub App

Within a PR, you can make the comment:

```
drivers-pr-bot please backport to {target_branch}
```

The preferred workflow is to make the comment and then merge the PR.

If you merge the PR and the "backport-pr" task runs before you make the comment, you can
make the comment and then re-run the "backport-pr" task for that commit.

#### Manually

You must first install the `gh` cli (`brew install gh`), then set your GitHub username:

```bash
git config --global github.user <github_handle>
```

If a Pull Request needs to be cherry-picked to a new branch, get the sha of the commit in the base branch, and then run

```bash
task cherry-picker -- <sha>
```

By default it will use `master` as the target branch.  The branch can be specified as the second argument, e.g.

```bash
task cherry-picker -- <sha> branch
```

It will create a new checkout in a temp dir, create a new branch, perform the cherry-pick, and then
prompt before creating a PR to the target branch.

## Testing / Development

The driver tests can be run against several database configurations. The most simple configuration is a standalone mongod with no auth, no ssl, and no compression. To run these basic driver tests, make sure a standalone MongoDB server instance is running at localhost:27017. To run the tests, you can run `task`. This will run coverage, run go-lint, run go-vet, and build the examples.

You can install `libmongocrypt` locally by running `task install-libmongocrypt`, which will create an `install` directory
in the repository top level directory.  On Windows you will also need to add `c:/libmongocrypt/` to your `PATH`.

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
1. [Docker](https://www.docker.com/products/docker-desktop/)

Local testing requires exporting the `MONGODB_URI` environment variables. To build the AWS Lambda image and invoke the `MongoDBFunction` lambda function use the `build-faas-awslambda` Taskfile target:

```bash
MONGODB_URI="mongodb://host.docker.internal:27017" task build-faas-awslambda
```

The usage of host.docker.internal comes from the [Docker networking documentation](https://docs.docker.com/desktop/networking/#i-want-to-connect-from-a-container-to-a-service-on-the-host).

There is currently no arm64 support for the go1.x runtime, see [here](https://docs.aws.amazon.com/lambda/latest/dg/lambda-runtimes.html). Known issues running on linux/arm64 include the inability to network with the localhost from the public.ecr.aws/lambda/go Docker image.

### Encryption Tests

Most of the tests requiring `libmongocrypt` can be run using the Docker workflow.

However, some of the tests require secrets handling.  Please see the team [Wiki](https://wiki.corp.mongodb.com/pages/viewpage.action?spaceKey=DRIVERS&title=Testing+CSFLE) for more information.

The test suite can be run with or without the secrets as follows:

```bash
task setup-env
task setup-test
task evg-test-versioned-api
```

### Load Balancer

To launch the load balancer on MacOS, run the following.

- `brew install haproxy`
- Clone drivers-evergreen-tools and save the path as `DRIVERS_TOOLS`.
- Start the servers using (or use the docker-based method below):

```bash
LOAD_BALANCER=true TOPOLOGY=sharded_cluster AUTH=noauth SSL=nossl MONGODB_VERSION=6.0 DRIVERS_TOOLS=$PWD/drivers-evergreen-tools MONGO_ORCHESTRATION_HOME=$PWD/drivers-evergreen-tools/.evergreen/orchestration $PWD/drivers-evergreen-tools/.evergreen/run-orchestration.sh
```

- Start the load balancer using:

```bash
MONGODB_URI='mongodb://localhost:27017,localhost:27018/' $PWD/drivers-evergreen-tools/.evergreen/run-load-balancer.sh start
```

- Run the load balancer tests (or use the docker runner below with `evg-test-load-balancers`):

```bash
task evg-test-load-balancers
```

### Testing in Docker

We support local testing in Docker.  To test using docker, you will need to set the `DRIVERS_TOOLs` environment variable to point to a local clone of the drivers-evergreen-tools repository. This is essential for running the testing matrix in a container. You can set the `DRIVERS_TOOLS` variable in your shell profile or in your project-specific environment.

1. First, start the drivers-tools server docker container, as:

```bash
bash $DRIVERS_TOOLS/.evergreen/docker/start-server.sh
```

See the readme in `$DRIVERS_TOOLS/.evergreen/docker` for more information on usage.

2. Next, start any other required services in another terminal, like a load balancer.

1. Finally, run the Go Driver tests using the following script in this repo:

```bash
make run-docker
```

The script takes an optional argument for the `TASKFILE_TARGET` and allows for some environment variable overrides.
The docker container has the required binaries, including libmongocrypt.
The entry script executes the desired `TASKFILE_TARGET`.

For example, to test against a sharded cluster (make sure you started the server with a sharded_cluster),
using enterprise auth, run:

```bash
TOPOLOGY=sharded_cluster task run-docker -- evg-test-enterprise-auth
```

## Talk To Us

If you want to work on the driver, write documentation, or have questions/complaints, please reach out to us either via [MongoDB Community Forums](https://www.mongodb.com/community/forums/tag/go-driver) or by creating a Question issue in [Jira](https://jira.mongodb.org/secure/CreateIssue!default.jspa).
