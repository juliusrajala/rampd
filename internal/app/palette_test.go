package app

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"rampd/internal/keymap"
	"rampd/internal/music"
)

func TestPaletteRunsUnboundActionAndPreservesFilter(t *testing.T) {
	keys, err := keymap.New(map[string][]string{"play_pause": {}})
	if err != nil {
		t.Fatal(err)
	}
	server := &fakeBackend{}
	m := New(context.Background(), server, Options{Keys: keys}).(*model)
	m.connected, m.busy = true, false
	m.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	m.Update(prepareLibrary(fixtureTracks, nil))
	m.page().setFilter("violet")
	m.Update(keyRune(':'))
	m.Update(tea.PasteMsg{Content: "pause"})
	view := ansi.Strip(m.View().Content)
	if m.prompt != palettePrompt || !strings.Contains(view, "Play or pause") || !strings.Contains(view, "[ : pause") {
		t.Fatalf("no searchable palette: %q", view)
	}
	cmd := keyCommand(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("palette did not run the command")
	}
	cmd()
	if len(server.actions) != 1 || server.actions[0].Operation != music.Toggle {
		t.Fatalf("wrong palette action: %+v", server.actions)
	}
	if m.prompt != noPrompt || m.page().filter != "violet" {
		t.Fatal("palette did not restore the filtered view")
	}
}

func TestPaletteClearRequiresConfirmation(t *testing.T) {
	server := &fakeBackend{}
	m := New(context.Background(), server, Options{}).(*model)
	m.connected, m.busy, m.tab = true, false, queueTab
	m.queue = []music.Entry{{ID: 53, Track: fixtureTracks[0]}}
	m.buildQueuePage()
	m.key(keyRune(':'))
	m.Update(tea.PasteMsg{Content: "clear"})
	if cmd := m.key(tea.KeyPressMsg{Code: tea.KeyEnter}); cmd != nil || m.confirmation == nil || len(server.actions) != 0 {
		t.Fatal("destructive palette action bypassed confirmation")
	}
	if cmd := m.key(tea.KeyPressMsg{Code: tea.KeyEsc}); cmd != nil || m.confirmation != nil {
		t.Fatal("Escape did not cancel the queue clear")
	}
	if len(server.actions) != 0 {
		t.Fatal("cancelling changed the queue")
	}
}

func TestCustomBindingChangesHintAndDispatch(t *testing.T) {
	keys, err := keymap.New(map[string][]string{"filter": {"f"}})
	if err != nil {
		t.Fatal(err)
	}
	m := New(context.Background(), &fakeBackend{}, Options{Keys: keys}).(*model)
	if hints := ansi.Strip(m.actionHints()); !strings.Contains(hints, "[f] Filter") || strings.Contains(hints, "[/] Filter") {
		t.Fatalf("default binding left in hint: %q", hints)
	}
	m.key(keyRune('/'))
	if m.prompt != noPrompt {
		t.Fatal("old binding still dispatches")
	}
	m.key(keyRune('f'))
	if m.prompt != searchPrompt {
		t.Fatal("new binding did not dispatch")
	}
}
