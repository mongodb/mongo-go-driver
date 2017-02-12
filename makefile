PKGS = ./auth ./core ./core/connstring ./core/desc ./core/msg ./ops
LINTARGS = -min_confidence="0.3"
TEST_TIMEOUT = 20

default: test-cover lint vet

doc:
    godoc -http=:6060 -index

lint:
    golint $(LINTARGS) ./auth
    golint $(LINTARGS) ./core
    golint $(LINTARGS) ./core/connstring
    golint $(LINTARGS) ./core/desc
    golint $(LINTARGS) ./core/msg
    golint $(LINTARGS) ./ops

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
