package server

import (
	"encoding/json"
	"fmt"
	"godb/index"
	"godb/query"
	"godb/storage"
	"godb/wal"
	"net/http"
	"strings"
)

var (
	store   *storage.StorageEngine
	idxMgr  *index.IndexManager
	qEngine *query.Engine
	walLog  *wal.WAL
)

// response is a generic API response wrapper
type response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func Start(addr string) {
	var err error

	// Initialize storage
	store, err = storage.NewStorageEngine("./data")
	if err != nil {
		panic(fmt.Sprintf("failed to init storage: %v", err))
	}

	// Initialize WAL
	walLog, err = wal.NewWAL("./data/wal.log")
	if err != nil {
		panic(fmt.Sprintf("failed to init WAL: %v", err))
	}
	defer walLog.Close()

	// Initialize index manager and query engine
	idxMgr = index.NewIndexManager()
	qEngine = query.NewEngine(store)

	// Serve frontend
	http.Handle("/ui/", http.StripPrefix("/ui/", http.FileServer(http.Dir("./frontend"))))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/ui/", http.StatusFound)
		}
	})

	// Routes
	
	http.HandleFunc("/tables", withCORS(handleTables))
	http.HandleFunc("/tables/", withCORS(handleTableOps))
	http.HandleFunc("/query", withCORS(handleQuery))
	http.HandleFunc("/index", withCORS(handleIndex))
	http.HandleFunc("/wal", withCORS(handleWAL))
	http.HandleFunc("/health", withCORS(handleHealth))

	fmt.Printf("🌐 GoDB HTTP Server running on %s\n", addr)
	fmt.Println("📖 Endpoints:")
	fmt.Println("   POST   /tables              - Create table")
	fmt.Println("   GET    /tables              - List tables")
	fmt.Println("   POST   /tables/{name}/records      - Insert record")
	fmt.Println("   GET    /tables/{name}/records/{id} - Get record")
	fmt.Println("   PUT    /tables/{name}/records/{id} - Update record")
	fmt.Println("   DELETE /tables/{name}/records/{id} - Delete record")
	fmt.Println("   GET    /tables/{name}/records      - Scan all records")
	fmt.Println("   POST   /query               - Query with filter/sort/limit")
	fmt.Println("   POST   /index               - Create B-tree index")
	fmt.Println("   GET    /wal                 - View WAL log")

	if err := http.ListenAndServe(addr, nil); err != nil {
		panic(err)
	}
}

func withCORS(h http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusOK)
            return
        }
        h(w, r)
    }
}

// handleHealth - GET /health
func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, response{Success: true, Data: map[string]string{
		"status":  "ok",
		"engine":  "GoDB v1.0",
		"storage": "disk-persisted JSON",
		"index":   "B-tree (in-memory)",
		"wal":     "append-only log",
	}})
}

// handleTables - GET/POST /tables
func handleTables(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
			writeJSON(w, 400, response{Success: false, Error: "missing table name"})
			return
		}
		if err := store.CreateTable(body.Name); err != nil {
			writeJSON(w, 400, response{Success: false, Error: err.Error()})
			return
		}
		walLog.Log("CREATE_TABLE", body.Name, "", nil)
		writeJSON(w, 201, response{Success: true, Data: fmt.Sprintf("table '%s' created", body.Name)})

	case http.MethodGet:
		writeJSON(w, 200, response{Success: true, Data: store.ListTables()})

	default:
		writeJSON(w, 405, response{Success: false, Error: "method not allowed"})
	}
}

