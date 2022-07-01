module go.mongodb.org/mongo-driver

go 1.10

retract (
	[v1.7.0, v1.7.1] // Contains data race bug in background connection establishment.
	[v1.6.0, v1.6.1] // Contains data race bug in background connection establishment.
)

// gopkg.in/yaml.v3@v3.0.0-20200313102051-9f266ea9e77c introduced through github.com/stretchr/testify@v1.6.1 is
// vulnerable to Denial of Service (DoS) via the Unmarshal function, which causes the program to crash when attempting
// to deserialize invalid input. https://www.cve.org/CVERecord?id=CVE-2022-28948
replace gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c => gopkg.in/yaml.v3 v3.0.1

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golang/snappy v0.0.1
	github.com/google/go-cmp v0.5.2
	github.com/klauspost/compress v1.13.6
	github.com/kr/pretty v0.1.0
	github.com/montanaflynn/stats v0.0.0-20171201202039-1bf9dbcd8cbe
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.6.1
	github.com/tidwall/pretty v1.0.0
	github.com/xdg-go/scram v1.1.1
	github.com/xdg-go/stringprep v1.0.3
	github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d
	golang.org/x/crypto v0.0.0-20220622213112-05595931fe9d
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
