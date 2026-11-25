package keymap

import "testing"

func TestDefaultHelpAndBackKeys(t *testing.T) {
	b := Default()
	if b.Action("h") != Guide || b.Action("?") != Help ||
		b.Action("left") != Back || b.Action("backspace") != Back {
		t.Fatal("expected separate bindings for Help, Keybindings, and Back")
	}
}

func TestDefaultSeekKeys(t *testing.T) {
	b := Default()
	if b.Action(",") != SeekBack || b.Action(".") != SeekForward {
		t.Fatal("comma and period should seek backward and forward")
	}
}

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
