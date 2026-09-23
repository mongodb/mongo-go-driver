# prose

Helpers for prose tests that need credentials from an AWS SSO login.

This is a standalone Go module with its own `go.work`, so its Docker and AWS
dependencies stay out of the driver module's dependency graph. It is not listed
in the root `go.work`.

## Running it with docker

There is no setup step. The settings `aws configure sso` would otherwise
prompt for — start URL, SSO region, account and role — are hardcoded as
defaults in `docker/entrypoint.sh`, which writes the profile inside the
container before logging in. Nothing on the host has to be configured, and the
host's `~/.aws/config` is never read or written.

Build once:

```
docker build -t aws-sso-login -f docker/aws-sso-login.Dockerfile docker/
```

Then run it:

```
mkdir -p out
docker run --rm \
    -v "$HOME/.aws/sso/cache:/root/.aws/sso/cache" \
    -v "$PWD/out:/secrets" \
    aws-sso-login
cat out/secrets-export.sh
```

`--no-browser` prints a verification URL and a code; open the URL and approve
the login. The SSO token cache is shared with the host, so once a login
succeeds — here or with the host's own AWS CLI — later runs reuse it until the
token expires.

To log in somewhere else, override any of `AWS_PROFILE`, `SSO_START_URL`,
`SSO_REGION`, `SSO_ACCOUNT_ID`, `SSO_ROLE_NAME` or `AWS_REGION` with `-e`.

`secrets-export.sh` is the `export KEY=VALUE` format drivers-evergreen-tools
uses, so it can be sourced directly:

```
set -a && . ./out/secrets-export.sh && set +a
```

## ExportSecrets

`ExportSecrets` builds `docker/aws-sso-login.Dockerfile`, runs it, and returns
the directory holding the credentials it exported:

```go
dir := prose.ExportSecrets(t)
// dir/secrets-export.sh holds "export KEY=VALUE" lines, e.g. to pass to a
// subprocess:
cmd.Env = append(os.Environ(), "BASH_ENV="+filepath.Join(dir, "secrets-export.sh"))
```

The directory is bind-mounted into the container, which writes
`secrets-export.sh` into it. It is a `t.TempDir()`, so the credentials do not
outlive the test.

The host's `~/.aws/sso/cache` is mounted read-write so the SSO token persists
between runs; override it with `WithSSOCacheDir`. Override the profile with
`WithProfile` or `AWS_PROFILE`.

The login is **interactive** the first time: `--no-browser` prints a
verification URL and code that a human has to approve. Container output is
streamed to the test log, so run with `-v` to see them:

```
go test -v -run TestThatNeedsSecrets ./...
```

Before exiting, the container verifies the credentials it just wrote by
running `aws sts get-caller-identity` with them loaded as environment
variables (not via `--profile`, which would only re-check the SSO session).
A failure there makes the container exit non-zero and `ExportSecrets` fail.
Set `VERIFY_CREDENTIALS=0` in the image to skip it.

## Requirements

- A running Docker daemon (used via testcontainers-go).
- Membership in the SSO account the hardcoded profile names.

## Tests

`TestExportSecrets` runs against `docker/aws-sso-login-fixture.Dockerfile`,
which pairs the real `entrypoint.sh` with a stub `aws` binary, and points
`WithSSOCacheDir` at an empty temp directory so it never mounts real
credentials. The suite therefore needs Docker but neither real AWS credentials
nor an interactive SSO prompt:

```
go test ./...
```

`TestRealLogin` is the end-to-end check against a real SSO login. It is the
only test that prompts.
