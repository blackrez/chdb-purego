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
    42`
    result := conn.Query(q, "JSONColumns")
    defer result.Free()
    
    fmt.Println(string(result.Data))
}