# GoDB — Database Engine From Scratch (Go)

A lightweight database engine built from scratch in Go, featuring:
- 📦 **Storage Engine** — disk-persisted JSON storage with full CRUD
- 🌳 **B-tree Index** — fast field-level lookups
- 📝 **Write-Ahead Log (WAL)** — crash recovery support
- 🔍 **Query Engine** — filter, sort, limit, and projection
- 🌐 **REST API** — full HTTP interface via `net/http`

---

## 🛠️ Setup & Run

### Prerequisites
- Go 1.21+ installed → https://go.dev/dl/

### Step 1 — Clone / Copy the project
```bash
git clone https://github.com/YOUR_USERNAME/godb.git
cd godb
```

### Step 2 — Run the server
```bash
go run main.go
```
You should see:
```
🚀 GoDB - Database Engine Starting...
🌐 GoDB HTTP Server running on :8080
```

---

## 📡 API Usage (via curl or Postman)

### ✅ Health Check
```bash
curl http://localhost:8080/health
```

### 📋 Create a Table
```bash
curl -X POST http://localhost:8080/tables \
  -H "Content-Type: application/json" \
  -d '{"name": "users"}'
```

### ➕ Insert a Record
```bash
curl -X POST http://localhost:8080/tables/users/records \
  -H "Content-Type: application/json" \
  -d '{
    "id": "u1",
    "data": {"name": "Ujjwal", "age": "21", "city": "Bijnor"}
  }'
```

### 🔍 Get a Record by ID
```bash
curl http://localhost:8080/tables/users/records/u1
```

### 📄 Scan All Records
```bash
curl http://localhost:8080/tables/users/records
```

### ✏️ Update a Record
```bash
curl -X PUT http://localhost:8080/tables/users/records/u1 \
  -H "Content-Type: application/json" \
  -d '{"data": {"name": "Ujjwal Kumar", "age": "22", "city": "Agartala"}}'
```

### ❌ Delete a Record
```bash
curl -X DELETE http://localhost:8080/tables/users/records/u1
```

### 🔎 Query with Filter + Sort + Limit
```bash
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{
    "table": "users",
    "filter": {"city": "Bijnor"},
    "order_by": "name",
    "limit": 10,
    "fields": ["name", "city"]
  }'
```

### 🌳 Create a B-tree Index
```bash
curl -X POST http://localhost:8080/index \
  -H "Content-Type: application/json" \
  -d '{"table": "users", "field": "city"}'
```

### 📝 View WAL (Write-Ahead Log)
```bash
curl http://localhost:8080/wal
```

---

## 🏗️ Architecture

```
godb/
├── main.go           # Entry point
├── server/
│   └── server.go     # HTTP REST API layer
├── storage/
│   └── storage.go    # Storage engine (disk persistence, CRUD)
├── index/
│   └── btree.go      # B-tree index for fast lookups
├── query/
│   └── query.go      # Query engine (filter, sort, limit, projection)
├── wal/
│   └── wal.go        # Write-Ahead Log for crash recovery
└── data/             # Auto-created: persisted table files + WAL
```

---

## 💡 Key Concepts (Interview Talking Points)

| Concept | Implementation |
|---------|----------------|
| **Storage Engine** | JSON files per table, loaded on startup |
| **B-tree Index** | In-memory sorted key structure for O(log n) lookup |
| **WAL** | Append-only log written before every mutation |
| **Concurrency** | `sync.RWMutex` on table and engine level |
| **Query Engine** | Filter → Sort → Limit → Project pipeline |
| **REST API** | Pure `net/http`, no external frameworks |

---

## 📌 Resume Bullet Points

- Built a database engine from scratch in Go with a custom storage engine, B-tree indexing, Write-Ahead Log (WAL), and a query engine supporting filter, sort, and projection
- Implemented disk-persistent storage with concurrent read/write safety using `sync.RWMutex`, supporting full CRUD operations via a REST API
- Designed a WAL for crash recovery and a B-tree index layer for O(log n) field lookups, exposed via a pure `net/http` HTTP server
