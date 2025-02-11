package chdb

import (
	"fmt"
	"testing"
)



func TestBasicConnect(t *testing.T){
	_result := `[42]
`
	conn, err := Connect(":memory:")
    if err != nil {
        panic(err)
    }
    defer conn.Close()
    q := `SELECT 42`
    result := conn.Query(q, "JSONCompactEachRow")
    defer result.Free()
    if (_result != string(result.Data)){
    	t.Fatalf(`Basic test failed`)
    }
}