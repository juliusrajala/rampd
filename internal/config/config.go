// Package config reads the optional player settings file.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type Settings struct {
	MPD struct {
		Host     string `toml:"host"`
		Password string `toml:"password"`
		Port     int    `toml:"port"`
		Timeout  string `toml:"timeout"`
	} `toml:"mpd"`
	UI struct {
		Artwork bool `toml:"artwork"`
	} `toml:"ui"`
	Dream struct {
		TrackCount int `toml:"track_count"`
	} `toml:"dream"`
	Keybindings map[string][]string `toml:"keybindings"`
	Theme       map[string]string   `toml:"theme"`
}

func Defaults() Settings {
	var s Settings
	s.MPD.Host, s.MPD.Port = "127.0.0.1", 6600
	s.MPD.Timeout = (5 * time.Second).String()
	s.UI.Artwork = true
	s.Dream.TrackCount = 25
	return s
}

// Load reads path or the default XDG file. Only the default file may be absent.
func Load(path string) (Settings, error) {
	settings := Defaults()
	optional := path == ""
	if optional {
		dir, err := os.UserConfigDir()
		if err != nil {
			return settings, nil
		}
		path = filepath.Join(dir, "rampd", "config.toml")
	}
	file, err := os.Open(path)
	if optional && errors.Is(err, os.ErrNotExist) {
		return settings, nil
	}
	if err != nil {
		return Settings{}, err
	}
	defer file.Close()
	if err := toml.NewDecoder(file).DisallowUnknownFields().Decode(&settings); err != nil {
		// Report the location, not the contents of a line that may hold a password.
		var decodeErr *toml.DecodeError
		if errors.As(err, &decodeErr) {
			line, column := decodeErr.Position()
			return Settings{}, fmt.Errorf("invalid TOML in %s at %d:%d", path, line, column)
		}
		return Settings{}, fmt.Errorf("cannot decode %s as TOML", path)
	}
	return settings, nil
}
