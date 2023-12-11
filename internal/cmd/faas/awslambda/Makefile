.PHONY: default
default:
	sam build --debug
	# Cannot connect to a local mongo server on non x86_64 architectures.
	# These cases require either building a mongodb docker container or
	# using a serverless service such as Atlas.
	sam local invoke --parameter-overrides "MongoDbUri=${MONGODB_URI}"
