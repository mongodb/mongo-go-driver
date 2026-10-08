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

### crypt_shared

The image also downloads the `crypt_shared` library (with
drivers-evergreen-tools' `mongodl.py`) and sets `CRYPT_SHARED_LIB_PATH`.
Automatic encryption needs it for query analysis; without it the driver tries
to spawn `mongocryptd`, which is not installed.

### Wrapped prose tests

The CSE tests in `internal/integration` (the files with the `cse` build tag)
still run there in CI. `TestCSEIntegration` runs them in the CSE container instead.
It finds them with `go test -list`, run in the container because the `cse` tag
needs libmongocrypt to compile: the tests listed with `-tags cse`, minus the
ones listed without it. Each one runs as a subtest, so there is no list of test
names to maintain:

```
go test -v . -cse -run TestCSEIntegration
go test -v . -cse -run 'TestCSEIntegration/TestClientSideEncryptionProse_27$'
```

### KMS mock servers

Tests that use KMS providers, such as
`TestClientSideEncryptionProse_11_kms_tls_options_tests`, need the KMS mock
servers from drivers-evergreen-tools. Without them the tests skip inside the
container, but the wrapper still reports a pass. Start them on the host with:

```
task setup-encryption
```

This runs `etc/setup-encryption.sh`, which starts the mocks with the EC test
certificates in `testdata/kmip-certs`. It includes an AWS SSO login.

When `-cse` is passed and every mock port (9000-9003 and 5698) accepts a
connection on `localhost`:

- `KMS_MOCK_SERVERS_RUNNING=true` is set in the container's environment.
- Each port is relayed inside the container, with `socat`, from
  `127.0.0.1:<port>` to `host.docker.internal:<port>`. The tests dial
  `127.0.0.1`, which in the container is the container itself.
- `CSFLE_TLS_CA_FILE` and `CSFLE_TLS_CLIENT_CERT_FILE` point at the EC
  certificates in `testdata/kmip-certs`, overriding the RSA paths in
  `secrets-export.sh`. The KMIP mock and Go's default TLS settings share no
  cipher suite when the mock uses an RSA certificate.

If any mock is not running, nothing is relayed and these tests skip.

Mock servers that listen only on `127.0.0.1` on the host, such as 9003 and
5698 in some setups, may not be reachable from the container on Linux. Start
them on `0.0.0.0` there.
