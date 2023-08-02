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
    touch test.txt
    # https://github.community/t/github-actions-bot-email-address/17204/6
    git_user_name="41898282+github-actions[bot]"
    git config user.email "${git_user_name}@users.noreply.github.com"
    git config user.name "{$git_user_name}"
    git add test.txt
    git commit -a -m "foo"
else
    branch=$(git rev-parse --abbrev-ref HEAD)
    sha=$(git --no-pager reflog show $branch | tail -n 1 | awk '{print $1;}')
fi

git status --porcelain
git --no-pager show
gorelease -base=$sha > $1
