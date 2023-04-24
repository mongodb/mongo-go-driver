.PHONY: build

GC=go
HOST=host.docker.internal
CONNSTRING=mongodb://${HOST}:27017


export GO111MODULE=on

build:
	rm -f hello-world/bootstrap # remove the old binary
	cd hello-world && GOOS=linux GOARCH=amd64 $(GC) build -o ./bootstrap ./...
	#sam build
	sam local invoke --parameter-overrides "MongoDbUri=$(CONNSTRING)" \
		--container-host-interface 0.0.0.0 \
		--container-host=$(HOST)

