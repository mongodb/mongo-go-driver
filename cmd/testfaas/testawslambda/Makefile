.PHONY: build

build:
	sam build --debug
	sam local invoke --parameter-overrides "MongoDbUri=${MONGODB_URI}"
