package mpd

import (
	"errors"
	"fmt"
	"slices"
	"strconv"

	"rampd/internal/music"
)

func replaceAndPlay(w *wire, tracks []music.Track) error {
	if len(tracks) == 0 {
		return nil
	}
	previous, err := readQueue(w)
	if err != nil {
		return err
	}

	adds := make([]command, 0, len(tracks))
	for _, track := range tracks {
		adds = append(adds, command{"addid", []string{track.URI}})
	}
	fields, err := w.queryList(adds)
	var staged []string
	for _, f := range fields {
		if f.key == "Id" {
			staged = append(staged, f.value)
		}
	}
	if err != nil {
		var ack ackError
		if errors.As(err, &ack) {
			// An ACK ends the response. The connection is still usable and
			// we can undo the earlier adds without touching the live entries.
			var rollback []command
			for _, id := range staged {
				rollback = append(rollback, command{"deleteid", []string{id}})
			}
			if rollbackErr := w.runList(rollback); rollbackErr != nil {
				return errors.Join(err, fmt.Errorf("remove staged Dream tracks: %w", rollbackErr))
			}
		}
		return err
	}
	if len(staged) != len(tracks) {
		return errors.New("MPD did not return all staged track IDs; original queue entries were not removed")
	}

	commands := make([]command, 0, len(previous)+2)
	if len(previous) > 0 {
		commands = append(commands, command{"stop", nil})
	}
	for _, entry := range previous {
		commands = append(commands, command{"deleteid", []string{strconv.Itoa(entry.ID)}})
	}
	commands = append(commands, command{"playid", []string{staged[0]}})
	return w.runList(commands)
}

func replaceQueue(w *wire, tracks []music.Track, plan *music.QueueReplacement) error {
	if plan == nil || len(tracks) == 0 || plan.KeepAt < -1 || plan.KeepAt >= len(tracks) ||
		(plan.KeepAt >= 0 && plan.CurrentID < 0) {
		return errors.New("invalid queue replacement")
	}
	entries, err := readQueue(w)
	if err != nil {
		return err
	}
	status, err := readStatus(w)
	if err != nil {
		return err
	}
	// Check the queue and playback again after the confirmation. MPD has no
	// atomic compare-and-update operation, but the writes form one command list.
	if status.QueueVersion != plan.Version || status.PlayingID() != plan.CurrentID ||
		!slices.EqualFunc(entries, plan.Before, func(a, b music.Entry) bool { return a.ID == b.ID && a.URI == b.URI }) {
		return errors.New("queue or current track changed; review and try again")
	}

	keepID := -1
	if plan.KeepAt >= 0 {
		current := slices.IndexFunc(entries, func(entry music.Entry) bool { return entry.ID == plan.CurrentID })
		if current < 0 || entries[current].URI != tracks[plan.KeepAt].URI {
			return errors.New("current track does not match the replacement")
		}
		keepID = plan.CurrentID
	}
	commands := make([]command, 0, len(tracks)+len(entries)+2)
	// Add first. A failed add may leave extra entries, but cannot remove the
	// original queue or interrupt playback.
	for i, track := range tracks {
		if i != plan.KeepAt {
			commands = append(commands, command{"addid", []string{track.URI}})
		}
	}
	if plan.CurrentID >= 0 && keepID < 0 {
		commands = append(commands, command{"stop", nil})
	}
	for _, entry := range entries {
		if entry.ID != keepID {
			commands = append(commands, command{"deleteid", []string{strconv.Itoa(entry.ID)}})
		}
	}
	if plan.KeepAt > 0 {
		commands = append(commands, command{"moveid", []string{strconv.Itoa(keepID), strconv.Itoa(plan.KeepAt)}})
	}
	return w.runList(commands)
}
