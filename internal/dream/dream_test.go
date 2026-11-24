package dream

import (
	"context"
	"errors"
	"slices"
	"testing"

	"rampd/internal/music"
)

func TestBaselineAvoidsDuplicatesAndSpacesArtists(t *testing.T) {
	tracks := []music.Track{
		{URI: "a", Artist: "X"}, {URI: "b", Artist: "X"}, {URI: "c", Artist: "X"},
		{URI: "d", Artist: "Y"}, {URI: "e", Artist: "Y"}, {URI: "d", Artist: "Y"},
		{URI: "", Artist: "Z"}, {URI: "other"},
	}
	before := slices.Clone(tracks)
	for _, size := range []int{3, 25} {
		session, err := (Baseline{}).Generate(context.Background(), tracks, Options{TrackCount: size})
		if err != nil || len(session) != min(size, 6) {
			t.Fatalf("session size %d: %+v (%v)", size, session, err)
		}
		seen := map[string]bool{}
		for i, track := range session {
			if track.URI == "" || seen[track.URI] || i > 0 && track.Artist == session[i-1].Artist {
				t.Fatalf("duplicate, unplayable, or consecutive artist in %+v", session)
			}
			seen[track.URI] = true
		}
	}
	if !slices.Equal(tracks, before) {
		t.Fatal("generation changed the source library")
	}
}

func TestBaselineHandlesSmallAndCancelledLibraries(t *testing.T) {
	tracks := []music.Track{{URI: "untagged"}, {URI: "also-untagged"}}
	options := Options{TrackCount: 25}
	session, err := (Baseline{}).Generate(context.Background(), tracks, options)
	if err != nil || len(session) != 2 {
		t.Fatalf("sparsely tagged library: %+v (%v)", session, err)
	}
	if _, err := (Baseline{}).Generate(context.Background(), nil, options); err == nil {
		t.Fatal("accepted an empty library")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := (Baseline{}).Generate(ctx, tracks, options); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled generation: %v", err)
	}
}
