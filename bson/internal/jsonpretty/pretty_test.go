package pretty

import (
	"bytes"
	gojson "encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func j(json interface{}) string {
	var v interface{}
	if err := gojson.Unmarshal([]byte(fmt.Sprintf("%s", json)), &v); err != nil {
		fmt.Printf(">>%s<<\n", json)
		panic(err)
	}
	data, err := gojson.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(data)
}

var example1 = []byte(`
{
	"name": {
		"last": "Sanders",
		"first": "Janet"
	}, 
	"children": [
		"Andy", "Carol", "Mike"
	],
	"values": [
		10.10, true, false, null, "hello", {}
	],
	"values2": {},
	"values3": [],
	"deep": {"deep":{"deep":[1,2,3,4,5]}}
}
`)

var example2 = `[ 0, 10, 10.10, true, false, null, "hello \" "]`

func TestPretty(t *testing.T) {
	pretty := Pretty(Ugly(Pretty([]byte(example1))))
	assert.Equal(t, j(pretty), j(pretty))
	assert.Equal(t, j(example1), j(pretty))
	pretty = Pretty(Ugly(Pretty([]byte(example2))))
	assert.Equal(t, j(pretty), j(pretty))
	assert.Equal(t, j(example2), j(pretty))
	pretty = Pretty([]byte(" "))
	assert.Equal(t, "", string(pretty))
	opts := *DefaultOptions
	opts.SortKeys = true
	pretty = NewOptions(Ugly(Pretty([]byte(example2))), &opts)
	assert.Equal(t, j(pretty), j(pretty))
	assert.Equal(t, j(example2), j(pretty))
}

func TestUgly(t *testing.T) {
	ugly := Ugly([]byte(example1))
	var buf bytes.Buffer
	err := gojson.Compact(&buf, []byte(example1))
	assert.Equal(t, nil, err)
	assert.Equal(t, buf.Bytes(), ugly)
	ugly = UglyInPlace(ugly)
	assert.Equal(t, nil, err)
	assert.Equal(t, buf.Bytes(), ugly)
}

