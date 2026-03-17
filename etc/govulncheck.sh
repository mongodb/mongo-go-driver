#!/usr/bin/env bash
set -ex

# Use a specific Go version so that local govulncheck results are consistent
# with CI results.
#
# Note: this needs to be updated if the listed Go version has vulnerabilities
# discovered because they will show up in the scan results along with Go Driver
# and dependency vulnerabilities.
BASE_VERSION=1.25
GO_VERSION=$(curl -s 'https://go.dev/dl/?mode=json'\
    | jq -r '.[].version'\
    | grep -m 1 -E "^go${BASE_VERSION}")

go install golang.org/dl/$GO_VERSION@latest
${GO_VERSION} download
go install golang.org/x/vuln/cmd/govulncheck@latest

# govulncheck uses the Go binary it finds from the PATH, so modify PATH to point
# to the Go version we just downloaded.
PATH="$(${GO_VERSION} env GOROOT)/bin:$PATH" govulncheck -show verbose ./...
