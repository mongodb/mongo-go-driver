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
text=$(echo $old_title | sed -r 's/([A-Z]+-[0-9]+) (.*) \(#[0-9]*\)/\2/')
pr_number=$(echo $old_title| sed -r 's/.*(#[0-9]*)\)/\1/')
user=$(git config github.user)

if [ -n "$user" ]; then
    head="$user:$branch"
else
    head="$origin:$branch"
fi

title="$ticket [$target] $text"
body="Cherry-pick of $pr_number from $base to $target"

echo
echo "Creating PR..."
echo "Title: $title"
echo "Body: $body"
echo "Base: $base"
echo "Head: $head"
echo

read -p 'Push changes? (Y/n) ' choice
if [[ "$choice" == "Y" || "$choice" == "y" || -z "$choice" ]]; then
    gh pr create --title "$title" --base $target --head $head --body "$body"
fi
