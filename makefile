PKGS = ./auth ./bson ./cluster ./conn ./connstring ./internal ./internal/feature ./msg ./ops ./readpref ./server
VETPKGS = ./auth ./bson ./cluster ./conn ./connstring ./msg ./ops ./readpref ./server
LINTARGS = -min_confidence="0.3"
TEST_TIMEOUT = 20

default: generate test-cover lint vet build-examples

doc:
	godoc -http=:6060 -index

build-examples:
	go build $(BUILD_TAGS) ./examples/...

generate:
	go generate $(PKGS)

lint:
	golint $(LINTARGS) ./auth
	golint -min_confidence="1.0" ./bson
	golint $(LINTARGS) ./cluster
	golint $(LINTARGS) ./conn
	golint $(LINTARGS) ./connstring
	golint $(LINTARGS) ./internal/auth
	golint $(LINTARGS) ./internal/feature
	golint $(LINTARGS) ./msg
	golint $(LINTARGS) ./ops
	golint $(LINTARGS) ./readpref
	golint $(LINTARGS) ./server

test:
	go test $(BUILD_TAGS) -timeout $(TEST_TIMEOUT)s $(PKGS)

test-cover:
	go test $(BUILD_TAGS) -timeout $(TEST_TIMEOUT)s -cover $(COVER_ARGS) $(PKGS)

test-race:
	go test $(BUILD_TAGS) -timeout $(TEST_TIMEOUT)s -race $(PKGS)

test-short:
	go test $(BUILD_TAGS) -timeout $(TEST_TIMEOUT)s -short $(PKGS)

vet:
	go tool vet -composites=false -structtags=false -unusedstringmethods="Error" $(VETPKGS)


# Evergreen specific targets
evg-test:
	go test $(BUILD_TAGS) -v -timeout $(TEST_TIMEOUT)s $(PKGS) > test.suite

evg-test-auth:
	go run -tags gssapi ./examples/auth/main.go
