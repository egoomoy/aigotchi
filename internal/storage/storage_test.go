package storage_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoomoy/aigotchi/internal/storage"
)

func TestNewStore_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".aigotchi")

	_, err := storage.NewStore(path)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("expected directory to be created")
	}
}

func TestStore_WriteAndReadJSON(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewStore(filepath.Join(dir, ".aigotchi"))

	type testData struct {
		Version int    `json:"version"`
		Name    string `json:"name"`
	}

	input := testData{Version: 1, Name: "Mochi"}
	err := store.WriteJSON("test.json", input)
	if err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	var output testData
	err = store.ReadJSON("test.json", &output)
	if err != nil {
		t.Fatalf("ReadJSON failed: %v", err)
	}

	if output.Name != "Mochi" || output.Version != 1 {
		t.Fatalf("expected {1, Mochi}, got {%d, %s}", output.Version, output.Name)
	}
}

func TestStore_AppendLine(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewStore(filepath.Join(dir, ".aigotchi"))

	err := store.AppendLine("events.jsonl", []byte(`{"tokens":100}`))
	if err != nil {
		t.Fatalf("AppendLine failed: %v", err)
	}
	err = store.AppendLine("events.jsonl", []byte(`{"tokens":200}`))
	if err != nil {
		t.Fatalf("AppendLine failed: %v", err)
	}

	lines, err := store.ReadLines("events.jsonl")
	if err != nil {
		t.Fatalf("ReadLines failed: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
}

func TestStore_Exists(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewStore(filepath.Join(dir, ".aigotchi"))

	if store.Exists("nonexistent.json") {
		t.Fatal("expected file to not exist")
	}

	store.WriteJSON("exists.json", map[string]int{"version": 1})
	if !store.Exists("exists.json") {
		t.Fatal("expected file to exist")
	}
}
