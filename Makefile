TEST_TIMEOUT = 1800

### Utility targets. ###
.PHONY: default
default: build check-license check-fmt check-modules lint test-short

.PHONY: add-license
add-license:
	etc/check_license.sh -a

.PHONY: check-license
check-license:
	etc/check_license.sh

.PHONY: build
build: cross-compile build-tests build-compile-check
	go build ./...
	go build $(BUILD_TAGS) ./...

# Use ^$ to match no tests so that no tests are actually run but all tests are
# compiled. Run with -short to ensure none of the TestMain functions try to
# connect to a server.
.PHONY: build-tests
build-tests:
	go test -short $(BUILD_TAGS) -run ^$$ ./...

.PHONY: build-compile-check
build-compile-check:
	etc/compile_check.sh

# Cross-compiling on Linux for architectures 386, arm, arm64, amd64, ppc64le, and s390x.
# Omit any build tags because we don't expect our build environment to support compiling the C
# libraries for other architectures.
.PHONY: cross-compile
cross-compile:
	GOOS=linux GOARCH=386 go build ./...
	GOOS=linux GOARCH=arm go build ./...
	GOOS=linux GOARCH=arm64 go build ./...
	GOOS=linux GOARCH=amd64 go build ./...
	GOOS=linux GOARCH=ppc64le go build ./...
	GOOS=linux GOARCH=s390x go build ./...

.PHONY: install-lll
install-lll:
	go install github.com/walle/lll/...@latest

.PHONY: check-fmt
check-fmt: install-lll
	etc/check_fmt.sh

# check-modules runs "go mod tidy" then "go mod vendor" and exits with a non-zero exit code if there
# are any module or vendored modules changes. The intent is to confirm two properties:
#
# 1. Exactly the required modules are declared as dependencies. We should always be able to run
# "go mod tidy" and expect that no unrelated changes are made to the "go.mod" file.
#
# 2. All required modules are copied into the vendor/ directory and are an exact copy of the
# original module source code (i.e. the vendored modules are not modified from their original code).
.PHONY: check-modules
check-modules:
	go mod tidy -v
	go mod vendor
	git diff --exit-code go.mod go.sum ./vendor

.PHONY: doc
doc:
	godoc -http=:6060 -index

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: install-golangci-lint
install-golangci-lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.52.2

# Lint with various GOOS and GOARCH targets to catch static analysis failures that may only affect
# specific operating systems or architectures. For example, staticcheck will only check for 64-bit
# alignment of atomically accessed variables on 32-bit architectures (see
# https://staticcheck.io/docs/checks#SA1027)
.PHONY: lint
lint: install-golangci-lint
	GOOS=linux GOARCH=386 golangci-lint run --config .golangci.yml ./...
	GOOS=linux GOARCH=arm golangci-lint run --config .golangci.yml ./...
	GOOS=linux GOARCH=arm64 golangci-lint run --config .golangci.yml ./...
	GOOS=linux GOARCH=amd64 golangci-lint run --config .golangci.yml ./...
	GOOS=linux GOARCH=ppc64le golangci-lint run --config .golangci.yml ./...
	GOOS=linux GOARCH=s390x golangci-lint run --config .golangci.yml ./...

.PHONY: update-notices
update-notices:
	etc/generate_notices.pl > THIRD-PARTY-NOTICES

### Local testing targets. ###
.PHONY: test
test:
	go test $(BUILD_TAGS) -timeout $(TEST_TIMEOUT)s -p 1 ./...

.PHONY: test-cover
test-cover:
	go test $(BUILD_TAGS) -timeout $(TEST_TIMEOUT)s -cover $(COVER_ARGS) -p 1 ./...

.PHONY: test-race
test-race:
	go test $(BUILD_TAGS) -timeout $(TEST_TIMEOUT)s -race -p 1 ./...

.PHONY: test-short
test-short:
	go test $(BUILD_TAGS) -timeout 60s -short ./...

### Local FaaS targets. ###
.PHONY: build-faas-awslambda
build-faas-awslambda:
	$(if $(MONGODB_URI),,$(error MONGODB_URI is not set))
	$(MAKE) -C internal/test/faas/awslambda

### Evergreen specific targets. ###
.PHONY: build-aws-ecs-test
build-aws-ecs-test:
	go build $(BUILD_TAGS) ./cmd/testaws/main.go

.PHONY: evg-test
evg-test:
	go test -exec "env PKG_CONFIG_PATH=${PKG_CONFIG_PATH} LD_LIBRARY_PATH=${LD_LIBRARY_PATH} DYLD_LIBRARY_PATH=$(MACOS_LIBRARY_PATH)}" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s -p 1 ./... >> test.suite

.PHONY: evg-test-atlas-data-lake
evg-test-atlas-data-lake:
	ATLAS_DATA_LAKE_INTEGRATION_TEST=true go test -v ./mongo/integration -run TestUnifiedSpecs/atlas-data-lake-testing >> spec_test.suite
	ATLAS_DATA_LAKE_INTEGRATION_TEST=true go test -v ./mongo/integration -run TestAtlasDataLake >> spec_test.suite

.PHONY: evg-test-enterprise-auth
evg-test-enterprise-auth:
	go run -tags gssapi ./cmd/testentauth/main.go

