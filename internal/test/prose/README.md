# prose

Helpers for prose tests that need credentials from an AWS SSO login.

This is a standalone Go module with its own `go.work`, so its Docker and AWS
dependencies stay out of the driver module's dependency graph. It is not listed
in the root `go.work`.

## Running it with docker

Build once:

```
docker build -t aws-sso-login -f docker/aws-sso-login.Dockerfile docker/
```

Then run it interactively. The entrypoint checks whether the profile already
has an `sso_start_url`; if not, it runs `aws configure sso`, which prompts for
the start URL and lists the accounts and roles your session grants, so nothing
needs to be known up front:

```
mkdir -p out
docker run -it --rm \
    -v "$HOME/.aws:/root/.aws" \
    -v "$PWD/out:/secrets" \
    -e AWS_PROFILE=sso \
    aws-sso-login
cat out/secrets-export.sh
```

Because `~/.aws` is mounted read-write, the profile and the SSO token cache
persist on the host. Later runs take the `aws sso login --no-browser` path,
and reuse the cached token until it expires.

`aws configure sso` needs a TTY, so `-it` is required for that first run. A
non-interactive run against an unconfigured profile exits with code 2 and
points back at the command above rather than hanging on stdin.

## ExportSecrets

`ExportSecrets` builds `docker/aws-sso-login.Dockerfile`, runs it, and returns
the credentials it exports:

```go
secrets := prose.ExportSecrets(t)
accessKeyID := secrets["AWS_ACCESS_KEY_ID"]
```

The container runs `aws sso login --profile "$AWS_PROFILE" --no-browser` and
writes the resulting credentials to `secrets-export.sh` in a directory
bind-mounted from the host — the same `export KEY=VALUE` format that
drivers-evergreen-tools uses. The host directory is `t.TempDir()`, so the
credentials do not outlive the test.

The host's `~/.aws` is mounted read-write at `/root/.aws` so the CLI can
resolve the profile and so the SSO token cache it writes persists. Override the
directory with `WithAWSDir`.

The login is **interactive**: `--no-browser` prints a verification URL and code
that a human has to approve. Container output is streamed to the test log, so
run with `-v` to see them:

```
AWS_PROFILE=my-profile go test -v -run TestThatNeedsSecrets ./...
```

Before exiting, the container verifies the credentials it just wrote by
running `aws sts get-caller-identity` with them loaded as environment
variables (not via `--profile`, which would only re-check the SSO session).
A failure there makes the container exit non-zero and `ExportSecrets` fail.
Set `VERIFY_CREDENTIALS=0` in the image to skip it.

Once a login succeeds, the cached SSO token is reused until it expires, so
later runs complete without a prompt. Nothing about the first login is
automatic, and the returned credentials live only for the duration of the test.

`ExportSecrets` skips the test when `AWS_PROFILE` is unset and no profile was
passed via `WithProfile`.

## Requirements

- A running Docker daemon (used via testcontainers-go).
- `AWS_PROFILE` naming a profile configured for SSO, for real logins.

## Tests

`TestExportSecrets` runs against `docker/aws-sso-login-fixture.Dockerfile`,
which pairs the real `entrypoint.sh` with a stub `aws` binary, and points
`WithAWSDir` at an empty temp directory so it never mounts real credentials.
The suite therefore needs Docker but neither real AWS credentials nor an
interactive SSO prompt:

```
go test ./...
```
