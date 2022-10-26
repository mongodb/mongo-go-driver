# Contributing to the MongoDB Go Driver

Thank you for your interest in contributing to the MongoDB Go Driver!

We are building this software together and strongly encourage contributions from the community that are within the guidelines set forth below.

## Bug Fixes and New Features

Before starting to write code, look for existing [tickets](https://jira.mongodb.org/browse/GODRIVER) or [create one](https://jira.mongodb.org/secure/CreateIssue!default.jspa) for your bug, issue, or feature request. This helps the community avoid working on something that might not be of interest or which has already been addressed.

## Pull Requests & Patches

The Go Driver team uses GitHub to manage and review all code changes. Patches should generally be made against the master (default) branch and include relevant tests, if
applicable.

Code should compile and tests should pass under all Go versions which the driver currently supports. Currently the Go Driver supports a minimum version of Go 1.13 and requires Go 1.18 for development. Please run the following Make targets to validate your changes:
- `make fmt`
- `make lint` (requires [golangci-lint](https://github.com/golangci/golangci-lint) and [lll](https://github.com/walle/lll) to be installed and available in the `PATH`)
- `make test`
- `make test-race`

**Running the tests requires that you have a `mongod` server running on localhost, listening on the default port (27017). At minimum, please test against the latest release version of the MongoDB server.**

If any tests do not pass, or relevant tests are not included, the patch will not be considered.

If you are working on a bug or feature listed in Jira, please include the ticket number prefixed with GODRIVER in the commit message and GitHub pull request title, (e.g. GODRIVER-123). For the patch commit message itself, please follow the [How to Write a Git Commit Message](https://chris.beams.io/posts/git-commit/) guide.

## Talk To Us

If you want to work on the driver, write documentation, or have questions/complaints, please reach out to us either via [MongoDB Community Forums](https://community.mongodb.com/tags/c/drivers-odms-connectors/7/go-driver) or by creating a Question issue in [Jira](https://jira.mongodb.org/secure/CreateIssue!default.jspa).
