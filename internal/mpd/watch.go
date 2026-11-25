package mpd

import (
	"context"
	"errors"
	"slices"
	"time"

	"rampd/internal/music"
)

// Watch listens for changes on a separate MPD connection until ctx is
// cancelled or the server closes the connection.
func (c Client) Watch(ctx context.Context, notify func(music.Changes)) error {
	w, err := c.open(ctx)
	if err != nil {
		return err
	}
	defer w.Close()
	commands, err := w.query("commands")
	if err != nil {
		return err
	}
	if !slices.Contains(commands, field{key: "command", value: "idle"}) {
		return errors.New("MPD does not permit idle notifications")
	}
	// The dial and greeting are bounded. Idle itself may wait indefinitely.
	// Cancel closes the socket so the read cannot outlive the application.
	if err := w.SetDeadline(time.Time{}); err != nil {
		return err
	}
	notify(music.SessionReady)
	for ctx.Err() == nil {
		fields, err := w.query("idle", "player", "playlist", "options", "mixer", "database", "update", "stored_playlist")
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		var changes music.Changes
		var updateChanged bool
		for _, f := range fields {
			if f.key != "changed" {
				continue
			}
			switch f.value {
			case "player", "playlist", "options", "mixer":
				changes |= music.PlaybackChanged
			case "database":
				changes |= music.DatabaseChanged
			case "update":
				updateChanged = true
			case "stored_playlist":
				changes |= music.StoredPlaylistsChanged
			}
		}
		if updateChanged && changes&music.DatabaseChanged == 0 {
			// Some servers report only "update" on the first database scan.
			// Check status so a notification at scan start does not reload the library.
			deadline := time.Now().Add(c.Timeout)
			if err := w.SetDeadline(deadline); err != nil {
				return err
			}
			status, err := w.query("status")
			if err != nil {
				return err
			}
			// The next idle wait is not a request with a time limit.
			if err := w.SetDeadline(time.Time{}); err != nil {
				return err
			}
			if get(status, "updating_db") == "" {
				changes |= music.DatabaseChanged
			}
		}
		if changes != 0 {
			notify(changes)
		}
	}
	return ctx.Err()
}
