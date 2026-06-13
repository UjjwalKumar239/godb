package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// Record represents a single row in a table
type Record struct {
	ID   string                 `json:"id"`
	Data map[string]interface{} `json:"data"`
}

// Table holds records in memory and persists to disk
type Table struct {
	Name    string             `json:"name"`
	Records map[string]Record  `json:"records"`
	mu      sync.RWMutex
}

// StorageEngine manages multiple tables
type StorageEngine struct {
	tables  map[string]*Table
	dataDir string
	mu      sync.RWMutex
}

// NewStorageEngine initializes the engine
func NewStorageEngine(dataDir string) (*StorageEngine, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data dir: %w", err)
	}
	engine := &StorageEngine{
		tables:  make(map[string]*Table),
		dataDir: dataDir,
	}
	// Load existing tables from disk
	engine.loadFromDisk()
	return engine, nil
}

// CreateTable creates a new table
func (e *StorageEngine) CreateTable(name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.tables[name]; exists {
		return fmt.Errorf("table '%s' already exists", name)
	}
	e.tables[name] = &Table{
		Name:    name,
		Records: make(map[string]Record),
	}
	return e.persistTable(name)
}

// Insert adds a record to a table
func (e *StorageEngine) Insert(tableName, id string, data map[string]interface{}) error {
	e.mu.RLock()
	table, exists := e.tables[tableName]
	e.mu.RUnlock()

	if !exists {
		return fmt.Errorf("table '%s' not found", tableName)
	}

	table.mu.Lock()
	defer table.mu.Unlock()

	if _, exists := table.Records[id]; exists {
		return fmt.Errorf("record with id '%s' already exists", id)
	}

	table.Records[id] = Record{ID: id, Data: data}
	return e.persistTable(tableName)
}

// Get retrieves a record by ID
func (e *StorageEngine) Get(tableName, id string) (*Record, error) {
	e.mu.RLock()
	table, exists := e.tables[tableName]
	e.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("table '%s' not found", tableName)
	}

	table.mu.RLock()
	defer table.mu.RUnlock()

	record, exists := table.Records[id]
	if !exists {
		return nil, fmt.Errorf("record '%s' not found", id)
	}
	return &record, nil
}

// Update modifies an existing record
func (e *StorageEngine) Update(tableName, id string, data map[string]interface{}) error {
	e.mu.RLock()
	table, exists := e.tables[tableName]
	e.mu.RUnlock()

	if !exists {
		return fmt.Errorf("table '%s' not found", tableName)
	}

	table.mu.Lock()
	defer table.mu.Unlock()

	if _, exists := table.Records[id]; !exists {
		return fmt.Errorf("record '%s' not found", id)
	}

	table.Records[id] = Record{ID: id, Data: data}
	return e.persistTable(tableName)
}

// Delete removes a record
func (e *StorageEngine) Delete(tableName, id string) error {
	e.mu.RLock()
	table, exists := e.tables[tableName]
	e.mu.RUnlock()

	if !exists {
		return fmt.Errorf("table '%s' not found", tableName)
	}

	table.mu.Lock()
	defer table.mu.Unlock()

	if _, exists := table.Records[id]; !exists {
		return fmt.Errorf("record '%s' not found", id)
	}

	delete(table.Records, id)
	return e.persistTable(tableName)
}

// Scan returns all records in a table
func (e *StorageEngine) Scan(tableName string) ([]Record, error) {
	e.mu.RLock()
	table, exists := e.tables[tableName]
	e.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("table '%s' not found", tableName)
	}

	table.mu.RLock()
	defer table.mu.RUnlock()

	records := make([]Record, 0, len(table.Records))
	for _, r := range table.Records {
		records = append(records, r)
	}
	return records, nil
}

// ListTables returns all table names
func (e *StorageEngine) ListTables() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	names := make([]string, 0, len(e.tables))
	for name := range e.tables {
		names = append(names, name)
	}
	return names
}

// persistTable saves a table to disk as JSON
func (e *StorageEngine) persistTable(name string) error {
	table := e.tables[name]
	filePath := fmt.Sprintf("%s/%s.json", e.dataDir, name)

	data, err := json.MarshalIndent(table, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal table: %w", err)
	}
	return os.WriteFile(filePath, data, 0644)
}

// loadFromDisk loads all tables from disk on startup
func (e *StorageEngine) loadFromDisk() {
	entries, err := os.ReadDir(e.dataDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filePath := fmt.Sprintf("%s/%s", e.dataDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		var table Table
		if err := json.Unmarshal(data, &table); err != nil {
			continue
		}
		e.tables[table.Name] = &table
		fmt.Printf("✅ Loaded table: %s (%d records)\n", table.Name, len(table.Records))
	}
}
