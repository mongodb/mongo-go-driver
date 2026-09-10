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

# ext/awsauth declares the minimum aws-sdk-go-v2 version that is layout-compatible with
# its AWSCredentials type (see ext/awsauth/doc.go). That floor is easy to raise by
# accident: "go work sync" rewrites each workspace member's go.mod with the
# workspace-resolved maximum, and Dependabot's workspace update procedure runs it on
# every gomod update. Assert the floor so an accidental bump fails loudly.
awsauth_dep=github.com/aws/aws-sdk-go-v2
awsauth_floor=v1.28.0
echo "Checking ext/awsauth aws-sdk-go-v2 floor..."
awsauth_got=$(cd ext/awsauth && GOWORK=off go list -m -f '{{.Version}}' "$awsauth_dep")
if [ "$awsauth_got" != "$awsauth_floor" ]; then
  echo "ext/awsauth requires $awsauth_dep $awsauth_got, want $awsauth_floor."
  echo "If this was an unintended bump (e.g. from \"go work sync\"), restore the floor."
  echo "If the floor is being raised on purpose, update ext/awsauth/doc.go and the"
  echo "awsauth_floor value in this script."
  exit_code=1
fi
echo "Checking ext/awsauth aws-sdk-go-v2 floor... done"

exit $exit_code
