package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestRandomRim_EmptySlice(t *testing.T) {
	got := RandomRim(nil)
	if got != "" {
		t.Fatalf("expected empty string for nil slice, got %q", got)
	}

	got = RandomRim([]string{})
	if got != "" {
		t.Fatalf("expected empty string for empty slice, got %q", got)
	}
}

func TestRandomRim_NonEmptySlice(t *testing.T) {
	choices := []string{"a", "b", "c"}

	got := RandomRim(choices)
	found := false
	for _, c := range choices {
		if got == c {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected result to be one of %v, got %q", choices, got)
	}
}

func TestNewStore_Defaults(t *testing.T) {
	s, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore returned error: %v", err)
	}

	if s.filePath != "messages.json" {
		t.Fatalf("expected default filePath %q, got %q", "messages.json", s.filePath)
	}

	if s.popMaxWait != 0 {
		t.Fatalf("expected default popMaxWait 0, got %v", s.popMaxWait)
	}

	if s.entries == nil {
		t.Fatalf("expected entries map to be initialized, got nil")
	}

	if len(s.entries) != 0 {
		t.Fatalf("expected empty entries map, got len=%d", len(s.entries))
	}
}

func TestWithFilePathOption(t *testing.T) {
	tmpDir := t.TempDir()
	customPath := filepath.Join(tmpDir, "custom.json")

	s, err := NewStore(WithFilePath(customPath))
	if err != nil {
		t.Fatalf("NewStore with WithFilePath returned error: %v", err)
	}

	if s.filePath != customPath {
		t.Fatalf("expected filePath %q, got %q", customPath, s.filePath)
	}
}

func TestWithPopMaxWaitOption(t *testing.T) {
	wait := 2 * time.Second

	s, err := NewStore(WithPopMaxWait(wait))
	if err != nil {
		t.Fatalf("NewStore with WithPopMaxWait returned error: %v", err)
	}

	if s.popMaxWait != wait {
		t.Fatalf("expected popMaxWait %v, got %v", wait, s.popMaxWait)
	}
}

func TestList_ReadsFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "store.json")

	// Prepare file content matching Lists expected format.
	wantEntries := map[EntryID]*Entry{
		"id-1": {
			ID:      "id-1",
			Author:  "author1",
			Content: "content1",
			Created: time.Now().Format(time.RFC3339),
		},
		"id-2": {
			ID:      "id-2",
			Author:  "author2",
			Content: "content2",
			Created: time.Now().Format(time.RFC3339),
		},
	}

	data, err := json.Marshal(wantEntries)
	if err != nil {
		t.Fatalf("failed to marshal entries: %v", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("failed to write test store file: %v", err)
	}

	s, err := NewStore(WithFilePath(path))
	if err != nil {
		t.Fatalf("NewStore returned error: %v", err)
	}

	got, err := s.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if !reflect.DeepEqual(got, wantEntries) {
		t.Fatalf("List mismatch.\nGot:  %#v\nWant: %#v", got, wantEntries)
	}
}

func TestSaveAndGet_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	// Change into the temp directory so that Save, which uses a relative
	// path and CreateTemp in the current directory, works with a simple
	// filename pattern (no path separators).
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir to temp dir failed: %v", err)
	}

	path := "store.json"

	// Initialize store file with an empty JSON map so List() succeeds.
	if err := os.WriteFile(path, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("failed to write initial empty store file: %v", err)
	}

	s, err := NewStore(WithFilePath(path))
	if err != nil {
		t.Fatalf("NewStore returned error: %v", err)
	}

	entry := &Entry{
		Author:    "tester",
		Content:   "hello world",
		IPAddr:    "127.0.0.1",
		UserAgent: "test-agent",
	}

	saved, err := s.Save(entry)
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	if saved.ID == "" {
		t.Fatalf("expected Save to set ID, got empty")
	}
	if saved.Created == "" {
		t.Fatalf("expected Save to set Created, got empty")
	}

	if saved.Author != entry.Author || saved.Content != entry.Content {
		t.Fatalf("saved entry content mismatch: got %#v, want author=%q content=%q",
			saved, entry.Author, entry.Content)
	}

	// Verify Get reads the same entry back from file.
	got, err := s.Get(saved.ID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}

	if !reflect.DeepEqual(saved, got) {
		t.Fatalf("Get mismatch.\nGot:  %#v\nWant: %#v", got, saved)
	}

	// Also verify that the in-memory queue contains the entry.
	q, err := s.Queue()
	if err != nil {
		t.Fatalf("Queue returned error: %v", err)
	}
	if len(q) != 1 {
		t.Fatalf("expected Queue length 1, got %d", len(q))
	}
	qe, ok := q[EntryID(saved.ID)]
	if !ok {
		t.Fatalf("expected Queue to contain entry with ID %q", saved.ID)
	}
	if qe.ID != saved.ID || qe.Author != saved.Author || qe.Content != saved.Content {
		t.Fatalf("Queue entry mismatch: got %#v, want ID=%q Author=%q Content=%q", qe, saved.ID, saved.Author, saved.Content)
	}
}

