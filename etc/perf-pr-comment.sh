#!/usr/bin/env bash
# perf-pr-comment
# Generates a report of Go Driver perf changes for the current branch.

set -eux

pushd $DRIVERS_TOOLS/.evergreen >/dev/null
sh run-perf-comp.sh

./perfcomp/bin/perfcomp compare --project="mongo-go-driver" ${VERSION_ID}

    if [[ -n "${BASE_SHA+set}" && -n "${HEAD_SHA+set}" && "$BASE_SHA" != "$HEAD_SHA" ]]; then
    # Parse and generate perf comparison comment.
    GOWORK=off ./perfcomp/bin/perfcomp mdreport
    # Make the PR comment.
    target=github_app/create_or_modify_comment.sh
    bash $target -m "## 🧪 Performance Results" -c "$(pwd)/perf-report.md" -h $HEAD_SHA -o "mongodb" -n "mongo-go-driver"
    rm ./perf-report.txt
    rm ./perf-report.md
else
    # Skip comment if it isn't a PR run.
    echo "Skipping Perf PR comment"
fi

popd >/dev/null
