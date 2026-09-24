# prose

Prose tests that need drivers test secrets (e.g. the `drivers/csfle` vault).

`TestMain` loads the secrets into the test process before any test runs:

1. The Dockerfile and entrypoint are fetched from the private
   [10gen/go-driver-tools](https://github.com/10gen/go-driver-tools/tree/main/docker)
   repo.
1. The image is built and run with testcontainers-go. The container logs in
   with `aws sso login --no-browser` and runs drivers-evergreen-tools'
   `csfle/setup-secrets.sh`, which writes `secrets-export.sh`.
1. The file is written to `$TMPDIR/mongo-go-driver-prose/secrets-export.sh`
   and loaded into the environment with godotenv.

## Requirements

- The [GitHub CLI](https://cli.github.com/), logged in with `gh auth login`
  as a user with read access to `10gen/go-driver-tools`.
- A running Docker daemon.
- Access to the drivers test-secrets SSO role.

## Running

```
go test -v ./...
```

The login is interactive: the container prints a verification URL and code
to stderr, which a human has to approve in a browser.

Later runs reuse the existing `secrets-export.sh` without logging in again.
The credentials in it expire, so delete the file to force a fresh login:

```
rm "${TMPDIR:-/tmp}/mongo-go-driver-prose/secrets-export.sh"
```
