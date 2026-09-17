#!/usr/bin/env bash
# perf-pr-report
#
# Benchmarks BASE_SHA and HEAD_SHA on this machine and posts the comparison as a
# PR comment.
#
# Both revisions are measured in this one task, with their runs interleaved, and
# compared with benchstat's Mann-Whitney U test. This replaces the previous
# approach of comparing a single run against a baseline held in the Evergreen
# performance monitoring cluster, which was both noisy and dependent on a system
# the Go Driver team does not own.
#
# Environment:
#   BASE_SHA  revision to compare against (required)
#   HEAD_SHA  revision under test (required); no-op when equal to BASE_SHA
#   REPS      repetitions per revision (default 10; benchstat needs >= 6 to be
#             able to report p < 0.05)
#   BENCHTIME_BSON / BENCHTIME_SERVER / BENCHTIME_INSERT
#             fixed iteration counts per benchmark group
#   DRIVERS_TOOLS  when set, the report is posted as a PR comment

set -eux
set -o pipefail

# Skip the report if it isn't a PR run.
if [ "${BASE_SHA:-}" == "${HEAD_SHA:-}" ]; then
    echo "Skipping Perf Report"
    exit 0
fi

REPS=${REPS:-10}

# Fixed iteration counts rather than a wall-clock -benchtime. Several benchmarks
# size their setup off b.N (BenchmarkSingleFindOneByID inserts 10k documents,
# BenchmarkMultiFindMany and benchmarkMultiInsert insert b.N documents), so a
# time-based -benchtime makes b.N vary run to run and leaves ns/op
# non-comparable. The three groups exist because the benchmarks span four orders
# of magnitude, from ~7µs BSON encoding to ~38ms large-document inserts.
BENCHTIME_BSON=${BENCHTIME_BSON:-20000x}
BENCHTIME_SERVER=${BENCHTIME_SERVER:-2000x}
BENCHTIME_INSERT=${BENCHTIME_INSERT:-100x}

BENCH_BSON='BSON'
BENCH_SERVER='SingleRunCommand|SingleFindOneByID|MultiFindMany|MultiInsertSmallDocument'
BENCH_INSERT='SmallDocInsertOne|LargeDocInsertOne|MultiInsertLargeDocument'

BENCHSTAT_VERSION=${BENCHSTAT_VERSION:-v0.0.0-20260908200009-22c9c6c9d4da}

REPO_DIR=$(git rev-parse --show-toplevel)
BENCH_PKG_DIR=internal/cmd/benchmark

WORK_DIR=$(mktemp -d)
BASE_DIR=$WORK_DIR/base

cleanup() {
    git -C "$REPO_DIR" worktree remove --force "$BASE_DIR" >/dev/null 2>&1 || true
    rm -rf "$WORK_DIR"
}
trap cleanup EXIT

# Ensure benchstat is installed and on PATH.
GOBIN_DIR=$(go env GOPATH)/bin
export PATH="$GOBIN_DIR:$PATH"
if ! command -v benchstat >/dev/null; then
    go install "golang.org/x/perf/cmd/benchstat@${BENCHSTAT_VERSION}"
fi

# The Evergreen checkout does not necessarily contain the base commit, so fetch
# it if it is missing.
if ! git -C "$REPO_DIR" cat-file -e "${BASE_SHA}^{commit}" 2>/dev/null; then
    git -C "$REPO_DIR" fetch --no-tags origin "$BASE_SHA"
fi

# Check out the base revision in a worktree. The benchmark module resolves the
# driver through its own "replace ../../../", so building it inside the worktree
# picks up the base revision's driver code automatically.
git -C "$REPO_DIR" worktree add --detach "$BASE_DIR" "$BASE_SHA"

# Compile both benchmark binaries up front: a compile error then fails fast, and
# build time stays out of the measurement loop.
( cd "$REPO_DIR/$BENCH_PKG_DIR" && go test -c -o "$WORK_DIR/head.bench" . )
( cd "$BASE_DIR/$BENCH_PKG_DIR" && go test -c -o "$WORK_DIR/base.bench" . )

# -test.run='^$' keeps TestRunAllBenchmarks from also firing. It needs --fullRun
# to do any work, so it is inert by default, but excluding it is clearer.
run_group() {
    local bin=$1 dir=$2 pattern=$3 benchtime=$4 out=$5

    ( cd "$dir/$BENCH_PKG_DIR" && "$bin" \
        -test.run='^$' \
        -test.bench="$pattern" \
        -test.benchmem \
        -test.benchtime="$benchtime" ) >> "$out"
}

run_all() {
    local bin=$1 dir=$2 out=$3

    run_group "$bin" "$dir" "$BENCH_BSON"   "$BENCHTIME_BSON"   "$out"
    run_group "$bin" "$dir" "$BENCH_SERVER" "$BENCHTIME_SERVER" "$out"
    run_group "$bin" "$dir" "$BENCH_INSERT" "$BENCHTIME_INSERT" "$out"
}

# Warm up. This also provisions testdata/perf in the main checkout, which the
# base worktree then shares rather than downloading a second copy.
run_all "$WORK_DIR/head.bench" "$REPO_DIR" /dev/null

mkdir -p "$BASE_DIR/testdata"
ln -sfn "$REPO_DIR/testdata/perf" "$BASE_DIR/testdata/perf"

run_all "$WORK_DIR/base.bench" "$BASE_DIR" /dev/null

# Interleave base and head within each repetition so that thermal drift or a
# noisy neighbor on the host cannot masquerade as a real difference.
for ((rep = 1; rep <= REPS; rep++)); do
    echo "=== repetition $rep of $REPS ==="
    run_all "$WORK_DIR/base.bench" "$BASE_DIR"  "$WORK_DIR/base.txt"
    run_all "$WORK_DIR/head.bench" "$REPO_DIR"  "$WORK_DIR/head.txt"
done

cd "$REPO_DIR"

benchstat -format=csv "base=$WORK_DIR/base.txt" "head=$WORK_DIR/head.txt" > perf-report.csv
benchstat "base=$WORK_DIR/base.txt" "head=$WORK_DIR/head.txt" || true

# Warn in the comment when the PR changes the benchmarks themselves, since each
# revision is measured with its own harness. This compares the base revision
# against the working tree, which is what was actually compiled for "head" --
# HEAD_SHA is the PR head commit and is not necessarily present in an Evergreen
# checkout.
if git -C "$REPO_DIR" diff --name-only "$BASE_SHA" -- "$BENCH_PKG_DIR" | grep -q .; then
    export PERF_HARNESS_CHANGED=true
fi

go run ./internal/cmd/parse-perf-report
cat perf-report.md

if [ -n "${DRIVERS_TOOLS:-}" ]; then
    target=$DRIVERS_TOOLS/.evergreen/github_app/create_or_modify_comment.sh
    bash "$target" -m "## 🧪 Performance Results" -c "$(pwd)/perf-report.md" \
        -h "$HEAD_SHA" -o "mongodb" -n "mongo-go-driver"
else
    echo "DRIVERS_TOOLS is unset; skipping PR comment"
fi

rm -f perf-report.csv perf-report.md
