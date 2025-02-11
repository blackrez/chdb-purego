# non-official go binding for chDB (using purego)

If you are looking for the official golang, it's available at

https://github.com/chdb-io/chdb-go

This go binding don't use cgo for bind chdb.

## Motivations

My first motivation was to learn purego and facilitate the distribution and usage of the chdb binding.
Feel free to fork or open an issue.


## Installation

You need to download and decompress libchb.so on your system.

The download links are on the release page of chdb : 

https://github.com/chdb-io/chdb/releases/tag/v3.0.0
```
linux-aarch64-libchdb.tar.gz
linux-x86_64-libchdb.tar.gz
macos-arm64-libchdb.tar.gz
macos-x86_64-libchdb.tar.gz

```


The module looks the lib to this path

```
        "/usr/local/lib/libchdb.so",
        "/opt/homebrew/lib/libchdb.so",
```

or use `CHDB_LIB_PATH` as an environement variable.


## Basic usage 

Install the go module

Like the API of chdb, it has 2 methods 

```

package main

import (
    "fmt"
    "github.com/blackrez/chdb-purego"
)


func main() {
    conn, err := chdb.Connect(":memory:")
    if err != nil {
        panic(err)
    }
    defer conn.Close()
    q := `
    SELECT
    floor(randNormal(100, 5)) AS k,
    count(*) AS c,
FROM numbers(10) GROUP BY k ORDER BY k ASC 
    `
    result := conn.Query(q, "JSON")
    defer result.Free()
    
    fmt.Println(string(result.Data))
}
```


Then run 

```
CHDB_LIB_PATH="/$HOME/chdb-purego/libchdb.so" go run main.go
```

```
(base) nabil in ~/project/chdb-purego/example λ CGO_ENABLED=0 CHDB_LIB_PATH="/Users/nabil/project/chdb-purego/libchdb.so" go run main.go
{
	"meta":
	[
		{
			"name": "k",
			"type": "Float64"
		},
		{
			"name": "c",
			"type": "UInt64"
		}
	],

	"data":
	[
		{
			"k": 91,
			"c": 1
		},
		{
			"k": 94,
			"c": 1
		},
		{
			"k": 99,
			"c": 3
		},
		{
			"k": 102,
			"c": 1
		},
		{
			"k": 104,
			"c": 1
		},
		{
			"k": 106,
			"c": 3
		}
	],

	"rows": 6,

	"statistics":
	{
		"elapsed": 0.0226755,
		"rows_read": 0,
		"bytes_read": 0
	}
}
```


## Roadmap :


The objective is to add the support of chproto https://github.com/ClickHouse/ch-go/tree/main/proto for an easier data exchange between go and clickhouse.

- [ ] more tests
- [ ] add chproto support
- [ ] better handling lib path
- [ ] test (and support) persistent sessions
- [ ] add sql interface
- [ ] add optionnal go-arrow binding
