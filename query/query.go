package query

import (
	"fmt"
	"godb/storage"
	"sort"
	"strings"
)

// QueryRequest defines a query
type QueryRequest struct {
	Table   string            `json:"table"`
	Filter  map[string]string `json:"filter,omitempty"`  // field -> value (exact match)
	OrderBy string            `json:"order_by,omitempty"` // field name
	Limit   int               `json:"limit,omitempty"`
	Fields  []string          `json:"fields,omitempty"` // projection
}

// QueryResult holds the result of a query
type QueryResult struct {
	Records []map[string]interface{} `json:"records"`
	Total   int                      `json:"total"`
	Message string                   `json:"message,omitempty"`
}

// Engine processes queries against storage
type Engine struct {
	store *storage.StorageEngine
}

// NewEngine creates a query engine
func NewEngine(store *storage.StorageEngine) *Engine {
	return &Engine{store: store}
}

// Execute runs a query and returns results
func (e *Engine) Execute(req QueryRequest) (*QueryResult, error) {
	// 1. Full table scan
	records, err := e.store.Scan(req.Table)
	if err != nil {
		return nil, fmt.Errorf("scan failed: %w", err)
	}

	// 2. Filter (WHERE equivalent)
	filtered := []storage.Record{}
	for _, r := range records {
		if matchesFilter(r, req.Filter) {
			filtered = append(filtered, r)
		}
	}

	// 3. Sort (ORDER BY equivalent)
	if req.OrderBy != "" {
		sort.Slice(filtered, func(i, j int) bool {
			vi := fmt.Sprintf("%v", filtered[i].Data[req.OrderBy])
			vj := fmt.Sprintf("%v", filtered[j].Data[req.OrderBy])
			return strings.Compare(vi, vj) < 0
		})
	}

	// 4. Limit
	if req.Limit > 0 && req.Limit < len(filtered) {
		filtered = filtered[:req.Limit]
	}

	// 5. Projection (SELECT fields)
	result := make([]map[string]interface{}, 0, len(filtered))
	for _, r := range filtered {
		row := map[string]interface{}{"id": r.ID}
		if len(req.Fields) == 0 {
			for k, v := range r.Data {
				row[k] = v
			}
		} else {
			for _, f := range req.Fields {
				if v, ok := r.Data[f]; ok {
					row[f] = v
				}
			}
		}
		result = append(result, row)
	}

	return &QueryResult{
		Records: result,
		Total:   len(result),
		Message: fmt.Sprintf("Query returned %d record(s)", len(result)),
	}, nil
}

// matchesFilter checks if a record matches all filter conditions
func matchesFilter(r storage.Record, filter map[string]string) bool {
	for field, value := range filter {
		if field == "id" {
			if r.ID != value {
				return false
			}
			continue
		}
		v, ok := r.Data[field]
		if !ok {
			return false
		}
		if fmt.Sprintf("%v", v) != value {
			return false
		}
	}
	return true
}