.PHONY: evg-test-kmip
evg-test-kmip:
	go test -exec "env PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) LD_LIBRARY_PATH=$(LD_LIBRARY_PATH) DYLD_LIBRARY_PATH=$(MACOS_LIBRARY_PATH)" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s ./mongo/integration -run TestClientSideEncryptionSpec/kmipKMS >> test.suite
	go test -exec "env PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) LD_LIBRARY_PATH=$(LD_LIBRARY_PATH) DYLD_LIBRARY_PATH=$(MACOS_LIBRARY_PATH)" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s ./mongo/integration -run TestClientSideEncryptionProse/data_key_and_double_encryption >> test.suite
	go test -exec "env PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) LD_LIBRARY_PATH=$(LD_LIBRARY_PATH) DYLD_LIBRARY_PATH=$(MACOS_LIBRARY_PATH)" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s ./mongo/integration -run TestClientSideEncryptionProse/corpus >> test.suite
	go test -exec "env PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) LD_LIBRARY_PATH=$(LD_LIBRARY_PATH) DYLD_LIBRARY_PATH=$(MACOS_LIBRARY_PATH)" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s ./mongo/integration -run TestClientSideEncryptionProse/custom_endpoint >> test.suite
	go test -exec "env PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) LD_LIBRARY_PATH=$(LD_LIBRARY_PATH) DYLD_LIBRARY_PATH=$(MACOS_LIBRARY_PATH)" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s ./mongo/integration -run TestClientSideEncryptionProse/kms_tls_options_test >> test.suite

.PHONY: evg-test-kms
evg-test-kms:
	go test -exec "env PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) LD_LIBRARY_PATH=$(LD_LIBRARY_PATH) DYLD_LIBRARY_PATH=$(MACOS_LIBRARY_PATH)" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s ./mongo/integration -run TestClientSideEncryptionProse/kms_tls_tests >> test.suite

.PHONY: evg-test-load-balancers
evg-test-load-balancers:
	# Load balancer should be tested with all unified tests as well as tests in the following
	# components: retryable reads, retryable writes, change streams, initial DNS seedlist discovery.
	go test $(BUILD_TAGS) ./mongo/integration -run TestUnifiedSpecs/retryable-reads -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestRetryableWritesSpec -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestChangeStreamSpec -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestInitialDNSSeedlistDiscoverySpec/load_balanced -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestLoadBalancerSupport -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestLoadBalancedConnectionHandshake -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration/unified -run TestUnifiedSpec -v -timeout $(TEST_TIMEOUT)s >> test.suite

.PHONY: evg-test-search-index
evg-test-search-index:
	go test ./mongo/integration -run TestSearchIndexProse -v -timeout $(TEST_TIMEOUT)s >> test.suite

.PHONY: evg-test-ocsp
evg-test-ocsp:
	go test -v ./mongo -run TestOCSP $(OCSP_TLS_SHOULD_SUCCEED) >> test.suite

.PHONY: evg-test-serverless
evg-test-serverless:
	# Serverless should be tested with all unified tests as well as tests in the following components: CRUD, load balancer,
	# retryable reads, retryable writes, sessions, transactions and cursor behavior.
	go test $(BUILD_TAGS) ./mongo/integration -run TestCrudSpec -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestWriteErrorsWithLabels -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestWriteErrorsDetails -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestHintErrors -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestWriteConcernError -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestErrorsCodeNamePropagated -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestLoadBalancerSupport -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestUnifiedSpecs/retryable-reads -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestRetryableReadsProse -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestRetryableWritesSpec -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestRetryableWritesProse -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestUnifiedSpecs/sessions -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestSessionsProse -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestUnifiedSpecs/transactions/legacy -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestConvenientTransactions -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestCursor -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration/unified -run TestUnifiedSpec -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test -exec "env PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) LD_LIBRARY_PATH=$(LD_LIBRARY_PATH) DYLD_LIBRARY_PATH=$(MACOS_LIBRARY_PATH)" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s ./mongo/integration -run TestClientSideEncryptionSpec >> test.suite
	go test -exec "env PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) LD_LIBRARY_PATH=$(LD_LIBRARY_PATH) DYLD_LIBRARY_PATH=$(MACOS_LIBRARY_PATH)" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s ./mongo/integration -run TestClientSideEncryptionProse >> test.suite

.PHONY: evg-test-versioned-api
evg-test-versioned-api:
	# Versioned API related tests are in the mongo, integration and unified packages.
	go test -exec "env PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) LD_LIBRARY_PATH=$(LD_LIBRARY_PATH) DYLD_LIBRARY_PATH=$(MACOS_LIBRARY_PATH)" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s ./mongo >> test.suite
	go test -exec "env PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) LD_LIBRARY_PATH=$(LD_LIBRARY_PATH) DYLD_LIBRARY_PATH=$(MACOS_LIBRARY_PATH)" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s ./mongo/integration >> test.suite
	go test -exec "env PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) LD_LIBRARY_PATH=$(LD_LIBRARY_PATH) DYLD_LIBRARY_PATH=$(MACOS_LIBRARY_PATH)" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s ./mongo/integration/unified >> test.suite

.PHONY: build-kms-test
build-kms-test:
	go build $(BUILD_TAGS) ./cmd/testkms

### Benchmark specific targets and support. ###
.PHONY: benchmark
benchmark:perf
	go test $(BUILD_TAGS) -benchmem -bench=. ./benchmark | test benchmark.suite

.PHONY: driver-benchmark
driver-benchmark:perf
	@go run cmd/godriver-benchmark/main.go | tee perf.suite

perf:driver-test-data.tar.gz
	tar -zxf $< $(if $(eq $(UNAME_S),Darwin),-s , --transform=s)/testdata/perf/
	@touch $@

driver-test-data.tar.gz:
	curl --retry 5 "https://s3.amazonaws.com/boxes.10gen.com/build/driver-test-data.tar.gz" -o driver-test-data.tar.gz --silent --max-time 120
