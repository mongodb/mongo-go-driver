# prose

Prose tests that need drivers test secrets (e.g. the `drivers/csfle` vault).

`TestMain` loads the secrets into the test process before any test runs:

1. The Dockerfile and entrypoint are fetched from the private
   [10gen/go-driver-tools](https://github.com/10gen/go-driver-tools/tree/main/docker)
   repo.
1. The image is built and run with testcontainers-go. The container logs in
   with `aws sso login --no-browser` and runs drivers-evergreen-tools'
   `csfle/setup-secrets.sh`, which writes `secrets-export.sh`.
1. The file is written to `secrets-export.sh` in the repository root
   and loaded into the environment with godotenv.

## Requirements

- The [GitHub CLI](https://cli.github.com/), logged in with `gh auth login`
  as a user with read access to `10gen/go-driver-tools`.
- A running Docker daemon.
- Access to the drivers-test-secrets SSO role.

## Running

Secrets are only loaded when a flag asks for them; without one, tests that
need them are skipped:

```
cd internal/test/prose
go test -v . -load-secrets
go test -v . -cse
```

The login is interactive: the container prints a verification URL and code
to stderr, which a human has to approve in a browser.

Runs within 50 minutes of a login reuse the existing `secrets-export.sh`.
After that the temporary tokens in it are close to expiring, so the next run
logs in again. To force a fresh login sooner, delete the file:

```
rm secrets-export.sh  # from the repository root
```

## CSE

`-cse` also starts a container built from `cse.Dockerfile`, which has
libmongocrypt installed, and `TestCSE` runs the `cse`-tagged tests inside it:

```
cd internal/test/prose
go test -v . -cse -run TestCSE
```

The container is named `mongo-go-driver-cse` and is reused across runs (Ryuk
is disabled so it outlives the test process). The repository root is
bind-mounted at `/mongo-go-driver`, so driver changes do not need a rebuild.
To rebuild the image, for example after changing `cse.Dockerfile`, remove the
container:

```
docker rm -f mongo-go-driver-cse
```

Commands run in the container get every secret from `secrets-export.sh` and
`MONGODB_URI` as environment variables. These are set on each exec, not when
the container is created, so a reused container always gets fresh secrets.
`MONGODB_URI` defaults to `mongodb://localhost:27017`, and loopback hosts are
rewritten to `host.docker.internal` so the container reaches the host's
`mongod`. On Linux, `mongod` must listen on an address the Docker bridge can
reach (for example `--bind_ip_all`), not only `127.0.0.1`.

A single loopback host also gets `directConnection=true`: replica set members
advertise themselves as `localhost`, which the container cannot reach, so the
driver must not follow the member list. `directConnection` is invalid with
multiple hosts, so point it at a single member, such as the primary.

### Wrapped prose tests

Some CSE prose tests still live in `internal/integration` and run there in CI.
Wrappers in this package run them in the CSE container instead, for example:

```
go test -v . -cse -run TestClientSideEncryptionProse_27
```
