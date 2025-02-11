package chdb

import (
    "errors"
    "os/exec"
    "os"
    "runtime"
    "unsafe"
    "github.com/ebitengine/purego"
)

type local_result struct {
    buf       *byte
    len       uintptr
    _vec      unsafe.Pointer
    elapsed   float64
    rows_read uint64
    bytes_read uint64
}

type local_result_v2 struct {
    buf          *byte
    len          uintptr
    _vec         unsafe.Pointer
    elapsed      float64
    rows_read    uint64
    bytes_read   uint64
    error_message *byte
}

type chdb_conn struct {
    server    unsafe.Pointer
    connected bool
    queue     unsafe.Pointer
}

// TODO remove old API ?
var (
    queryStable    func(argc int, argv **byte) *local_result
    freeResult     func(result *local_result)
    queryStableV2  func(argc int, argv **byte) *local_result_v2
    freeResultV2   func(result *local_result_v2)
    connectChdb    func(argc int, argv **byte) **chdb_conn
    closeConn      func(conn **chdb_conn)
    queryConn      func(conn *chdb_conn, query *byte, format *byte) *local_result_v2
)

// Result represent the results from a query.
// Data is byte formatted 
// Don't forget to free memory with freeFunc.
type Result struct {
    Data        []byte
    Elapsed     float64
    RowsRead    uint64
    BytesRead   uint64
    Error       string
    freeFunc    func()
}

// Conn is the connection to the database.
type Conn struct {
    cConn **chdb_conn
}

func init() {
	// TODO: move to internal    
	libchdb_path := findLibrary()
    libchdb, err := purego.Dlopen(libchdb_path, purego.RTLD_NOW|purego.RTLD_GLOBAL)
    if err != nil {
        panic(err)
    }

    purego.RegisterLibFunc(&queryStable, libchdb, "query_stable")
    purego.RegisterLibFunc(&freeResult, libchdb, "free_result")
    purego.RegisterLibFunc(&queryStableV2, libchdb, "query_stable_v2")
    purego.RegisterLibFunc(&freeResultV2, libchdb, "free_result_v2")
    purego.RegisterLibFunc(&connectChdb, libchdb, "connect_chdb")
    purego.RegisterLibFunc(&closeConn, libchdb, "close_conn")
    purego.RegisterLibFunc(&queryConn, libchdb, "query_conn")
}

// NewResultFromV2 is the wrapper of Result from the low-level local_result_v2. 
func NewResultFromV2(cRes *local_result_v2) *Result {
    res := &Result{
        Elapsed:   cRes.elapsed,
        RowsRead:  cRes.rows_read,
        BytesRead: cRes.bytes_read,
        freeFunc:  func() { freeResultV2(cRes) },
    }

    if cRes.buf != nil && cRes.len > 0 {
        res.Data = unsafe.Slice(cRes.buf, cRes.len)
    }
    
    if cRes.error_message != nil {
        res.Error = ptrToGoString(cRes.error_message)
    }
    
    return res
}

// Free release memory from Result
func (r *Result) Free() {
    if r.freeFunc != nil {
        r.freeFunc()
    }
}

// Connect manage the connection.
func Connect(connStr string) (*Conn, error) {
    // string can not be empty
    if connStr == "" {
        return nil, errors.New("connection string cannot be empty")
    }
    
    // transform uri to args for connect
    args := []string{connStr}
    argc, argv := convertArgs(args)
    
    cConn := connectChdb(argc, argv)
    if cConn == nil {
        return nil, errors.New("connection failed for: " + connStr)
    }
    
    // check is the connection works
    conn := &Conn{cConn: cConn}
    if !(*cConn).connected {
        return nil, errors.New("connection not properly initialized")
    }
    
    return conn, nil
}

// Query execute the SQL query with a format as parameters.
// format can be found at the clickhouse documentation :
// https://clickhouse.com/docs/en/interfaces/formats
// it return Result.
func (c *Conn) Query(query, format string) *Result {
	//TODO verify format if avaiable in clickhouse
    var pinner runtime.Pinner
    defer pinner.Unpin()

    qPtr := stringToPtr(query, &pinner)
    fPtr := stringToPtr(format, &pinner)
    
    conn := *c.cConn
    cRes := queryConn(conn, qPtr, fPtr)
    return NewResultFromV2(cRes)
}

// Close close the connection
func (c *Conn) Close() {
    if c.cConn != nil {
        closeConn(c.cConn)
        c.cConn = nil
    }
}

// Helpers
// TODO: move to internal
func ptrToGoString(ptr *byte) string {
    if ptr == nil {
        return ""
    }
    
    var length int
    for {
        if *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(ptr)) + uintptr(length))) == 0 {
            break
        }
        length++
    }
    
    return string(unsafe.Slice(ptr, length))
}


func stringToPtr(s string, pinner *runtime.Pinner) *byte {
    // Pinne for convert string to bytes
    // maybe there is simpler solution but it was late when I write this code.
    data := make([]byte, len(s)+1)
    copy(data, s)
    data[len(s)] = 0 // Null-terminator
    
    ptr := &data[0]
    pinner.Pin(ptr)
    
    return ptr
}

func convertArgs(args []string) (int, **byte) {
	// maybe there is simpler solution but it was late when I write this code.
    if len(args) == 0 {
        return 0, nil
    }

    var pinner runtime.Pinner
    defer pinner.Unpin()

    // Création d'un tableau C de pointeurs
    cArgs := make([]*byte, len(args))
    
    for i, arg := range args {
        argData := make([]byte, len(arg)+1)
        copy(argData, arg)
        argData[len(arg)] = 0
        
        ptr := &argData[0]
        pinner.Pin(ptr)
        cArgs[i] = ptr
    }
    
    if len(cArgs) > 0 {
        pinner.Pin(&cArgs[0])
    }
    
    return len(args), (**byte)(unsafe.Pointer(&cArgs[0]))
}


// find the library libchdb.so
// TODO find a better solution
func findLibrary() string {
    // Env var
    if envPath := os.Getenv("CHDB_LIB_PATH"); envPath != "" {
        return envPath
    }
    
    // ldconfig with Linux
    if path, err := exec.LookPath("libchdb.so"); err == nil {
        return path
    }
    
    // default path
    commonPaths := []string{
        "/usr/local/lib/libchdb.so",
        "/opt/homebrew/lib/libchdb.dylib",
    }
    
    for _, p := range commonPaths {
        if _, err := os.Stat(p); err == nil {
            return p
        }
    }

    //should be an error ?
    return "libchdb.so"
}



/**
 * libchdb have the name on linux and macos.
 * This code is not needed for now
func getSystemLibrary() string {
	switch runtime.GOOS {
	case "darwin":
		return ""
	case "linux":
		return ""
	default:
		panic(fmt.Errorf("GOOS=%s is not supported", runtime.GOOS))

	}
}
**/