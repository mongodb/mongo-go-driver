PKGS = ./auth ./cluster ./conn ./connstring ./internal/feature ./msg ./ops ./readpref ./server
LINTARGS = -min_confidence="0.3"
TEST_TIMEOUT = 20

default: generate test-cover lint vet build-examples

doc:
    godoc -http=:6060 -index

build-examples:
    go build ./examples/...

generate:
    go generate $(PKGS)

lint: 
    golint $(LINTARGS) ./auth
    golint $(LINTARGS) ./cluster
    golint $(LINTARGS) ./conn
    golint $(LINTARGS) ./connstring
    golint $(LINTARGS) ./internal/feature
    golint $(LINTARGS) ./msg
    golint $(LINTARGS) ./ops
    golint $(LINTARGS) ./readpref
    golint $(LINTARGS) ./server

test: 
    go test -timeout $(TEST_TIMEOUT)s $(PKGS)

test-cover:
    go test -cover -timeout $(TEST_TIMEOUT)s $(PKGS)

test-race:
    go test -race -timeout $(TEST_TIMEOUT)s $(PKGS)

test-short:
    go test -short -timeout $(TEST_TIMEOUT)s $(PKGS)

vet: 
    go tool vet -composites=false $(PKGS)
