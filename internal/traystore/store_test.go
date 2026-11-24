package traystore

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"rampd/internal/music"
)

func TestTrayRoundTripPreservesOrderAndRepeats(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)
	path, err := Path()
	if err != nil || path != filepath.Join(dir, "rampd", "tray.json") {
		t.Fatalf("state path: %q, %v", path, err)
	}
	if empty, err := Load(path); len(empty) != 0 || err != nil {
		t.Fatalf("missing state should be empty: %v, %v", empty, err)
	}
	a := music.Track{URI: "one.flac", Title: "One", Duration: time.Minute}
	tracks := []music.Track{a, {URI: "two.flac"}, a}
	if err := Save(path, tracks); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil || !slices.Equal(got, tracks) {
		t.Fatalf("round trip: %+v, %v", got, err)
	}
	if err := Save(path, nil); err != nil {
		t.Fatal(err)
	}
	if empty, err := Load(path); len(empty) != 0 || err != nil {
		t.Fatalf("empty Tray did not replace old contents: %v, %v", empty, err)
	}
}

func TestLoadLeavesInvalidStateUntouched(t *testing.T) {
	for _, data := range []string{"{", `{"version":2,"tracks":[]}`, `{"version":1,"tracks":[{}]}`} {
		path := filepath.Join(t.TempDir(), "tray.json")
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Fatalf("accepted invalid state: %s", data)
		}
		if got, err := os.ReadFile(path); err != nil || string(got) != data {
			t.Fatalf("invalid state changed: %q, %v", got, err)
		}
	}
}
