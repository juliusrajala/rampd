package mpd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"rampd/internal/music"
)

func TestDisposableMPD(t *testing.T) {
	if os.Getenv("RAMPD_MPD_INTEGRATION") != "1" {
		t.Skip("set RAMPD_MPD_INTEGRATION=1 to run disposable MPD tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	dir, err := os.MkdirTemp("", "rampd-mpd-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	musicDir := filepath.Join(dir, "music")
	playlistDir := filepath.Join(dir, "playlists")
	for _, path := range []string{musicDir, playlistDir} {
		if err := os.Mkdir(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	for _, title := range []string{"One", "Two"} {
		path := filepath.Join(musicDir, title+" quote \".flac")
		cmd := exec.CommandContext(ctx, "ffmpeg", "-v", "error", "-f", "lavfi", "-i",
			"sine=frequency=440:duration=30", "-metadata", "artist=Test artist", "-metadata", "album=Test album",
			"-metadata", "title="+title, path)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("make fixture: %v: %s", err, output)
		}
	}
	cover := image.NewRGBA(image.Rect(0, 0, 128, 128))
	random := rand.New(rand.NewPCG(1, 2))
	for i := 0; i < len(cover.Pix); i += 4 {
		rgb := random.Uint32()
		cover.Pix[i], cover.Pix[i+1], cover.Pix[i+2], cover.Pix[i+3] = byte(rgb), byte(rgb>>8), byte(rgb>>16), 255
	}
	var pixels bytes.Buffer
	if err := png.Encode(&pixels, cover); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(musicDir, "cover.png"), pixels.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(dir, "socket")
	cfg := fmt.Sprintf(
		"music_directory %q\nplaylist_directory %q\ndb_file %q\nbind_to_address %q\n"+
			"auto_update \"no\"\naudio_output {\n type \"null\"\n name \"Test output\"\n}\n",
		musicDir, playlistDir, filepath.Join(dir, "db"), socket,
	)
	conf := filepath.Join(dir, "mpd.conf")
	if err := os.WriteFile(conf, []byte(cfg), 0600); err != nil {
		t.Fatal(err)
	}
	log, err := os.Create(filepath.Join(dir, "mpd.log"))
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	process := exec.CommandContext(ctx, "mpd", "--no-daemon", "--stderr", conf)
	process.Stdout, process.Stderr = log, log
	if err := process.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		process.Process.Kill()
		process.Wait()
		if t.Failed() {
			log.Seek(0, io.SeekStart)
			debug, _ := io.ReadAll(log)
			t.Logf("MPD log:\n%s", debug)
		}
	}()

	client := Client{Network: "unix", Address: socket, Timeout: 2 * time.Second}
	w := waitForMPD(t, ctx, client)
	defer w.Close()

	watchCtx, stopWatch := context.WithCancel(ctx)
	defer stopWatch()
	events := make(chan music.Changes, 32)
	watchDone := make(chan error, 1)
	go func() {
		watchDone <- client.Watch(watchCtx, func(change music.Changes) {
			select {
			case events <- change:
			case <-watchCtx.Done():
			}
		})
	}()
	expectChange := func(want music.Changes) {
		t.Helper()
		timer := time.NewTimer(2 * time.Second)
		defer timer.Stop()
		for {
			select {
			case change := <-events:
				if change&want != 0 {
					return
				}
			case err := <-watchDone:
				t.Fatalf("watch ended before event %d: %v", want, err)
			case <-timer.C:
				t.Fatalf("watch did not deliver event %d", want)
			}
		}
	}
	expectChange(music.SessionReady)
	if err := w.run("update"); err != nil {
		t.Fatal(err)
	}
	expectChange(music.DatabaseChanged)
	var tracks []music.Track
	wait(t, func() bool {
		tracks, err = client.Library(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return len(tracks) == 2
	})
	if tracks[0].Artist != "Test artist" || tracks[0].Album != "Test album" {
		t.Fatalf("unexpected fixture metadata: %+v", tracks)
	}

	if tracks[0].Duration != 30*time.Second {
		t.Fatalf("MPD track duration = %s, want 30s", tracks[0].Duration)
	}
	picture, err := client.Artwork(ctx, tracks[0].URI)
	if err != nil || !bytes.Equal(picture, pixels.Bytes()) {
		t.Fatalf("cover artwork: %d bytes, expected %d (%v)", len(picture), pixels.Len(), err)
	}
	if data, err := client.Artwork(ctx, "missing/none.flac"); err != nil || len(data) != 0 {
		t.Fatalf("missing artwork: %d bytes (%v)", len(data), err)
	}

	snapshot := func(version string) music.Snapshot {
		t.Helper()
		s, err := client.Snapshot(ctx, version)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	act := func(a music.Action) {
		t.Helper()
		if err := client.Apply(ctx, a); err != nil {
			t.Fatal(err)
		}
	}
	s := snapshot("")
	if s.State != "stop" || len(s.Queue) != 0 || s.SongID != -1 {
		t.Fatalf("initial snapshot: %+v", s)
	}
	if s := snapshot(s.QueueVersion); s.QueueChanged {
		t.Fatalf("unchanged queue was fetched: %+v", s)
	}
	act(music.Action{Operation: music.AppendAndPlay, Tracks: tracks})
	expectChange(music.PlaybackChanged)
	s = snapshot("")
	if s.State != "play" || len(s.Queue) != 2 || s.SongID != s.Queue[0].ID {
		t.Fatalf("after append/play: %+v", s)
	}
	id := s.SongID
	act(music.Action{Operation: music.Toggle})
	if s := snapshot(""); s.State != "pause" {
		t.Fatalf("not paused: %+v", s)
	}
	act(music.Action{Operation: music.Seek, Delta: 5})
	if s := snapshot(""); s.Elapsed < 4*time.Second {
		t.Fatalf("seek did not update elapsed time: %+v", s)
	}
	act(music.Action{Operation: music.Move, ID: id, Delta: 1})
	if s := snapshot(""); s.Queue[1].ID != id {
		t.Fatalf("track did not move: %+v", s)
	}
	act(music.Action{Operation: music.Remove, ID: id})
	if s := snapshot(""); len(s.Queue) != 1 || s.Queue[0].ID == id {
		t.Fatalf("track was not removed: %+v", s)
	}

	// An independent client can change options without a write from our client.
	if err := w.run("random", "1"); err != nil {
		t.Fatal(err)
	}
	if s := snapshot(""); !s.Random {
		t.Fatalf("external change was not read: %+v", s)
	}
	before := snapshot("")
	act(music.Action{Operation: music.SaveQueue, Name: "From Queue"})
	expectChange(music.StoredPlaylistsChanged)
	stopWatch()
	select {
	case err := <-watchDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("watch was not cancelled: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancelling watch did not close the idle socket")
	}
	act(music.Action{Operation: music.SaveTracks, Name: "From Dream", Tracks: tracks})
	names, err := client.Playlists(ctx)
	if err != nil || len(names) != 2 {
		t.Fatalf("saved playlist names: %q (%v)", names, err)
	}
	for name, count := range map[string]int{"From Queue": 1, "From Dream": 2} {
		saved, err := client.Playlist(ctx, name)
		if err != nil || len(saved) != count {
			t.Fatalf("saved playlist %q: %+v (%v)", name, saved, err)
		}
	}
	for _, duplicate := range []music.Action{
		{Operation: music.SaveQueue, Name: "From Queue"},
		{Operation: music.SaveTracks, Name: "From Dream", Tracks: tracks},
	} {
		if err := client.Apply(ctx, duplicate); err == nil {
			t.Fatalf("duplicate save accepted: %q", duplicate.Name)
		}
	}
	act(music.Action{Operation: music.CreatePlaylist, Name: "Empty"})
	if empty, err := client.Playlist(ctx, "Empty"); err != nil || len(empty) != 0 {
		t.Fatalf("create empty playlist: %+v (%v)", empty, err)
	}
	act(music.Action{Operation: music.AddToPlaylist, Name: "Empty", Tracks: tracks})
	if err := client.Apply(ctx, music.Action{Operation: music.CreatePlaylist, Name: "Empty"}); err == nil {
		t.Fatal("duplicate create accepted")
	}
	if err := client.Apply(ctx, music.Action{Operation: music.RenamePlaylist, Name: "Empty", NewName: "From Queue"}); err == nil {
		t.Fatal("rename would overwrite another playlist")
	}
	act(music.Action{Operation: music.RenamePlaylist, Name: "Empty", NewName: "Edited"})
	act(music.Action{Operation: music.MovePlaylistTrack, Name: "Edited", Position: 1, Delta: -1, Expected: tracks})
	if err := client.Apply(ctx, music.Action{Operation: music.DeletePlaylistTrack, Name: "Edited", Position: 0, Expected: tracks}); err == nil {
		t.Fatal("stale playlist edit accepted")
	}
	edited, err := client.Playlist(ctx, "Edited")
	if err != nil || len(edited) != 2 || edited[0].URI != tracks[1].URI || edited[1].URI != tracks[0].URI {
		t.Fatalf("playlist tracks not reordered or stale edit applied: %+v (%v)", edited, err)
	}
	act(music.Action{Operation: music.DeletePlaylistTrack, Name: "Edited", Position: 0, Expected: edited})
	if edited, err := client.Playlist(ctx, "Edited"); err != nil || len(edited) != 1 || edited[0].URI != tracks[0].URI {
		t.Fatalf("playlist track not removed: %+v (%v)", edited, err)
	}
	act(music.Action{Operation: music.DeletePlaylist, Name: "Edited"})
	if _, err := client.Playlist(ctx, "Edited"); err == nil {
		t.Fatal("playlist not deleted")
	}
	if err := client.Apply(ctx, music.Action{Operation: music.AddToPlaylist, Name: "Edited", Tracks: tracks}); err == nil {
		t.Fatal("append recreated a missing playlist")
	}
	if after := snapshot(""); after.QueueVersion != before.QueueVersion || after.SongID != before.SongID {
		t.Fatalf("saving a playlist changed the active queue: %+v", after)
	}
	act(music.Action{Operation: music.ReplaceAndPlay, Tracks: tracks})
	if s := snapshot(""); len(s.Queue) != 2 || s.State != "play" || s.Queue[0].ID != s.SongID {
		t.Fatalf("replace/play did not start at the first track: %+v", s)
	}
	s = snapshot("")
	id = s.SongID
	act(music.Action{Operation: music.Seek, Delta: 5})
	replace := music.Action{
		Operation:   music.ReplaceQueue,
		Tracks:      []music.Track{tracks[1], tracks[0], tracks[1]},
		Replacement: &music.QueueReplacement{Version: s.QueueVersion, Before: s.Queue, CurrentID: id, KeepAt: 1},
	}
	act(replace)
	s = snapshot("")
	if len(s.Queue) != 3 || s.Queue[1].ID != id || s.SongID != id || s.State != "play" || s.Elapsed < 4*time.Second ||
		s.Queue[0].URI != tracks[1].URI || s.Queue[2].URI != tracks[1].URI {
		t.Fatalf("Tray replacement did not preserve track, position, order, or repeats: %+v", s)
	}
	if err := client.Apply(ctx, replace); err == nil {
		t.Fatal("stale replacement should be rejected")
	}
	act(music.Action{
		Operation: music.ReplaceQueue, Tracks: []music.Track{tracks[1]},
		Replacement: &music.QueueReplacement{Version: s.QueueVersion, Before: s.Queue, CurrentID: id, KeepAt: -1},
	})
	s = snapshot("")
	if len(s.Queue) != 1 || s.Queue[0].URI != tracks[1].URI || s.State != "stop" {
		t.Fatalf("removing the active track must not start a different track: %+v", s)
	}
	act(music.Action{Operation: music.Clear})
	if s := snapshot(""); len(s.Queue) != 0 {
		t.Fatalf("queue was not cleared: %+v", s)
	}
}

func waitForMPD(t *testing.T, ctx context.Context, c Client) *wire {
	t.Helper()
	var w *wire
	wait(t, func() bool {
		var err error
		w, err = c.open(ctx)
		return err == nil
	})
	return w
}

func wait(t *testing.T, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatal("MPD fixture did not become ready")
}