func TestRandom(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 100000; i++ {
		b := make([]byte, 1024)
		rand.Read(b)
		Pretty(b)
		Ugly(b)
	}
}
func TestBig(t *testing.T) {
	json := `[
  {
    "_id": "58d19e070f4898817162964a",
    "index": "<ReferenceError: indexkk is not defined>",
    "guid": "65d46c3e-9d3a-4bfe-bab2-252f36a53c6b",
    "isActive": false,
    "balance": "$1,064.00",
    "picture": "http://placehold.it/32x32",
    "age": 37,
    "eyeColor": "brown",
    "name": "Chan Orr",
    "gender": "male",
    "company": "SURETECH",
    "email": "chanorr@suretech.com",
    "phone": "+1 (808) 496-3754",
    "address": "792 Bushwick Place, Glenbrook, Vermont, 9893",
    "about": "Amet consequat eu enim laboris cillum ad laboris in quis laboris reprehenderit. Eu deserunt occaecat dolore eu veniam non dolore et magna ex incididunt. Ea dolor laboris ex officia culpa laborum amet adipisicing laboris tempor magna elit mollit ad. Tempor ex aliqua mollit enim laboris sunt fugiat. Sint sunt ex est non dolore consectetur culpa ullamco id dolor nulla labore. Sunt duis fugiat cupidatat sunt deserunt qui aute elit consequat sint cupidatat. Consequat ullamco aliqua nulla velit tempor aute.\r\n",
    "registered": "2014-08-04T04:09:10 +07:00",
    "latitude": 80.707807,
    "longitude": 18.857548,
    "tags": [
      "consectetur",
      "est",
      "cupidatat",
      "nisi",
      "incididunt",
      "aliqua",
      "ullamco"
    ],
    "friends": [
      {
        "id": 0,
        "name": "Little Edwards"
      },
      {
        "id": 1,
        "name": "Gay Johns"
      },
      {
        "id": 2,
        "name": "Hoover Noble"
      }
    ],
    "greeting": "Hello, Chan Orr! You have 3 unread messages.",
    "favoriteFruit": "banana"
  },
  {
    "_id": "58d19e07c2119248f8fa11ff",
    "index": "<ReferenceError: indexkk is not defined>",
    "guid": "b362f0a0-d1ed-4b94-9d6b-213712620a20",
    "isActive": false,
    "balance": "$1,321.26",
    "picture": "http://placehold.it/32x32",
    "age": 28,
    "eyeColor": "blue",
    "name": "Molly Hyde",
    "gender": "female",
    "company": "QUALITEX",
    "email": "mollyhyde@qualitex.com",
    "phone": "+1 (849) 455-2934",
    "address": "440 Visitation Place, Bridgetown, Palau, 5053",
    "about": "Ipsum reprehenderit nulla est nostrud ad incididunt officia in commodo id esse id. Ullamco ullamco commodo mollit ut id cupidatat veniam nostrud minim duis qui sit. Occaecat esse nostrud velit qui non dolor proident. Ipsum ipsum anim non mollit minim voluptate amet irure in. Sunt commodo occaecat aute ullamco sunt fugiat laboris culpa Lorem anim. Aliquip tempor excepteur labore aute deserunt consectetur incididunt aute eu est ullamco consectetur excepteur. Sunt sint consequat cupidatat nisi exercitation minim enim occaecat esse ex amet ex non.\r\n",
    "registered": "2014-09-12T08:51:11 +07:00",
    "latitude": 15.867177,
    "longitude": 165.862595,
    "tags": [
      "enim",
      "sint",
      "elit",
      "laborum",
      "elit",
      "cupidatat",
      "ipsum"
    ],
    "friends": [
      {
        "id": 0,
        "name": "Holmes Hurley"
      },
      {
        "id": 1,
        "name": "Rhoda Spencer"
      },
      {
        "id": 2,
        "name": "Tommie Gallegos"
      }
    ],
    "greeting": "Hello, Molly Hyde! You have 10 unread messages.",
    "favoriteFruit": "banana"
  },
  {
    "_id": "58d19e07fc27eedd9159d710",
    "index": "<ReferenceError: indexkk is not defined>",
    "guid": "1d343fd3-44f7-4246-a5e6-a9297afb3146",
    "isActive": false,
    "balance": "$1,459.65",
    "picture": "http://placehold.it/32x32",
    "age": 26,
    "eyeColor": "brown",
    "name": "Jaime Kennedy",
    "gender": "female",
    "company": "RECRITUBE",
    "email": "jaimekennedy@recritube.com",
    "phone": "+1 (983) 483-3522",
    "address": "997 Vanderveer Street, Alamo, Marshall Islands, 4767",
    "about": "Qui consequat veniam ex enim excepteur aliqua dolor duis Lorem deserunt. Lorem occaecat laboris quis nisi Lorem aute exercitation consectetur officia velit aliqua aliquip commodo. Tempor irure ad ipsum aliquip. Incididunt mollit aute cillum non magna duis officia anim laboris deserunt voluptate.\r\n",
    "registered": "2015-08-31T06:51:25 +07:00",
    "latitude": -7.486839,
    "longitude": 57.659287,
    "tags": [
      "veniam",
      "aliqua",
      "aute",
      "amet",
      "laborum",
      "quis",
      "sint"
    ],
    "friends": [
      {
        "id": 0,
        "name": "Brown Christensen"
      },
      {
        "id": 1,
        "name": "Robyn Whitehead"
      },
      {
        "id": 2,
        "name": "Dolly Weaver"
      }
    ],
    "greeting": "Hello, Jaime Kennedy! You have 3 unread messages.",
    "favoriteFruit": "banana"
  },
  {
    "_id": "58d19e0783c362da4b71240d",
    "index": "<ReferenceError: indexkk is not defined>",
    "guid": "dbe60229-60d2-4879-82f3-d9aca0baaf6f",
    "isActive": false,
    "balance": "$3,221.63",
    "picture": "http://placehold.it/32x32",
    "age": 32,
    "eyeColor": "green",
    "name": "Cherie Vinson",
    "gender": "female",
    "company": "SLAX",
    "email": "cherievinson@slax.com",
    "phone": "+1 (905) 474-3132",
    "address": "563 Macdougal Street, Navarre, New York, 8733",
    "about": "Ad laborum et magna quis veniam duis magna consectetur mollit in minim non officia aliquip. Ullamco dolor qui consectetur adipisicing. Incididunt ad ad incididunt duis velit laboris. Reprehenderit ullamco magna quis exercitation excepteur nisi labore pariatur laborum consequat eu laboris amet velit. Et dolore aliqua proident sunt dolore incididunt dolore fugiat ipsum tempor occaecat.\r\n",
    "registered": "2015-03-19T08:48:47 +07:00",
    "latitude": -56.480034,
    "longitude": -59.894094,
    "tags": [
      "irure",
      "commodo",
      "quis",
      "cillum",
      "quis",
      "nulla",
      "irure"
    ],
    "friends": [
      {
        "id": 0,
        "name": "Danielle Mullins"
      },
      {
        "id": 1,
        "name": "Maxine Peters"
      },
      {
        "id": 2,
        "name": "Francine James"
      }
    ],
    "greeting": "Hello, Cherie Vinson! You have 1 unread messages.",
    "favoriteFruit": "apple"
  },
  {
    "_id": "58d19e07b8f1ea8e3451870d",
    "index": "<ReferenceError: indexkk is not defined>",
    "guid": "91fd9527-770c-4006-a0ed-64ca0d819199",
    "isActive": true,
    "balance": "$2,387.38",
    "picture": "http://placehold.it/32x32",
    "age": 37,
    "eyeColor": "blue",
    "name": "Glenna Hanson",
    "gender": "female",
    "company": "ACUMENTOR",
    "email": "glennahanson@acumentor.com",
    "phone": "+1 (965) 564-3926",
    "address": "323 Seigel Street, Rosedale, Florida, 2700",
    "about": "Commodo id ex velit nulla incididunt occaecat aliquip ullamco consequat est. Esse officia adipisicing magna et et incididunt sit deserunt ex mollit id. Laborum proident sit sit duis proident cillum irure aliquip et commodo.\r\n",
    "registered": "2014-06-29T02:48:04 +07:00",
    "latitude": -6.141759,
    "longitude": 155.991532,
    "tags": [
      "amet",
      "pariatur",
      "culpa",
      "eu",
      "commodo",
      "magna",
      "excepteur"
    ],
    "friends": [
      {
        "id": 0,
        "name": "Blanchard Blackburn"
      },
      {
        "id": 1,
        "name": "Ayers Guy"
      },
      {
        "id": 2,
        "name": "Powers Salinas"
      }
    ],
    "greeting": "Hello, Glenna Hanson! You have 4 unread messages.",
    "favoriteFruit": "strawberry"
  },
  {
    "_id": "58d19e07f1ad063dac8b72dc",
    "index": "<ReferenceError: indexkk is not defined>",
    "guid": "9b8c6cef-cfcd-4e6d-85e4-fe2e6920ec31",
    "isActive": true,
    "balance": "$1,828.58",
    "picture": "http://placehold.it/32x32",
    "age": 29,
    "eyeColor": "green",
    "name": "Hays Shields",
    "gender": "male",
    "company": "ISOLOGICA",
    "email": "haysshields@isologica.com",
    "phone": "+1 (882) 469-3201",
    "address": "574 Columbus Place, Singer, Georgia, 8716",
    "about": "Consectetur et adipisicing ad quis incididunt qui labore et ex elit esse. Ad elit officia ullamco dolor reprehenderit. Sunt nisi ullamco mollit incididunt consectetur nostrud anim adipisicing ullamco aliqua eiusmod ad. Et excepteur voluptate adipisicing velit id quis duis Lorem id deserunt esse irure Lorem. Est irure sint Lorem aliqua adipisicing velit irure Lorem. Ex in culpa laborum nostrud esse eu laboris velit. Anim excepteur ex ipsum amet nostrud cillum.\r\n",
    "registered": "2014-02-10T07:17:14 +07:00",
    "latitude": -66.354543,
    "longitude": 138.400461,
    "tags": [
      "mollit",
      "labore",
      "id",
      "labore",
      "dolor",
      "in",
      "elit"
    ],
    "friends": [
      {
        "id": 0,
        "name": "Mendoza Craig"
      },
      {
        "id": 1,
        "name": "Rowena Carey"
      },
      {
        "id": 2,
        "name": "Barry Francis"
      }
    ],
    "greeting": "Hello, Hays Shields! You have 10 unread messages.",
    "favoriteFruit": "strawberry"
  }
]`

	opts := *DefaultOptions
	opts.SortKeys = true
	jsonb := NewOptions(Ugly([]byte(json)), &opts)
	assert.Equal(t, j(jsonb), j(json))
}
func BenchmarkPretty(t *testing.B) {
	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		Pretty(example1)
	}
}

func BenchmarkPrettySortKeys(t *testing.B) {
	opts := *DefaultOptions
	opts.SortKeys = true
	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		NewOptions(example1, &opts)
	}
}
func BenchmarkUgly(t *testing.B) {
	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		Ugly(example1)
	}
}

func BenchmarkUglyInPlace(t *testing.B) {
	example2 := []byte(string(example1))
	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		UglyInPlace(example2)
	}
}
func BenchmarkJSONIndent(t *testing.B) {
	var dst bytes.Buffer
	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		_ = gojson.Indent(&dst, example1, "", "  ")
	}
}

func BenchmarkJSONCompact(t *testing.B) {
	var dst bytes.Buffer
	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		_ = gojson.Compact(&dst, example1)
	}
}
