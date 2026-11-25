// Package music defines the data exchanged between the player and its UI.
package music

import "time"

type Track struct {
	URI, Title, Artist, AlbumArtist, Album, Genre string
	Disc, Number                                  int
	Duration                                      time.Duration // Zero if unknown.
	Added, Modified                               time.Time     // Zero if unknown.
}

// PlayHistory holds the statistics recorded by rämpd for one track.
type PlayHistory struct {
	LastPlayed time.Time
	Yearly     map[int]uint64
}

type Entry struct {
	ID int
	Track
}

type Status struct {
	State             string
	SongID            int // -1 if MPD has no current song.
	Elapsed, Duration time.Duration
	Random, Repeat    bool
	QueueVersion      string
}

// PlayingID includes paused playback. A stopped queue has no playing ID.
func (s Status) PlayingID() int {
	if s.State == "play" || s.State == "pause" {
		return s.SongID
	}
	return -1
}

type Snapshot struct {
	Status
	Queue        []Entry
	QueueChanged bool
}

// Changes identifies the data to refresh after an MPD notification.
type Changes uint8

const (
	SessionReady Changes = 1 << iota
	PlaybackChanged
	DatabaseChanged
	StoredPlaylistsChanged
)

type Operation int

const (
	Toggle Operation = iota
	Next
	Previous
	Seek
	Random
	Repeat
	Append
	AppendAndPlay
	ReplaceAndPlay
	ReplaceQueue
	Play
	Remove
	Move
	Clear
	Shuffle
	SaveQueue
	SaveTracks
	CreatePlaylist
	RenamePlaylist
	DeletePlaylist
	AddToPlaylist
	MovePlaylistTrack
	DeletePlaylistTrack
)

type Action struct {
	Operation   Operation
	ID          int
	Delta       int // Move positions or seek seconds.
	Tracks      []Track
	Name        string // Stored playlist name.
	NewName     string
	Position    int     // Position in the full stored playlist, not the filtered list.
	Expected    []Track // Stored playlist contents when the user chose a position.
	Replacement *QueueReplacement
}

// QueueReplacement describes the queue shown at confirmation time.
type QueueReplacement struct {
	Version   string
	Before    []Entry
	CurrentID int
	KeepAt    int // Desired index of CurrentID, or -1 to stop before removing it.
}
