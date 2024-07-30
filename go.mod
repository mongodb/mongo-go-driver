module go.mongodb.org/mongo-driver/v2

go 1.18

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/golang/snappy v0.0.1
	github.com/google/go-cmp v0.5.2
	github.com/klauspost/compress v1.13.6
	github.com/xdg-go/scram v1.1.2
	github.com/xdg-go/stringprep v1.0.4
	github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d
	golang.org/x/crypto v0.0.0-20220622213112-05595931fe9d
	golang.org/x/sync v0.0.0-20220722155255-886fb9371eb4
)

require (
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	golang.org/x/text v0.7.0 // indirect
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543 // indirect
)

replace golang.org/x/net/http2 => golang.org/x/net/http2 v0.23.0 // GODRIVER-3225
