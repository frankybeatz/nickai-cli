package personality

import "testing"

func TestAllVibesNonEmpty(t *testing.T) {
	vibes := AllVibes()
	if len(vibes) == 0 {
		t.Fatal("AllVibes returned empty slice")
	}
	for _, v := range vibes {
		if v.ID == "" {
			t.Error("vibe has empty ID")
		}
		if v.Name == "" {
			t.Errorf("vibe %q has empty Name", v.ID)
		}
		if v.Prompt == "" {
			t.Errorf("vibe %q has empty Prompt", v.ID)
		}
		if len(v.Greetings) == 0 {
			t.Errorf("vibe %q has no Greetings", v.ID)
		}
		if v.Emoji == "" {
			t.Errorf("vibe %q has empty Emoji", v.ID)
		}
		if v.Tagline == "" {
			t.Errorf("vibe %q has empty Tagline", v.ID)
		}
	}
}

func TestGetVibeKnown(t *testing.T) {
	v := GetVibe("degen")
	if v == nil {
		t.Fatal("GetVibe(degen) returned nil")
	}
	if v.ID != "degen" {
		t.Errorf("expected degen, got %q", v.ID)
	}
}

func TestGetVibeUnknownFallback(t *testing.T) {
	v := GetVibe("nonexistent")
	if v == nil {
		t.Fatal("GetVibe(nonexistent) returned nil")
	}
	if v.ID != DefaultVibeID {
		t.Errorf("expected fallback to %q, got %q", DefaultVibeID, v.ID)
	}
}

func TestNoDuplicateIDs(t *testing.T) {
	seen := make(map[string]bool)
	for _, v := range AllVibes() {
		if seen[v.ID] {
			t.Errorf("duplicate vibe ID: %q", v.ID)
		}
		seen[v.ID] = true
	}
}

func TestAllVibesReachable(t *testing.T) {
	for _, v := range AllVibes() {
		got := GetVibe(v.ID)
		if got.ID != v.ID {
			t.Errorf("GetVibe(%q) returned %q", v.ID, got.ID)
		}
	}
}