// handleTableOps - /tables/{name}/records[/{id}]
func handleTableOps(w http.ResponseWriter, r *http.Request) {
	// Parse: /tables/{name}/records[/{id}]
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	// parts[0]=tables, parts[1]=name, parts[2]=records, parts[3]=id(optional)

	if len(parts) < 3 {
		writeJSON(w, 400, response{Success: false, Error: "invalid path"})
		return
	}

	tableName := parts[1]
	var recordID string
	if len(parts) >= 4 {
		recordID = parts[3]
	}

	switch r.Method {
	case http.MethodPost: // INSERT
		var body struct {
			ID   string                 `json:"id"`
			Data map[string]interface{} `json:"data"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, 400, response{Success: false, Error: "invalid JSON"})
			return
		}
		if err := store.Insert(tableName, body.ID, body.Data); err != nil {
			writeJSON(w, 400, response{Success: false, Error: err.Error()})
			return
		}
		// Update indexes
		for field, val := range body.Data {
			idxMgr.Insert(tableName, field, fmt.Sprintf("%v", val), body.ID)
		}
		walLog.Log("INSERT", tableName, body.ID, body.Data)
		writeJSON(w, 201, response{Success: true, Data: fmt.Sprintf("record '%s' inserted", body.ID)})

	case http.MethodGet:
		if recordID != "" { // Get by ID
			record, err := store.Get(tableName, recordID)
			if err != nil {
				writeJSON(w, 404, response{Success: false, Error: err.Error()})
				return
			}
			writeJSON(w, 200, response{Success: true, Data: record})
		} else { // Scan all
			records, err := store.Scan(tableName)
			if err != nil {
				writeJSON(w, 404, response{Success: false, Error: err.Error()})
				return
			}
			writeJSON(w, 200, response{Success: true, Data: records})
		}

	case http.MethodPut: // UPDATE
		if recordID == "" {
			writeJSON(w, 400, response{Success: false, Error: "record ID required"})
			return
		}
		var body struct {
			Data map[string]interface{} `json:"data"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, 400, response{Success: false, Error: "invalid JSON"})
			return
		}
		if err := store.Update(tableName, recordID, body.Data); err != nil {
			writeJSON(w, 400, response{Success: false, Error: err.Error()})
			return
		}
		walLog.Log("UPDATE", tableName, recordID, body.Data)
		writeJSON(w, 200, response{Success: true, Data: fmt.Sprintf("record '%s' updated", recordID)})

	case http.MethodDelete: // DELETE
		if recordID == "" {
			writeJSON(w, 400, response{Success: false, Error: "record ID required"})
			return
		}
		if err := store.Delete(tableName, recordID); err != nil {
			writeJSON(w, 404, response{Success: false, Error: err.Error()})
			return
		}
		walLog.Log("DELETE", tableName, recordID, nil)
		writeJSON(w, 200, response{Success: true, Data: fmt.Sprintf("record '%s' deleted", recordID)})

	default:
		writeJSON(w, 405, response{Success: false, Error: "method not allowed"})
	}
}

// handleQuery - POST /query
func handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, response{Success: false, Error: "method not allowed"})
		return
	}

	var req query.QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, response{Success: false, Error: "invalid query JSON"})
		return
	}

	result, err := qEngine.Execute(req)
	if err != nil {
		writeJSON(w, 400, response{Success: false, Error: err.Error()})
		return
	}
	writeJSON(w, 200, response{Success: true, Data: result})
}

// handleIndex - POST /index
func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, response{Success: false, Error: "method not allowed"})
		return
	}

	var body struct {
		Table string `json:"table"`
		Field string `json:"field"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, response{Success: false, Error: "invalid JSON"})
		return
	}

	if err := idxMgr.CreateIndex(body.Table, body.Field); err != nil {
		writeJSON(w, 400, response{Success: false, Error: err.Error()})
		return
	}
	writeJSON(w, 201, response{Success: true, Data: fmt.Sprintf("B-tree index created on %s.%s", body.Table, body.Field)})
}

// handleWAL - GET /wal
func handleWAL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, 405, response{Success: false, Error: "method not allowed"})
		return
	}

	entries, err := walLog.ReadAll()
	if err != nil {
		writeJSON(w, 500, response{Success: false, Error: err.Error()})
		return
	}
	writeJSON(w, 200, response{Success: true, Data: entries})
}

// writeJSON sends a JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
