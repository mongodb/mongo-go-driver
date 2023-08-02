#!/usr/bin/env bash
# api-report
# Generates a markdown report of Go Driver API changes
set -eux

cmd=$(command -v gorelease || true)

if [ -z $cmd ]; then
    go install golang.org/x/exp/cmd/gorelease@latest
    cmd="$(go env GOPATH)/gorelease"
fi
branch=$(git rev-parse --abbrev-ref HEAD)
sha=$(git --no-pager reflog show $branch | tail -n 1 | awk '{print $1;}')
output=$($cmd -base=$sha || true)

echo $output