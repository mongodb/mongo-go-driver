#!/usr/bin/env bash
# check-modules runs "go mod tidy" on each module and exits with a non-zero exit code if there
# are any module changes. The intent is to confirm that exactly the required
# modules are declared as dependencies. We should always be able to run "go mod
# tidy" and expect that no unrelated changes are made to the "go.mod" file.
set -eu

mods=$(find . -name go.mod)
exit_code=0
for mod in $mods; do
  pushd "$(dirname $mod)" > /dev/null
  echo "Checking $mod..."
  go mod tidy -v
  git diff --exit-code go.mod go.sum || {
    exit_code=$?
  }
  echo "Checking $mod... done"
  popd > /dev/null
done
exit $exit_code
