// Package traystore saves the Tray independently of MPD.
package traystore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"rampd/internal/music"
)

type snapshot struct {
	Version int           `json:"version"`
	Tracks  []music.Track `json:"tracks"`
}

// Path returns the location of the saved Tray.
func Path() (string, error) {
	dir := os.Getenv("XDG_STATE_HOME")
	if !filepath.IsAbs(dir) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(dir, "rampd", "tray.json"), nil
}

// Load returns an empty Tray when no file exists. It never modifies the file.
func Load(path string) ([]music.Track, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var saved snapshot
	if err := json.Unmarshal(data, &saved); err != nil {
		return nil, fmt.Errorf("read Tray %q: %w", path, err)
	}
	if saved.Version != 1 {
		return nil, fmt.Errorf("unsupported Tray format in %q", path)
	}
	for _, track := range saved.Tracks {
		if track.URI == "" {
			return nil, fmt.Errorf("missing track URI in Tray %q", path)
		}
	}
	return saved.Tracks, nil
}

// Save replaces the file only after writing a complete snapshot.
func Save(path string, tracks []music.Track) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, ".tray-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if err := json.NewEncoder(file).Encode(snapshot{Version: 1, Tracks: tracks}); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
