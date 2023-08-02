#!/usr/bin/env bash
# api-report
# Generates a markdown report of Go Driver API changes
set -eux

cmd=$(command -v gorelease || true)

if [ -z $cmd ]; then
    go install golang.org/x/exp/cmd/gorelease@latest
fi

branch=${GITHUB_BASE_REF:-master}
sha=$(git merge-base $branch HEAD)

gorelease -base=$sha > $1 || true
