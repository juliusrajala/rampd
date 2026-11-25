package mpd

import (
	"context"
	"errors"
	"strconv"
	"time"

	"rampd/internal/music"
)

func (c Client) Snapshot(ctx context.Context, knownQueue string) (music.Snapshot, error) {
	w, err := c.open(ctx)
	if err != nil {
		return music.Snapshot{}, err
	}
	defer w.Close()

	status, err := readStatus(w)
	if err != nil {
		return music.Snapshot{}, err
	}
	snapshot := music.Snapshot{Status: status}
	if knownQueue == "" || knownQueue != status.QueueVersion {
		snapshot.Queue, err = readQueue(w)
		snapshot.QueueChanged = true
	}
	return snapshot, err
}

func (c Client) Library(ctx context.Context) ([]music.Track, error) {
	w, err := c.open(ctx)
	if err != nil {
		return nil, err
	}
	defer w.Close()
	fields, err := w.query("listallinfo")
	if err != nil {
		return nil, err
	}
	var tracks []music.Track
	eachTrack(fields, func(group []field) {
		tracks = append(tracks, track(group))
	})
	return tracks, nil
}

func (c Client) Apply(ctx context.Context, a music.Action) error {
	w, err := c.open(ctx)
	if err != nil {
		return err
	}
	defer w.Close()

	id := strconv.Itoa(a.ID)
	switch a.Operation {
	case music.Toggle, music.Random, music.Repeat:
		status, err := readStatus(w)
		if err != nil {
			return err
		}
		switch a.Operation {
		case music.Toggle:
			switch status.State {
			case "play":
				return w.run("pause", "1")
			case "pause":
				return w.run("pause", "0")
			default:
				return w.run("play")
			}
		case music.Random:
			return w.run("random", bit(!status.Random))
		default:
			return w.run("repeat", bit(!status.Repeat))
		}
	case music.Next:
		return w.run("next")
	case music.Previous:
		return w.run("previous")
	case music.Seek:
		seconds := strconv.Itoa(a.Delta)
		if a.Delta >= 0 {
			seconds = "+" + seconds
		}
		return w.run("seekcur", seconds)
	case music.ReplaceAndPlay:
		return replaceAndPlay(w, a.Tracks)
	case music.Append, music.AppendAndPlay:
		if len(a.Tracks) == 0 {
			return nil
		}
		if a.Operation != music.Append {
			fields, err := w.query("addid", a.Tracks[0].URI)
			if err != nil {
				return err
			}
			id = get(fields, "Id")
			if id == "" {
				return errors.New("MPD did not return an ID for the added track")
			}
			a.Tracks = a.Tracks[1:]
		}
		commands := make([]command, 0, len(a.Tracks)+1)
		for _, track := range a.Tracks {
			commands = append(commands, command{"add", []string{track.URI}})
		}
		if a.Operation != music.Append {
			commands = append(commands, command{"playid", []string{id}})
		}
		return w.runList(commands)
	case music.ReplaceQueue:
		return replaceQueue(w, a.Tracks, a.Replacement)
	case music.Play:
		return w.run("playid", id)
	case music.Remove:
		return w.run("deleteid", id)
	case music.Move:
		entries, err := readQueue(w)
		if err != nil {
			return err
		}
		for pos, entry := range entries {
			if entry.ID == a.ID {
				target := pos + a.Delta
				if target < 0 || target >= len(entries) {
					return nil
				}
				// Resolve the destination from the live queue. The source is always an ID.
				return w.run("moveid", id, strconv.Itoa(target))
			}
		}
		return errors.New("the selected track is no longer in the queue")
	case music.Clear:
		return w.run("clear")
	case music.Shuffle:
		return w.run("shuffle")
	case music.SaveQueue:
		if err := checkPlaylistName(a.Name); err != nil {
			return err
		}
		return w.run("save", a.Name)
	case music.SaveTracks:
		return writePlaylistTracks(w, a.Name, a.Tracks, true)
	case music.CreatePlaylist, music.RenamePlaylist, music.DeletePlaylist, music.AddToPlaylist,
		music.MovePlaylistTrack, music.DeletePlaylistTrack:
		return editPlaylist(w, a)
	default:
		return errors.New("unknown playback operation")
	}
}

func readStatus(w *wire) (music.Status, error) {
	fields, err := w.query("status")
	if err != nil {
		return music.Status{}, err
	}
	id, err := strconv.Atoi(get(fields, "songid"))
	if err != nil {
		id = -1
	}
	return music.Status{
		State: get(fields, "state"), SongID: id,
		Elapsed: duration(get(fields, "elapsed")), Duration: duration(get(fields, "duration")),
		Random: get(fields, "random") == "1", Repeat: get(fields, "repeat") == "1",
		QueueVersion: get(fields, "playlist"),
	}, nil
}

func duration(s string) time.Duration {
	d, _ := time.ParseDuration(s + "s")
	return d
}

func bit(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func readQueue(w *wire) ([]music.Entry, error) {
	fields, err := w.query("playlistinfo")
	if err != nil {
		return nil, err
	}
	var entries []music.Entry
	eachTrack(fields, func(group []field) {
		if id, err := strconv.Atoi(get(group, "Id")); err == nil {
			entries = append(entries, music.Entry{ID: id, Track: track(group)})
		}
	})
	return entries, nil
}

func eachTrack(fields []field, visit func([]field)) {
	start := -1
	for i, f := range fields {
		switch f.key {
		case "file", "directory", "playlist":
			if start >= 0 {
				visit(fields[start:i])
			}
			start = -1
			if f.key == "file" {
				start = i
			}
		}
	}
	if start >= 0 {
		visit(fields[start:])
	}
}

func track(fields []field) music.Track {
	length := duration(get(fields, "duration"))
	if length <= 0 {
		// Older MPD servers report whole seconds in the Time field.
		length = max(0, duration(get(fields, "Time")))
	}
	added, _ := time.Parse(time.RFC3339, get(fields, "added"))
	modified, _ := time.Parse(time.RFC3339, get(fields, "Last-Modified"))
	return music.Track{Duration: length, Added: added.UTC(), Modified: modified.UTC(),
		URI: get(fields, "file"), Title: get(fields, "Title"), Artist: get(fields, "Artist"),
		Album: get(fields, "Album"), AlbumArtist: get(fields, "AlbumArtist"), Genre: get(fields, "Genre"),
		Disc: number(get(fields, "Disc")), Number: number(get(fields, "Track")),
	}
}

func number(s string) int {
	for i, ch := range s {
		if ch < '0' || ch > '9' {
			s = s[:i]
			break
		}
	}
	value, _ := strconv.Atoi(s)
	return value
}
