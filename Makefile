BSON_PKGS = $(shell etc/list_pkgs.sh ./bson)
BSON_TEST_PKGS = $(shell etc/list_test_pkgs.sh ./bson)
EVENT_PKGS = $(shell etc/list_pkgs.sh ./event)
EVENT_TEST_PKGS = $(shell etc/list_test_pkgs.sh ./event)
MONGO_PKGS = $(shell etc/list_pkgs.sh ./mongo)
MONGO_TEST_PKGS = $(shell etc/list_test_pkgs.sh ./mongo)
UNSTABLE_PKGS = $(shell etc/list_pkgs.sh ./x)
UNSTABLE_TEST_PKGS = $(shell etc/list_test_pkgs.sh ./x)
TAG_PKG = $(shell etc/list_pkgs.sh ./tag)
TAG_TEST_PKG = $(shell etc/list_test_pkgs.sh ./tag)
EXAMPLES_PKGS = $(shell etc/list_pkgs.sh ./examples)
EXAMPLES_TEST_PKGS = $(shell etc/list_test_pkgs.sh ./examples)
PKGS = $(BSON_PKGS) $(EVENT_PKGS) $(MONGO_PKGS) $(UNSTABLE_PKGS) $(TAG_PKG) $(EXAMPLES_PKGS)
TEST_PKGS = $(BSON_TEST_PKGS) $(EVENT_TEST_PKGS) $(MONGO_TEST_PKGS) $(UNSTABLE_TEST_PKGS) $(TAG_PKG) $(EXAMPLES_TEST_PKGS)
ATLAS_URIS = "$(ATLAS_FREE)" "$(ATLAS_REPLSET)" "$(ATLAS_SHARD)" "$(ATLAS_TLS11)" "$(ATLAS_TLS12)" "$(ATLAS_FREE_SRV)" "$(ATLAS_REPLSET_SRV)" "$(ATLAS_SHARD_SRV)" "$(ATLAS_TLS11_SRV)" "$(ATLAS_TLS12_SRV)" "$(ATLAS_SERVERLESS)" "$(ATLAS_SERVERLESS_SRV)"
GODISTS=linux/amd64 linux/386 linux/arm64 linux/arm linux/s390x

TEST_TIMEOUT = 1800

.PHONY: default
default: check-env check-fmt build-examples lint test-cover test-race

.PHONY: check-env
check-env:
	etc/check_env.sh

.PHONY: doc
doc:
	godoc -http=:6060 -index

.PHONY: build-examples
build-examples:
	go build $(BUILD_TAGS) ./examples/... ./x/mongo/driver/examples/...

.PHONY: build
build:
	go build $(BUILD_TAGS) $(filter-out ./core/auth/internal/gssapi,$(PKGS))

.PHONY: build-no-tags
build-no-tags:
	go build $(filter-out ./core/auth/internal/gssapi,$(PKGS))

.PHONY: build-tests
build-tests:
	for TEST in $(PKGS); do \
		go test $(BUILD_TAGS) -c $$TEST ; \
		if [ $$? -ne 0 ]; \
		then \
			exit 1; \
		fi \
	done

.PHONY: check-fmt
check-fmt:
	etc/check_fmt.sh $(PKGS)

.PHONY: fmt
fmt:
	gofmt -l -s -w $(PKGS)

.PHONY: lint
lint:
	for dist in $(GODISTS); do \
		goos=$$(echo $$dist | cut -d/ -f 1) ; \
		goarch=$$(echo $$dist | cut -d/ -f 2) ; \
		command="GOOS=$$goos GOARCH=$$goarch golangci-lint run --config .golangci.yml ./..." ; \
		echo $$command ; \
		eval $$command ; \
	done

.PHONY: test
test:
	for TEST in $(TEST_PKGS) ; do \
		go test $(BUILD_TAGS) -timeout $(TEST_TIMEOUT)s $$TEST ; \
	done

.PHONY: test-cover
test-cover:
	for TEST in $(TEST_PKGS) ; do \
    	go test $(BUILD_TAGS) -timeout $(TEST_TIMEOUT)s -cover $(COVER_ARGS) $$TEST ; \
    done

.PHONY: test-race
test-race:
	for TEST in $(TEST_PKGS) ; do \
    	go test $(BUILD_TAGS) -timeout $(TEST_TIMEOUT)s -race $(COVER_ARGS) $$TEST ; \
    done

.PHONY: test-short
test-short:
	for TEST in $(TEST_PKGS) ; do \
    	go test $(BUILD_TAGS) -timeout $(TEST_TIMEOUT)s -short $(COVER_ARGS) $$TEST ; \
    done

.PHONY: update-bson-corpus-tests
update-bson-corpus-tests:
	etc/update-spec-tests.sh bson-corpus

.PHONY: update-connection-string-tests
update-connection-string-tests:
	etc/update-spec-tests.sh connection-string

.PHONY: update-crud-tests
update-crud-tests:
	etc/update-spec-tests.sh crud

.PHONY: update-initial-dns-seedlist-discovery-tests
update-initial-dns-seedlist-discovery-tests:
	etc/update-spec-tests.sh initial-dns-seedlist-discovery

.PHONY: update-max-staleness-tests
update-max-staleness-tests:
	etc/update-spec-tests.sh max-staleness

.PHONY: update-server-discovery-and-monitoring-tests
update-server-discovery-and-monitoring-tests:
	etc/update-spec-tests.sh server-discovery-and-monitoring

