package dream

import (
	"context"
	"testing"

	"rampd/internal/music"
)

func TestDreamUsesSeedsWithoutReplayingThem(t *testing.T) {
	seed := music.Track{URI: "picked", Artist: "Band", Genre: "Jazz"}
	library := []music.Track{
		seed,
		{URI: "artist-match", AlbumArtist: " band "},
		{URI: "genre-match", Artist: "Other", Genre: "JAZZ"},
		{URI: "unrelated", Artist: "Unrelated", Genre: "Rock"},
		{URI: "genre-match", Artist: "Other", Genre: "JAZZ"},
	}
	options := Options{TrackCount: 25, Seeds: []music.Track{seed}}
	tracks, err := (Baseline{}).Generate(context.Background(), library, options)
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 2 || tracks[0].URI == tracks[1].URI {
		t.Fatalf("expected two distinct matches: %+v", tracks)
	}
	for _, track := range tracks {
		if track.URI != "artist-match" && track.URI != "genre-match" {
			t.Fatalf("Dream included a seed or unrelated track: %+v", track)
		}
	}
	if _, err := (Baseline{}).Generate(context.Background(), []music.Track{seed}, options); err == nil {
		t.Fatal("expected an explicit error when no related tracks exist")
	}
}