func TestGet_NotExist(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "store.json")

	// Empty JSON map
	if err := os.WriteFile(path, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("failed to write empty store file: %v", err)
	}

	s, err := NewStore(WithFilePath(path))
	if err != nil {
		t.Fatalf("NewStore returned error: %v", err)
	}

	_, err = s.Get("does-not-exist")
	if err == nil {
		t.Fatalf("expected error for non-existing id, got nil")
	}
	if !os.IsNotExist(err) {
		t.Fatalf("expected os.ErrNotExist, got %v", err)
	}
}

func TestQueue_ReturnsEntries(t *testing.T) {
	s := &Store{
		entries: map[EntryID]*Entry{
			"e1": {ID: "e1", Author: "a1"},
			"e2": {ID: "e2", Author: "a2"},
		},
	}

	got, err := s.Queue()
	if err != nil {
		t.Fatalf("Queue returned error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}

	if got["e1"].Author != "a1" || got["e2"].Author != "a2" {
		t.Fatalf("Queue returned incorrect entries: %#v", got)
	}
}

func TestPop_ReturnsDefaultWhenEmpty(t *testing.T) {
	s := &Store{
		entries:    map[EntryID]*Entry{},
		popMaxWait: 0,
	}

	got, err := s.Pop()
	if err != nil {
		t.Fatalf("Pop returned error: %v", err)
	}

	if got == nil {
		t.Fatalf("expected non-nil entry from Pop when empty")
	}

	if got.Author != "Tomten" {
		t.Fatalf("expected default Author %q, got %q", "Tomten", got.Author)
	}
	if got.IPAddr != "1270.0.0" {
		t.Fatalf("expected default IPAddr %q, got %q", "1270.0.0", got.IPAddr)
	}
	if got.Content == "" {
		t.Fatalf("expected default Content to be non-empty")
	}
}

func TestPop_RemovesAndReturnsEntry(t *testing.T) {
	entry := &Entry{ID: "id-1", Author: "tester"}

	s := &Store{
		entries: map[EntryID]*Entry{
			"key-1": entry,
		},
		popMaxWait: 0,
	}

	got, err := s.Pop()
	if err != nil {
		t.Fatalf("Pop returned error: %v", err)
	}

	if got != entry {
		t.Fatalf("expected Pop to return original entry, got %#v", got)
	}

	if len(s.entries) != 0 {
		t.Fatalf("expected entries to be empty after Pop, got len=%d", len(s.entries))
	}
}

func TestPop_RespectsBackoff(t *testing.T) {
	entry := &Entry{ID: "id-1", Author: "tester"}

	s := &Store{
		entries: map[EntryID]*Entry{
			"key-1": entry,
		},
		popMaxWait: time.Second,
		lastPop:    time.Now(),
		lastEntry:  &Entry{ID: "last", Author: "last-author"},
	}

	got, err := s.Pop()
	if err != nil {
		t.Fatalf("Pop returned error: %v", err)
	}

	// Since not enough time has passed, Pop should return lastEntry
	if got != s.lastEntry {
		t.Fatalf("expected Pop to return lastEntry due to backoff, got %#v", got)
	}

	// And the queue should remain untouched
	if len(s.entries) != 1 {
		t.Fatalf("expected entries to remain unchanged, got len=%d", len(s.entries))
	}
}
