package safefile

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")

	data := []byte(`{"key": "value"}`)
	if err := AtomicWrite(path, data, 0600); err != nil {
		t.Fatalf("AtomicWrite failed: %v", err)
	}

	// File should exist with correct content.
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("content = %q, want %q", string(got), string(data))
	}

	// Temp file should not exist.
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Error("temp file should not exist after atomic write")
	}

	// Permissions should be 0600.
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Errorf("permissions = %o, want 0600", info.Mode().Perm())
	}
}

func TestAtomicWriteOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")

	// Write initial content.
	AtomicWrite(path, []byte("old"), 0600)

	// Overwrite.
	if err := AtomicWrite(path, []byte("new"), 0600); err != nil {
		t.Fatalf("AtomicWrite overwrite failed: %v", err)
	}

	got, _ := os.ReadFile(path)
	if string(got) != "new" {
		t.Errorf("content = %q, want 'new'", string(got))
	}
}

func TestLock(t *testing.T) {
	// Same path should return the same mutex.
	mu1 := Lock("/tmp/test.json")
	mu2 := Lock("/tmp/test.json")
	if mu1 != mu2 {
		t.Error("Lock should return the same mutex for the same path")
	}

	// Different paths should return different mutexes.
	mu3 := Lock("/tmp/other.json")
	if mu1 == mu3 {
		t.Error("Lock should return different mutexes for different paths")
	}
}

func TestLockConcurrency(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "concurrent.json")

	var wg sync.WaitGroup
	counter := 0

	// 100 goroutines incrementing a counter protected by Lock.
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu := Lock(path)
			mu.Lock()
			defer mu.Unlock()
			counter++
		}()
	}
	wg.Wait()

	if counter != 100 {
		t.Errorf("counter = %d, want 100 (race condition)", counter)
	}
}
