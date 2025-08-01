#!/usr/bin/env bash
# perf-pr-comment
# Generates a report of Go Driver perf changes for the current branch.

set -eux

# Generate perf report.
pushd ./internal/cmd/perfcomp >/dev/null || exist
GOWORK=off go run main.go --project="mongo-go-driver" ${VERSION_ID} > ./parseperfcomp/perf-report.txt
popd >/dev/null

if [[ -n "${BASE_SHA+set}" && -n "${HEAD_SHA+set}" && "$BASE_SHA" != "$HEAD_SHA" ]]; then
    # Parse and generate perf comparison comment.
    go run ./internal/cmd/perfcomp/parseperfcomp/main.go
    # Make the PR comment.
    target=$DRIVERS_TOOLS/.evergreen/github_app/create_or_modify_comment.sh
    bash $target -m "## 🧪 Performance Results" -c "$(pwd)/perf-report.md" -h $HEAD_SHA -o "mongodb" -n "mongo-go-driver"
else
    # Skip comment if it isn't a PR run.
    echo "Skipping Perf PR comment"
fi

rm ./internal/cmd/perfcomp/parseperfcomp/perf-report.txt
