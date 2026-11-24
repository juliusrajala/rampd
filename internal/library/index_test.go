package library

import (
	"slices"
	"testing"

	"rampd/internal/music"
)

func TestBuildSortsWithoutChangingSource(t *testing.T) {
	tracks := []music.Track{
		{URI: "second", Artist: "Singer B", AlbumArtist: "Various", Album: "Mix", Disc: 2, Number: 1},
		{URI: "first", Artist: "Singer A", AlbumArtist: "Various", Album: "Mix", Disc: 1, Number: 3},
		{URI: "solo", Artist: "Singer A", Album: "Mix"},
		{URI: "untagged"},
	}
	before := slices.Clone(tracks)
	got := Build(tracks)
	if !slices.Equal(before, tracks) {
		t.Fatal("building the index changed the library")
	}
	if len(got.Albums) != 3 || len(got.Artists) != 3 {
		t.Fatalf("wrong grouping: %+v", got)
	}
	for i, name := range []string{"Singer A", "Unknown artist", "Various"} {
		if got.Artists[i].Name != name || len(got.Artists[i].Albums) != 1 {
			t.Fatalf("wrong artist order: %+v", got.Artists)
		}
	}
	compilation := got.Albums[got.Artists[2].Albums[0]]
	if compilation.Artist != "Various" || len(compilation.Tracks) != 2 ||
		compilation.Tracks[0].URI != "first" || compilation.Tracks[1].URI != "second" {
		t.Fatalf("wrong compilation or disc order: %+v", compilation)
	}
	if got.Albums[got.Artists[1].Albums[0]].Name != "Unknown album" {
		t.Fatal("missing fallback for untagged album")
	}
}
