module go.mongodb.org/mongo-driver

go 1.13

retract (
	v1.10.0 // Contains a possible data corruption bug in RewrapManyDataKey when using libmongocrypt versions less than 1.5.2.
	[v1.7.0, v1.7.1] // Contains data race bug in background connection establishment.
	[v1.6.0, v1.6.1] // Contains data race bug in background connection establishment.
)

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/golang/snappy v0.0.1
	github.com/google/go-cmp v0.5.2
	github.com/klauspost/compress v1.13.6
	github.com/kr/pretty v0.1.0
	github.com/montanaflynn/stats v0.0.0-20171201202039-1bf9dbcd8cbe
	github.com/xdg-go/scram v1.1.1
	github.com/xdg-go/stringprep v1.0.3
	github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d
	golang.org/x/crypto v0.0.0-20220622213112-05595931fe9d
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
)
