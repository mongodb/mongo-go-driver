#!/usr/bin/env bash
# api-report
# Generates a markdown report of Go Driver API changes
set -eux

cmd=$(command -v gorelease || true)

if [ -z $cmd ]; then
    go install golang.org/x/exp/cmd/gorelease@latest
fi

if [ -n "$GITHUB_BASE_REF" ]; then
    branch=$GITHUB_BASE_REF
    git fetch origin $GITHUB_BASE_REF
    sha=$(git rev-parse origin/$branch)
    git checkout -b my-new-branch
    git clean -dfx
else
    branch=$(git rev-parse --abbrev-ref HEAD)
    sha=$(git --no-pager reflog show $branch | tail -n 1 | awk '{print $1;}')
fi

gorelease -base=$sha > $1 || true
