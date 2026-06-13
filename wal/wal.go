package wal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// LogEntry represents a single WAL entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Operation string                 `json:"operation"` // INSERT, UPDATE, DELETE, CREATE_TABLE
	Table     string                 `json:"table"`
	RecordID  string                 `json:"record_id,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// WAL is the Write-Ahead Log
type WAL struct {
	file    *os.File
	writer  *bufio.Writer
	mu      sync.Mutex
	logPath string
}

// NewWAL creates or opens the WAL file
func NewWAL(logPath string) (*WAL, error) {
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open WAL: %w", err)
	}
	return &WAL{
		file:    file,
		writer:  bufio.NewWriter(file),
		logPath: logPath,
	}, nil
}

// Log writes an entry to the WAL
func (w *WAL) Log(op, table, recordID string, data map[string]interface{}) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	entry := LogEntry{
		Timestamp: time.Now(),
		Operation: op,
		Table:     table,
		RecordID:  recordID,
		Data:      data,
	}

	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	if _, err := w.writer.Write(append(line, '\n')); err != nil {
		return err
	}
	return w.writer.Flush()
}

// ReadAll returns all log entries (used for crash recovery)
func (w *WAL) ReadAll() ([]LogEntry, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	file, err := os.Open(w.logPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []LogEntry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var entry LogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}
	return entries, scanner.Err()
}

// Close closes the WAL file
func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.writer.Flush()
	return w.file.Close()
}
