#!/usr/bin/env bash
# api-report
# Generates a report of Go Driver API changes for the current branch.
set -eux

cmd=$(command -v gorelease || true)

if [ -z $cmd ]; then
    go install golang.org/x/exp/cmd/gorelease@latest
fi

branch=${GITHUB_BASE_REF:-master}
sha=$(git merge-base $branch HEAD)

sha=3efbd23a061a4069af96e56b894f4f72bcd51999
gorelease -base=$sha > api-report.txt || true

go run ./cmd/parse-api-report/main.go

rm api-report.txt