.PHONY: update-server-selection-tests
update-server-selection-tests:
	etc/update-spec-tests.sh server-selection

.PHONY: update-notices
update-notices:
	etc/generate-notices.pl > THIRD-PARTY-NOTICES

# Evergreen specific targets
.PHONY: evg-test
evg-test:
	for TEST in $(TEST_PKGS); do \
		go test -exec "env PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) LD_LIBRARY_PATH=$(LD_LIBRARY_PATH)" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s $$TEST >> test.suite ; \
	done

.PHONY: evg-test-auth
evg-test-auth:
	go run -tags gssapi ./x/mongo/driver/examples/count/main.go -uri $(MONGODB_URI)

.PHONY: evg-test-atlas
evg-test-atlas:
	go run ./mongo/testatlas/main.go $(ATLAS_URIS)

.PHONY: evg-test-ocsp
evg-test-ocsp:
	go test -v ./mongo -run TestOCSP $(OCSP_TLS_SHOULD_SUCCEED) >> test.suite

.PHONY: build-aws-ecs-test
build-aws-ecs-test:
	go build $(BUILD_TAGS) ./mongo/testaws/main.go

.PHONY: evg-test-atlas-data-lake
evg-test-atlas-data-lake:
	ATLAS_DATA_LAKE_INTEGRATION_TEST=true go test -v ./mongo/integration -run TestUnifiedSpecs/atlas-data-lake-testing >> spec_test.suite
	ATLAS_DATA_LAKE_INTEGRATION_TEST=true go test -v ./mongo/integration -run TestAtlasDataLake >> spec_test.suite

.PHONY: evg-test-versioned-api
evg-test-versioned-api:
	for TEST_PKG in ./mongo ./mongo/integration ./mongo/integration/unified; do \
		go test -exec "env PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) LD_LIBRARY_PATH=$(LD_LIBRARY_PATH)" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s $$TEST_PKG >> test.suite ; \
	done

.PHONY: evg-test-load-balancers
evg-test-load-balancers:
	go test $(BUILD_TAGS) ./mongo/integration -run TestUnifiedSpecs/retryable-reads -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestRetryableWritesSpec -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestChangeStreamSpec -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestInitialDNSSeedlistDiscoverySpec/load_balanced -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestLoadBalancerSupport -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration/unified -run TestUnifiedSpec -v -timeout $(TEST_TIMEOUT)s >> test.suite

.PHONY: evg-test-kms
evg-test-kms:
	go test -exec "env PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) LD_LIBRARY_PATH=$(LD_LIBRARY_PATH)" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s ./mongo/integration -run TestClientSideEncryptionProse/kms_tls_tests >> test.suite

.PHONY: evg-test-kmip
evg-test-kmip:
	go test -exec "env PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) LD_LIBRARY_PATH=$(LD_LIBRARY_PATH)" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s ./mongo/integration -run TestClientSideEncryptionSpec/kmipKMS >> test.suite
	go test -exec "env PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) LD_LIBRARY_PATH=$(LD_LIBRARY_PATH)" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s ./mongo/integration -run TestClientSideEncryptionProse/data_key_and_double_encryption >> test.suite
	go test -exec "env PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) LD_LIBRARY_PATH=$(LD_LIBRARY_PATH)" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s ./mongo/integration -run TestClientSideEncryptionProse/corpus >> test.suite
	go test -exec "env PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) LD_LIBRARY_PATH=$(LD_LIBRARY_PATH)" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s ./mongo/integration -run TestClientSideEncryptionProse/custom_endpoint >> test.suite
	go test -exec "env PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) LD_LIBRARY_PATH=$(LD_LIBRARY_PATH)" $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s ./mongo/integration -run TestClientSideEncryptionProse/kms_tls_options_test >> test.suite

.PHONY: evg-test-serverless
evg-test-serverless:
	go test $(BUILD_TAGS) ./mongo/integration -run TestCrudSpec -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestRetryableWritesSpec -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestUnifiedSpecs/retryable-reads -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestUnifiedSpecs/sessions -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration/unified -run TestUnifiedSpec/crud/unified -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration/unified -run TestUnifiedSpec/transactions/unified -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration/unified -run TestUnifiedSpec/versioned-api -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestUnifiedSpecs/transactions/legacy -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestWriteErrorsWithLabels -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestWriteErrorsDetails -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestHintErrors -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestAggregatePrimaryPreferredReadPreference -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestWriteConcernError -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestErrorsCodeNamePropagated -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestRetryableWritesProse -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration -run TestCursor -v -timeout $(TEST_TIMEOUT)s >> test.suite
	go test $(BUILD_TAGS) ./mongo/integration/unified -run TestUnifiedSpec/load-balancers -v -timeout $(TEST_TIMEOUT)s >> test.suite

# benchmark specific targets and support
perf:driver-test-data.tar.gz
	tar -zxf $< $(if $(eq $(UNAME_S),Darwin),-s , --transform=s)/data/perf/
	@touch $@
driver-test-data.tar.gz:
	curl --retry 5 "https://s3.amazonaws.com/boxes.10gen.com/build/driver-test-data.tar.gz" -o driver-test-data.tar.gz --silent --max-time 120
benchmark:perf
	go test $(BUILD_TAGS) -benchmem -bench=. ./benchmark
driver-benchmark:perf
	@go run cmd/godriver-benchmark/main.go | tee perf.suite
.PHONY:benchmark driver-benchmark
