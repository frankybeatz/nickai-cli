package credential

import "testing"

func TestKeyringGetSetDelete(t *testing.T) {
	if !KeyringAvailable() {
		t.Skip("OS keyring not available in this environment")
	}

	const testKey = "__nickai_test_key__"
	const testVal = "sk-test-12345"

	// Set a value.
	if ok := KeyringSet(testKey, testVal); !ok {
		t.Fatal("KeyringSet returned false")
	}

	// Get it back.
	got, ok := KeyringGet(testKey)
	if !ok {
		t.Fatal("KeyringGet returned false after Set")
	}
	if got != testVal {
		t.Errorf("KeyringGet = %q, want %q", got, testVal)
	}

	// Delete it.
	if ok := KeyringDelete(testKey); !ok {
		t.Fatal("KeyringDelete returned false")
	}

	// Verify it's gone.
	if _, ok := KeyringGet(testKey); ok {
		t.Error("KeyringGet returned true after Delete")
	}
}

func TestKeyringGetMissing(t *testing.T) {
	if !KeyringAvailable() {
		t.Skip("OS keyring not available in this environment")
	}

	_, ok := KeyringGet("__nickai_nonexistent_key__")
	if ok {
		t.Error("KeyringGet returned true for non-existent key")
	}
}

func TestKeyringAvailableConsistent(t *testing.T) {
	// The result should be consistent across calls.
	a := KeyringAvailable()
	b := KeyringAvailable()
	if a != b {
		t.Error("KeyringAvailable returned different values on consecutive calls")
	}
}
