ATLAS_URIS = "$(ATLAS_FREE)" "$(ATLAS_REPLSET)" "$(ATLAS_SHARD)" "$(ATLAS_TLS11)" "$(ATLAS_TLS12)" "$(ATLAS_FREE_SRV)" "$(ATLAS_REPLSET_SRV)" "$(ATLAS_SHARD_SRV)" "$(ATLAS_TLS11_SRV)" "$(ATLAS_TLS12_SRV)" "$(ATLAS_SERVERLESS)" "$(ATLAS_SERVERLESS_SRV)"
GODISTS=linux/amd64 linux/386 linux/arm64 linux/arm linux/s390x
TEST_TIMEOUT = 1800

### Utility targets. ###
.PHONY: default
default: add-license build build-examples check-env check-fmt check-modules lint test-short

.PHONY: add-license
add-license:
	# Find all .go files not in the vendor directory and try to write a license notice.
	find . -path ./vendor -prune -o -type f -name "*.go" -print | xargs ./etc/add_license.sh
	# Check for any changes made with -G. to ignore permissions changes. Exit with a non-zero
	# exit code if there is a diff.
	git diff -G. --quiet

.PHONY: build
build:
	go build $(BUILD_TAGS) ./...

.PHONY: build-examples
build-examples:
	go build $(BUILD_TAGS) ./examples/...

.PHONY: build-no-tags
build-no-tags:
	go build ./...

.PHONY: build-tests
build-tests:
	# Use ^$ to match no tests so that no tests are actually run but all tests are
	# compiled. Run with -short to ensure none of the TestMain functions try to
	# connect to a server.
	go test -short $(BUILD_TAGS) -run ^$$ ./...

.PHONY: check-fmt
check-fmt:
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

.PHONY: lint
lint:
	for dist in $(GODISTS); do \
		goos=$$(echo $$dist | cut -d/ -f 1) ; \
		goarch=$$(echo $$dist | cut -d/ -f 2) ; \
		command="GOOS=$$goos GOARCH=$$goarch golangci-lint run --config .golangci.yml ./..." ; \
		echo $$command ; \
		eval $$command ; \
	done

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
	go test $(BUILD_TAGS) -timeout 60s -short -p 1 ./...

### Evergreen specific targets. ###
.PHONY: build-aws-ecs-test
build-aws-ecs-test:
	go build $(BUILD_TAGS) ./cmd/testaws/main.go

.PHONY: evg-test
evg-test:
	go test -exec "env PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) LD_LIBRARY_PATH=$(LD_LIBRARY_PATH)" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s -p 1 ./... >> test.suite

.PHONY: evg-test-atlas
evg-test-atlas:
	go run ./cmd/testatlas/main.go $(ATLAS_URIS)

.PHONY: evg-test-atlas-data-lake
evg-test-atlas-data-lake:
	ATLAS_DATA_LAKE_INTEGRATION_TEST=true go test -v ./mongo/integration -run TestUnifiedSpecs/atlas-data-lake-testing >> spec_test.suite
	ATLAS_DATA_LAKE_INTEGRATION_TEST=true go test -v ./mongo/integration -run TestAtlasDataLake >> spec_test.suite

.PHONY: evg-test-enterprise-auth
evg-test-enterprise-auth:
	go run -tags gssapi ./cmd/testentauth/main.go

.PHONY: evg-test-kmip
evg-test-kmip:
	go test -exec "env PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) LD_LIBRARY_PATH=$(LD_LIBRARY_PATH)" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s ./mongo/integration -run TestClientSideEncryptionSpec/kmipKMS >> test.suite
	go test -exec "env PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) LD_LIBRARY_PATH=$(LD_LIBRARY_PATH)" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s ./mongo/integration -run TestClientSideEncryptionProse/data_key_and_double_encryption >> test.suite
	go test -exec "env PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) LD_LIBRARY_PATH=$(LD_LIBRARY_PATH)" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s ./mongo/integration -run TestClientSideEncryptionProse/corpus >> test.suite
	go test -exec "env PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) LD_LIBRARY_PATH=$(LD_LIBRARY_PATH)" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s ./mongo/integration -run TestClientSideEncryptionProse/custom_endpoint >> test.suite
	go test -exec "env PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) LD_LIBRARY_PATH=$(LD_LIBRARY_PATH)" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s ./mongo/integration -run TestClientSideEncryptionProse/kms_tls_options_test >> test.suite

.PHONY: evg-test-kms
evg-test-kms:
	go test -exec "env PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) LD_LIBRARY_PATH=$(LD_LIBRARY_PATH)" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s ./mongo/integration -run TestClientSideEncryptionProse/kms_tls_tests >> test.suite

.PHONY: evg-test-load-balancers
evg-test-load-balancers:
	# Load balancer should be tested with all unified tests as well as tests in the following
	# components: retryable reads, retryable writes, change streams, initial DNS seedlist discovery.
	go test $(BUILD_TAGS) ./mongo/integration -run TestUnifiedSpecs/retryable-reads -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestRetryableWritesSpec -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestChangeStreamSpec -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestInitialDNSSeedlistDiscoverySpec/load_balanced -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestLoadBalancerSupport -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration/unified -run TestUnifiedSpec -v -timeout $(TEST_TIMEOUT)s >> test.suite

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
	go test -exec "env PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) LD_LIBRARY_PATH=$(LD_LIBRARY_PATH)" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s ./mongo/integration -run TestClientSideEncryptionSpec >> test.suite
	go test -exec "env PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) LD_LIBRARY_PATH=$(LD_LIBRARY_PATH)" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s ./mongo/integration -run TestClientSideEncryptionProse >> test.suite

.PHONY: evg-test-versioned-api
evg-test-versioned-api:
	# Versioned API related tests are in the mongo, integration and unified packages.
	for TEST_PKG in ./mongo ./mongo/integration ./mongo/integration/unified; do \
		go test -exec "env PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) LD_LIBRARY_PATH=$(LD_LIBRARY_PATH)" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s $$TEST_PKG >> test.suite ; \
	done

.PHONY: build-gcpkms-test
build-gcpkms-test:
	go build $(BUILD_TAGS) ./cmd/testgcpkms

.PHONY: build-awskms-test
build-awskms-test:
	go build $(BUILD_TAGS) ./cmd/testawskms

### Benchmark specific targets and support. ###
.PHONY: benchmark
benchmark:perf
	go test $(BUILD_TAGS) -benchmem -bench=. ./benchmark

.PHONY: driver-benchmark
driver-benchmark:perf
	@go run cmd/godriver-benchmark/main.go | tee perf.suite

perf:driver-test-data.tar.gz
	tar -zxf $< $(if $(eq $(UNAME_S),Darwin),-s , --transform=s)/testdata/perf/
	@touch $@

driver-test-data.tar.gz:
	curl --retry 5 "https://s3.amazonaws.com/boxes.10gen.com/build/driver-test-data.tar.gz" -o driver-test-data.tar.gz --silent --max-time 120
