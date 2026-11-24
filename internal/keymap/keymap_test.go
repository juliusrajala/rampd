package keymap

import "testing"

func TestOverridesAndConflicts(t *testing.T) {
	b, err := New(map[string][]string{"play_pause": {"t", "ctrl+p"}, "skip": {}})
	if err == nil {
		t.Fatal("unknown action was accepted")
	}
	b, err = New(map[string][]string{"play_pause": {"n"}})
	if err == nil {
		t.Fatal("conflicting keys were accepted")
	}
	b, err = New(map[string][]string{"play_pause": {"n", "u"}, "next": {}})
	if err != nil {
		t.Fatal(err)
	}
	if b.Action("p") != "" || b.Action("n") != PlayPause || b.Action("u") != PlayPause ||
		b.Hint(PlayPause, "Play/pause") != "[n] Play/pause" || b.Hint(Next, "Next") != "" {
		t.Fatal("overrides did not replace the default keys and hints")
	}
}
