#!/usr/bin/env bash
# api-report
# Generates a report of Go Driver API changes for the current branch.
set -eux

cmd=$(command -v gorelease || true)

if [ -z $cmd ]; then
    go install golang.org/x/exp/cmd/gorelease@latest
fi

branch=${GITHUB_BASE_REF:-master}
if [ -z "${GITHUB_BASE_SHA:-}"]; then
    git fetch origin $branch:$branch
    sha=$(git merge-base $branch HEAD)
else
    sha="$GITHUB_BASE_SHA"
fi

gorelease -base=$sha > api-report.txt || true

go run ./cmd/parse-api-report/main.go || (cat api-report.txt && exit 1)

rm api-report.txt
