#!/usr/bin/env bash
# api-report
# Generates a report of Go Driver API changes for the current branch.
set -eux

# Skip the report of it isn't a PR run.
if [ "$BASE_SHA" == "$HEAD_SHA" ]; then
    echo "Skipping API Report"
    exit 0
fi

# Ensure a clean checkout.
git checkout -b test-api-report
git add .
git commit -m "local changes"

# Ensure gorelease is installed.
cmd=$(command -v gorelease || true)
if [ -z $cmd ]; then
    go install golang.org/x/exp/cmd/gorelease@latest
fi

# Generate and parse the report.
gorelease -base=$BASE_SHA > api-report.txt || true
cat api-report.txt
go run ./cmd/parse-api-report/main.go
rm api-report.txt

# Make the PR comment.
target=$DRIVERS_TOOLS/.evergreen/github_app/create_or_modify_comment.sh
bash $target -m "## API Change Report" -c "$(pwd)/api-report.md" -h $HEAD_SHA -o "mongodb" -n "mongo-go-driver"
