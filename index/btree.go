package index

import (
	"fmt"
	"sort"
	"sync"
)

// BTreeNode represents a node in the B-tree
type BTreeNode struct {
	keys     []string
	values   [][]string // list of record IDs per key
	children []*BTreeNode
	isLeaf   bool
}

// BTreeIndex is a simple in-memory B-tree index on a field
type BTreeIndex struct {
	root    *BTreeNode
	field   string
	order   int // max keys per node
	mu      sync.RWMutex
	// flat map for fast exact lookup
	lookup  map[string][]string
}

// IndexManager manages indexes for all tables
type IndexManager struct {
	indexes map[string]map[string]*BTreeIndex // table -> field -> index
	mu      sync.RWMutex
}

// NewIndexManager creates a new index manager
func NewIndexManager() *IndexManager {
	return &IndexManager{
		indexes: make(map[string]map[string]*BTreeIndex),
	}
}

// CreateIndex creates a B-tree index on a table field
func (m *IndexManager) CreateIndex(table, field string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.indexes[table]; !exists {
		m.indexes[table] = make(map[string]*BTreeIndex)
	}
	if _, exists := m.indexes[table][field]; exists {
		return fmt.Errorf("index on '%s.%s' already exists", table, field)
	}
	m.indexes[table][field] = &BTreeIndex{
		field:  field,
		order:  4,
		lookup: make(map[string][]string),
		root:   &BTreeNode{isLeaf: true},
	}
	fmt.Printf("🌳 Created B-tree index on %s.%s\n", table, field)
	return nil
}

// Insert adds a key->recordID mapping to the index
func (m *IndexManager) Insert(table, field, key, recordID string) {
	m.mu.RLock()
	idx, exists := m.getIndex(table, field)
	m.mu.RUnlock()

	if !exists {
		return
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.lookup[key] = append(idx.lookup[key], recordID)

	// Insert into B-tree node (simplified: sorted keys in root leaf)
	insertSorted(idx.root, key)
}

// Search finds all record IDs matching a key
func (m *IndexManager) Search(table, field, key string) ([]string, bool) {
	m.mu.RLock()
	idx, exists := m.getIndex(table, field)
	m.mu.RUnlock()

	if !exists {
		return nil, false
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	ids, found := idx.lookup[key]
	return ids, found
}

// Delete removes a record from the index
func (m *IndexManager) Delete(table, field, key, recordID string) {
	m.mu.RLock()
	idx, exists := m.getIndex(table, field)
	m.mu.RUnlock()

	if !exists {
		return
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	ids := idx.lookup[key]
	newIDs := ids[:0]
	for _, id := range ids {
		if id != recordID {
			newIDs = append(newIDs, id)
		}
	}
	if len(newIDs) == 0 {
		delete(idx.lookup, key)
	} else {
		idx.lookup[key] = newIDs
	}
}

// ListIndexes returns all indexes for a table
func (m *IndexManager) ListIndexes(table string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	fields := []string{}
	if tableIndexes, exists := m.indexes[table]; exists {
		for field := range tableIndexes {
			fields = append(fields, field)
		}
	}
	return fields
}

func (m *IndexManager) getIndex(table, field string) (*BTreeIndex, bool) {
	if tableIndexes, exists := m.indexes[table]; exists {
		if idx, exists := tableIndexes[field]; exists {
			return idx, true
		}
	}
	return nil, false
}

// insertSorted inserts a key into a leaf node maintaining sorted order
func insertSorted(node *BTreeNode, key string) {
	i := sort.SearchStrings(node.keys, key)
	if i < len(node.keys) && node.keys[i] == key {
		return // already exists
	}
	node.keys = append(node.keys, "")
	copy(node.keys[i+1:], node.keys[i:])
	node.keys[i] = key
}
