#!/bin/bash

sha=$1
base=v1
target=master
dirname="/tmp/go-driver-$(openssl rand -hex 12)"
mkdir -p $dirname
git clone git@github.com:mongodb/mongo-go-driver.git $dirname
cd $dirname
branch="cherry-pick-$sha"
git fetch origin $base
git fetch origin $target
git checkout -b $branch origin/$target
git cherry-pick -x $sha

old_title=$(git --no-pager log  -1 --pretty=%B | head -n 1)
ticket=$(echo $old_title | sed -r 's/([A-Z]+-[0-9]+).*/\1/')
echo "ticket $ticket"
text=$(echo $old_title | sed -r 's/([A-Z]+-[0-9]+) (.*) \(#[0-9]*\)/\2/')
echo "text $text"
pr_number=$(echo $old_title| sed -r 's/.*(#[0-9]*)\)/\1/')
echo "pr number: $pr_number"
title="$ticket [$target] $text"
body="Cherry-pick of $pr_number from $base to $target"

echo
echo "Creating PR..."
echo "Title: $title"
echo "Body: $body"
echo

read -p 'Push changes? (Y/n) ' choice
if [[ "$choice" == "Y" || "$choice" == "y" || -z "$choice" ]]; then
    gh pr create --title "$title" --base $target --body "$body"
fi
