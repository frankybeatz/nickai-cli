package safefile

import (
	"os"
	"sync"
)

// AtomicWrite writes data to path atomically using write-to-temp-then-rename.
// This prevents corruption if the process crashes mid-write.
func AtomicWrite(path string, data []byte, perm os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// mu holds per-path mutexes for store files.
var mu sync.Map

// Lock returns a mutex for the given file path, creating one if needed.
// Use this to guard Load-Modify-Save cycles and prevent TOCTOU races.
func Lock(path string) *sync.Mutex {
	val, _ := mu.LoadOrStore(path, &sync.Mutex{})
	return val.(*sync.Mutex)
}
