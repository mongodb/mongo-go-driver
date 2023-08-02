#!/usr/bin/env bash
# api-report
# Generates a markdown report of Go Driver API changes
set -eux

cmd=$(command -v gorelease || true)

if [ -z $cmd ]; then
    export GOPATH=$(go env GOPATH)
    go install golang.org/x/exp/cmd/gorelease@latest
    cmd="$(go env GOPATH)/gorelease"
fi

if [ -n "$GITHUB_BASE_REF" ]; then
    branch=$GITHUB_BASE_REF
    git fetch origin $branch
    sha=$(git rev-parse origin/$branch)
else
    branch=$(git rev-parse --abbrev-ref HEAD)
    sha=$(git --no-pager reflog show $branch | tail -n 1 | awk '{print $1;}')
fi

output=$($cmd -base=$sha || true)

echo "hello"
echo $output