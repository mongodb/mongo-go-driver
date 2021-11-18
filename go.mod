module go.mongodb.org/mongo-driver

go 1.10

retract (
	[v1.7.0, v1.7.1] // Contains data race bug in background connection establishment.
	[v1.6.0, v1.6.1] // Contains data race bug in background connection establishment.
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-stack/stack v1.8.0
	github.com/golang/snappy v0.0.1
	github.com/google/go-cmp v0.5.2
	github.com/klauspost/compress v1.13.6
	github.com/kr/pretty v0.1.0
	github.com/montanaflynn/stats v0.0.0-20171201202039-1bf9dbcd8cbe
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.6.1
	github.com/tidwall/pretty v1.0.0
	github.com/xdg-go/scram v1.0.2
	github.com/xdg-go/stringprep v1.0.2
	github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d
	golang.org/x/crypto v0.0.0-20201216223049-8b5274cf687f
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	golang.org/x/tools v0.0.0-20190531172133-b3315ee88b7d
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
)
