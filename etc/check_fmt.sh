#!/usr/bin/env bash
# check_fmt gopackages...
# Runs gofmt on given packages and checks that *_example_test.go files have wrapped lines.

gofmt_out="$(gofmt -l -s "$@")"

if [[ $gofmt_out ]]; then
  echo "gofmt check failed for:";
  sed -e 's/^/ - /' <<< "$gofmt_out";
  exit 1;
fi

# Use the "github.com/walle/lll" tool to check that all lines in *_example_test.go files are
# wrapped at 80 characters to keep them readable when rendered on https://pkg.go.dev.
# Ignore long lines that are comments containing URI-like strings.
# E.g ignored lines:
#     // "mongodb://ldap-user:ldap-pwd@localhost:27017/?authMechanism=PLAIN"
#     // (https://www.mongodb.com/docs/manual/core/authentication-mechanisms-enterprise/#security-auth-ldap).
lll_out="$(find "$@" -type f -name "*_examples_test.go" | lll -w 4 -l 80 -e '^\s*\/\/.+:\/\/' --files)"

if [[ $lll_out ]]; then
  echo "lll check failed for:";
  sed -e 's/^/ - /' <<< "$lll_out";
  exit 1;
fi
