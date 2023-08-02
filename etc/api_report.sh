#!/usr/bin/env bash
# api-report
# generates a markdown report of API changes
set -eux

go install golang.org/x/exp/cmd/gorelease@latest
branch=$(git rev-parse --abbrev-ref HEAD)
sha=$(git --no-pager reflog show $branch | tail -n 1 | awk '{print $1;}')
output=$(gorelease -base=$sha)
echo "hi"
echo $output