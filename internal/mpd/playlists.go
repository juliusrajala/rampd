package mpd

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"

	"rampd/internal/music"
)

func (c Client) Playlists(ctx context.Context) ([]string, error) {
	w, err := c.open(ctx)
	if err != nil {
		return nil, err
	}
	defer w.Close()
	return listPlaylists(w)
}

func (c Client) Playlist(ctx context.Context, name string) ([]music.Track, error) {
	w, err := c.open(ctx)
	if err != nil {
		return nil, err
	}
	defer w.Close()
	fields, err := w.query("listplaylistinfo", name)
	if err != nil {
		return nil, err
	}
	var tracks []music.Track
	eachTrack(fields, func(group []field) {
		tracks = append(tracks, track(group))
	})
	return tracks, nil
}

func listPlaylists(w *wire) ([]string, error) {
	fields, err := w.query("listplaylists")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, f := range fields {
		if f.key == "playlist" {
			names = append(names, f.value)
		}
	}
	return names, nil
}

func checkPlaylistName(name string) error {
	if strings.TrimSpace(name) == "" || strings.ContainsAny(name, "/\x00\r\n") {
		return errors.New("enter a nonempty playlist name without a slash or line break")
	}
	return nil
}

func writePlaylistTracks(w *wire, name string, tracks []music.Track, create bool) error {
	if err := checkPlaylistName(name); err != nil {
		return err
	}
	if len(tracks) == 0 {
		return errors.New("no tracks to save")
	}
	names, err := listPlaylists(w)
	if err != nil {
		return err
	}
	exists := slices.Contains(names, name)
	if create && exists {
		return errors.New("that playlist already exists; choose a new name")
	}
	if !create && !exists {
		return errors.New("that playlist does not exist")
	}
	// MPD creates a missing playlist on the first playlistadd. The existence
	// check prevents mistakes, but MPD cannot make it atomic with the write.
	commands := make([]command, 0, len(tracks))
	for _, track := range tracks {
		commands = append(commands, command{"playlistadd", []string{name, track.URI}})
	}
	return w.runList(commands)
}

func editPlaylist(w *wire, a music.Action) error {
	if err := checkPlaylistName(a.Name); err != nil {
		return err
	}
	switch a.Operation {
	case music.CreatePlaylist:
		// MPD can save a new playlist without touching the active queue. The
		// list stops at the first error, so an existing playlist is never cleared.
		return w.runList([]command{
			{"save", []string{a.Name}},
			{"playlistclear", []string{a.Name}},
		})
	case music.RenamePlaylist:
		if err := checkPlaylistName(a.NewName); err != nil {
			return err
		}
		return w.run("rename", a.Name, a.NewName)
	case music.DeletePlaylist:
		return w.run("rm", a.Name)
	case music.AddToPlaylist:
		return writePlaylistTracks(w, a.Name, a.Tracks, false)
	case music.MovePlaylistTrack, music.DeletePlaylistTrack:
		return editPlaylistPosition(w, a)
	default:
		return errors.New("unknown playlist operation")
	}
}

func editPlaylistPosition(w *wire, a music.Action) error {
	if a.Position < 0 || a.Position >= len(a.Expected) {
		return errors.New("invalid stored playlist position")
	}
	fields, err := w.query("listplaylist", a.Name)
	if err != nil {
		return err
	}
	var uris []string
	for _, f := range fields {
		if f.key == "file" {
			uris = append(uris, f.value)
		}
	}
	// MPD addresses stored tracks by position, not ID. Reject an edit to a
	// changed list. MPD has no atomic check-and-edit command.
	if len(uris) != len(a.Expected) {
		return errors.New("playlist changed in MPD; retry after refresh")
	}
	for i, uri := range uris {
		if uri != a.Expected[i].URI {
			return errors.New("playlist changed in MPD; retry after refresh")
		}
	}
	from := strconv.Itoa(a.Position)
	if a.Operation == music.DeletePlaylistTrack {
		return w.run("playlistdelete", a.Name, from)
	}
	to := a.Position + a.Delta
	if to < 0 || to >= len(uris) {
		return nil
	}
	return w.run("playlistmove", a.Name, from, strconv.Itoa(to))
}
