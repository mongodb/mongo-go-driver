PKGS = ./auth ./bson ./cluster ./conn ./connstring ./internal ./internal/feature ./msg ./ops ./readpref ./server
LINTARGS = -min_confidence="0.3"
TEST_TIMEOUT = 20
BUILD_TAGS = -tags gssapi

default: generate test-cover lint vet build-examples

doc:
	godoc -http=:6060 -index

build-examples:
	go build $(BUILD_TAGS) ./examples/...

generate:
	go generate $(PKGS)

lint:
	golint $(LINTARGS) ./auth
	golint $(LINTARGS) ./bson
	golint $(LINTARGS) ./cluster
	golint $(LINTARGS) ./conn
	golint $(LINTARGS) ./connstring
	golint $(LINTARGS) ./internal
	golint $(LINTARGS) ./internal/feature
	golint $(LINTARGS) ./msg
	golint $(LINTARGS) ./ops
	golint $(LINTARGS) ./readpref
	golint $(LINTARGS) ./server

test:
	go test $(BUILD_TAGS) -timeout $(TEST_TIMEOUT)s $(PKGS)

test-cover:
	go test $(BUILD_TAGS) -cover -timeout $(TEST_TIMEOUT)s $(PKGS)

test-race:
	go test $(BUILD_TAGS) -race -timeout $(TEST_TIMEOUT)s $(PKGS)

test-short:
	go test $(BUILD_TAGS) -short -timeout $(TEST_TIMEOUT)s $(PKGS)

vet:
	go tool vet -composites=false $(PKGS)
