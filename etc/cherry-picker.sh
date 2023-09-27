#!/bin/bash
set -e

sha=$1
base=v1
target=master
dirname=$(mktemp -d)
user=$(git config github.user)

if [ -z "$user" ]; then
    echo "Please set GitHub User"
    echo "git config --global github.user <github_handle>"
    exit 1
fi

mkdir -p $dirname
if [ -z "$AUTH_TOKEN" ]; then
    git clone git@github.com:mongodb/mongo-go-driver.git $dirname
else
    echo "$AUTH_TOKEN" > mytoken.txt
    gh auth login --with-token < mytoken.txt
    git clone https://github.com/mongodb/mongo-go-driver.git $dirname
fi

cd $dirname
if [ -z "$AUTH_TOKEN" ]; then
    git remote add $user git@github.com:$user/mongo-go-driver.git
else
    git remote add $user https://$user:${AUTH_TOKEN}@github.com/$user/mongo-go-driver.git
fi

gh repo set-default mongodb/mongo-go-driver
branch="cherry-pick-$sha"
head="$user:$branch"
git fetch origin $base
git fetch origin $target
git checkout -b $branch origin/$target
git cherry-pick -x $sha || true

files=$(git ls-files -m)
if [ -n "${files}" ]; then
    EDITOR=${EDITOR:-$(git config core.editor)}
    EDITOR=${EDITOR:-vim}
    for fname in $files; do
        echo "Fixing $fname..."
        $EDITOR $fname
        git add $fname
    done
    echo "Finishing cherry pick."
    git cherry-pick --continue
fi

old_title=$(git --no-pager log  -1 --pretty=%B | head -n 1)
ticket=$(echo $old_title | sed -r 's/([A-Z]+-[0-9]+).*/\1/')
text=$(echo $old_title | sed -r 's/([A-Z]+-[0-9]+) (.*) \(#[0-9]*\)/\2/')
pr_number=$(echo $old_title| sed -r 's/.*(#[0-9]*)\)/\1/')

title="$ticket [$target] $text"
body="Cherry-pick of $pr_number from $base to $target"

echo
echo "Creating PR..."
echo "Title: $title"
echo "Body: $body"
echo "Base: $target"
echo "Head: $head"
echo

if [ -n "$GITHUB_ACTOR" ]; then
    choice=Y
else
    read -p 'Push changes? (Y/n) ' choice
fi

if [[ "$choice" == "Y" || "$choice" == "y" || -z "$choice" ]]; then
    if [ -n "$user" ]; then
        git push $user
    fi
    gh pr create --title "$title" --base $target --head $head --body "$body"
fi
