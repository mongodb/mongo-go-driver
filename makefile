BSON_PKGS = $(shell ./find_pkgs.sh ./bson)
BSON_TEST_PKGS = $(shell ./find_pkgs.sh ./bson _test)
YAMGO_PKGS = $(shell ./find_pkgs.sh ./yamgo)
YAMGO_TEST_PKGS = $(shell ./find_pkgs.sh ./yamgo _test)
PKGS = $(BSON_PKGS) $(YAMGO_PKGS)
TEST_PKGS = $(BSON_TEST_PKGS) $(YAMGO_TEST_PKGS)

LINTARGS = -min_confidence="0.3"
TEST_TIMEOUT = 20

.PHONY: default
default: generate test-cover check-fmt lint vet build-examples

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
	golint $(LINTARGS) $(YAMGO_PKGS)
	golint -min_confidence="1.0"  $(BSON_PKGS)

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
