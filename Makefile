BSON_PKGS = $(shell ./etc/find_pkgs.sh ./bson)
BSON_TEST_PKGS = $(shell ./etc/find_pkgs.sh ./bson _test)
YAMGO_PKGS = $(shell ./etc/find_pkgs.sh ./yamgo)
YAMGO_TEST_PKGS = $(shell ./etc/find_pkgs.sh ./yamgo _test)
PKGS = $(BSON_PKGS) $(YAMGO_PKGS)
TEST_PKGS = $(BSON_TEST_PKGS) $(YAMGO_TEST_PKGS)

TEST_TIMEOUT = 20

.PHONY: default
default: generate test-cover check-fmt vet build-examples lint errcheck

.PHONY: doc
doc:
	godoc -http=:6060 -index

.PHONY: build-examples
build-examples:
	go build $(BUILD_TAGS) ./examples/...

.PHONY: check-fmt
check-fmt:
	@gofmt -l -s $(PKGS) | read; if [ $$? == 0 ]; then echo "gofmt check failed for:"; gofmt -l -s $(PKGS) | sed -e 's/^/ - /'; exit 1; fi

.PHONY: fmt
fmt:
	gofmt -l -s -w $(PKGS)

.PHONY: generate
generate: 
	go generate -x ./bson/... ./yamgo/...

.PHONY: lint
lint:
	golint $(PKGS) | ./etc/lintscreen.pl .lint-whitelist

.PHONY: lint-add-whitelist
lint-add-whitelist:
	golint $(PKGS) | ./etc/lintscreen.pl -u .lint-whitelist

.PHONY: errcheck
errcheck:
	errcheck ./bson/... ./yamgo/...

.PHONY: test
test:
	go test $(BUILD_TAGS) -timeout $(TEST_TIMEOUT)s $(TEST_PKGS)

.PHONY: test-cover
test-cover:
	go test $(BUILD_TAGS) -timeout $(TEST_TIMEOUT)s -cover $(COVER_ARGS) $(TEST_PKGS)

.PHONY: test-race
test-race:
	go test $(BUILD_TAGS) -timeout $(TEST_TIMEOUT)s -race $(TEST_PKGS)

.PHONY: test-short
test-short:
	go test $(BUILD_TAGS) -timeout $(TEST_TIMEOUT)s -short $(TEST_PKGS)

.PHONY: update-bson-corpus-tests
update-bson-corpus-tests:
	etc/update-spec-tests.sh bson-corpus

.PHONY: update-connection-string-tests
update-connection-string-tests:
	etc/update-spec-tests.sh connection-string

.PHONY: update-max-staleness-tests
update-max-staleness-tests:
	etc/update-spec-tests.sh max-staleness

.PHONY: update-server-discovery-and-monitoring-tests
update-server-discovery-and-monitoring-tests:
	etc/update-spec-tests.sh server-discovery-and-monitoring

.PHONY: update-server-selection-tests
update-server-selection-tests:
	etc/update-spec-tests.sh server-selection

.PHONY: vet
vet:
	go tool vet -composites=false -structtags=false -unusedstringmethods="Error" $(PKGS)


# Evergreen specific targets
.PHONY: evg-test
evg-test:
	go test $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s $(TEST_PKGS) > test.suite

.PHONY: evg-test-auth
evg-test-auth:
	go run -tags gssapi ./examples/count/main.go -uri $(MONGODB_URI)
