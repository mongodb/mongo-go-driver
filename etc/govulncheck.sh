#!/usr/bin/env bash
set -ex

# Use a specific Go version so that local govulncheck results are consistent
# with CI results.
#
# Note: this needs to be updated if the listed Go version has vulnerabilities
# discovered because they will show up in the scan results along with Go Driver
# and dependency vulnerabilities.
GO_VERSION=1.24.5

go install golang.org/dl/go$GO_VERSION@latest
go${GO_VERSION} download
go install golang.org/x/vuln/cmd/govulncheck@latest

# govulncheck uses the Go binary it finds from the PATH, so modify PATH to point
# to the Go version we just downloaded.
PATH="$(go${GO_VERSION} env GOROOT)/bin:$PATH" govulncheck -show verbose ./...
