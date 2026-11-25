package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"

	"rampd/internal/keymap"
)

func TestOptionPrecedence(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	root := filepath.Join(dir, "rampd")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "config.toml")
	content := `[mpd]
host = "config-host"
port = 6601
password = "from-file"
timeout = "3s"

[ui]
artwork = false

[dream]
track_count = 12

[keybindings]
play_pause = ["u", "ctrl+p"]

[theme]
progress_played = "#123ABC"
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	env := map[string]string{"MPD_HOST": "from-host@env-host", "MPD_PORT": "6602", "MPD_PASSWORD": "from-env"}
	client, opts, err := parseOptions([]string{"-host", "flag-host", "-dream-count", "8"}, func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	})
	if err != nil {
		t.Fatal(err)
	}
	if client.Address != "flag-host:6602" || client.Password != "from-env" || client.Timeout != 3*time.Second {
		t.Fatal("CLI, environment, and file options did not merge correctly")
	}
	if opts.Artwork || opts.DreamCount != 8 || opts.Keys.Action("u") != keymap.PlayPause || opts.Keys.Action("p") != "" {
		t.Fatal("UI options did not merge correctly")
	}
	if opts.Styles == nil || opts.Styles.ProgressFilled.GetForeground() != lipgloss.Color("#123ABC") {
		t.Fatal("theme settings did not reach the app options")
	}
	_, _, err = parseOptions([]string{"-config", filepath.Join(root, "missing.toml")}, func(string) (string, bool) { return "", false })
	if err == nil {
		t.Fatal("explicit missing config should report an error")
	}
}

func TestConfigErrorDoesNotEchoPassword(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.toml")
	if err := os.WriteFile(path, []byte("[mpd]\npassword = secret-value\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, _, err := parseOptions([]string{"-config", path}, func(string) (string, bool) { return "", false })
	if err == nil || strings.Contains(err.Error(), "secret-value") {
		t.Fatal("expected a safe error for invalid settings")
	}
}
