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
	keys, err := keymap.New(map[string][]string{"append": {}})
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
	m.Update(tea.PasteMsg{Content: "add to queue"})
	view := ansi.Strip(m.View().Content)
	if m.prompt != palettePrompt || !strings.Contains(view, "Add to queue") || !strings.Contains(view, "  : add to queue") || !strings.Contains(view, "[ / violet ]") {
		t.Fatalf("no searchable palette: %q", view)
	}
	cmd := keyCommand(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("palette did not run the command")
	}
	cmd()
	if len(server.actions) != 1 || server.actions[0].Operation != music.Append {
		t.Fatalf("wrong palette action: %+v", server.actions)
	}
	if m.prompt != noPrompt || m.page().filter != "violet" {
		t.Fatal("palette did not restore the filtered view")
	}
}

func TestPaletteOmitsControlsFromOtherPanels(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	for _, item := range m.commandRows() {
		switch keymap.Action(item.key) {
		case keymap.PlayPause, keymap.Previous, keymap.Next, keymap.SeekBack, keymap.SeekForward,
			keymap.Random, keymap.Repeat, keymap.FocusSong:
			t.Errorf("palette duplicated a Playback control: %q", item.label)
		}
		for _, action := range tabActions {
			if item.key == string(action) {
				t.Errorf("palette duplicated a navigation entry: %q", item.label)
			}
		}
	}
}

func TestSidebarCommandGroups(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	m.Update(tea.WindowSizeMsg{Width: 112, Height: 28})
	m.Update(prepareLibrary(fixtureTracks, nil))
	lines := strings.Split(m.View().Content, "\n")
	for _, line := range lines[1+navRows : sidebarCommandInputRow] {
		if strings.TrimSpace(ansi.Strip(ansi.Cut(line, 0, sidePanelWidth))) != "" {
			t.Fatal("expected blank rows between navigation and commands")
		}
	}
	var menu []string
	for _, line := range lines[sidebarCommandInputRow:] {
		menu = append(menu, strings.Join(strings.Fields(ansi.Strip(ansi.Cut(line, 0, sidePanelWidth))), " "))
	}
	want := "Sort by artist [o]\nPlay track [enter]\n\n" +
		"Add to Tray [t]\nAdd to queue [space]\nAdd to playlist [A]\n\n" +
		"Filter [/]\nCommands [:]\nKeybindings [?]\nHelp [h]\nQuit [q]"
	if got := strings.TrimRight(strings.Join(menu, "\n"), "\n"); got != want {
		t.Fatalf("unexpected sidebar menu:\n%s", got)
	}
	m.Update(keyRune('2'))
	var open string
	for _, item := range m.commandRows() {
		if item.key == string(keymap.Open) {
			open = item.label
		}
	}
	if open != "Open" {
		t.Fatalf("album action = %q, want Open", open)
	}
	m.Update(keyRune(':'))
	for range len(m.palettePage.rows) {
		m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	if got := ansi.Strip(m.View().Content); !strings.Contains(got, "› Quit") || strings.Contains(got, "COMMANDS") {
		t.Fatalf("expected to scroll past group gaps to Quit:\n%s", got)
	}
}

func TestRefreshIsSearchOnly(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	m.Update(tea.WindowSizeMsg{Width: 112, Height: 40})
	if view := ansi.Strip(m.View().Content); strings.Contains(view, "Refresh") {
		t.Fatal("Refresh should not appear in the sidebar preview")
	}
	m.Update(keyRune(':'))
	m.Update(tea.PasteMsg{Content: "refresh"})
	if item, ok := m.palettePage.selected(); !ok || item.key != string(keymap.Refresh) {
		t.Fatal("Refresh should be available in palette search")
	}
}

func TestHelpGuideAndKeybindingsAreSeparate(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	m.Update(tea.WindowSizeMsg{Width: 45, Height: 12})
	m.Update(prepareLibrary(fixtureTracks, nil))
	m.Update(keyRune('4'))
	m.browsePage().setFilter("first")
	m.Update(keyRune(':'))
	m.Update(tea.PasteMsg{Content: "help"})
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.help != guideHelp || !strings.Contains(ansi.Strip(m.View().Content), "Browse your MPD music library") {
		t.Fatal("Help did not open the guide")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
	if m.helpTop == 0 {
		t.Fatal("guide did not scroll")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnd})
	if !strings.Contains(ansi.Strip(m.View().Content), "for later.") {
		t.Fatal("end of the guide did not explain Playlists")
	}
	m.Update(tea.WindowSizeMsg{Width: 112, Height: 44})
	guide := ansi.Strip(m.View().Content)
	for _, text := range []string{"MPD plays from this list", "Collect and arrange tracks", "Make a preview"} {
		if !strings.Contains(guide, text) {
			t.Errorf("guide is missing %q", text)
		}
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if m.help != noHelp || m.tab != trayTab || m.browsePage().filter != "first" {
		t.Fatal("closing Help did not restore the filtered Tray")
	}
	m.Update(keyRune('?'))
	if m.help != keysHelp || !strings.Contains(ansi.Strip(m.View().Content), "Keybindings ·") {
		t.Fatal("the keybindings shortcut did not open the separate shortcut list")
	}
}

func TestDefaultCommandLabelsFitSidebar(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{InitialTray: fixtureTracks}).(*model)
	m.Update(tea.WindowSizeMsg{Width: 112, Height: 40})
	m.Update(prepareLibrary(fixtureTracks, nil))
	m.Update(dreamMsg{tracks: fixtureTracks})
	m.Update(playlistsMsg{names: []string{"Mix"}})
	m.queue = []music.Entry{{ID: 1, Track: fixtureTracks[0]}, {ID: 2, Track: fixtureTracks[1]}}
	m.buildQueuePage()
	width := m.innerWidth(sidePanelWidth)
	check := func() {
		for _, item := range m.commandRows() {
			key := keymap.Button(item.detail)
			if want := 2 + ansi.StringWidth(item.label) + 2 + ansi.StringWidth(key); want > width {
				t.Errorf("%s: %q and %q need %d columns, have %d", tabNames[m.tab], item.label, key, want, width)
			}
		}
	}
	for tab := range tabNames {
		m.tab = tab
		check()
	}
	m.tab = trayTab
	m.Update(keyRune('v'))
	check()
	m.tab = playlistsTab
	m.openPlaylist("Mix")
	m.Update(playlistMsg{name: "Mix", tracks: fixtureTracks})
	check()
}

func TestKeybindingsColumnsAndScroll(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	m.Update(tea.WindowSizeMsg{Width: 112, Height: 24})
	m.Update(keyRune('?'))
	first := ansi.Strip(m.helpRows()[0])
	for _, label := range []string{"General", "Playback", "By view"} {
		if !strings.Contains(first, label) {
			t.Fatalf("missing column %q in %q", label, first)
		}
	}

	m.Update(tea.WindowSizeMsg{Width: 38, Height: 12})
	if first := ansi.Strip(m.helpRows()[0]); first != "General" {
		t.Fatalf("narrow layout should stack the columns: %q", first)
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnd})
	if !strings.Contains(ansi.Strip(m.View().Content), "Queue all") {
		t.Fatal("last view-specific shortcut should be reachable")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if m.help != noHelp {
		t.Fatal("Escape should close Keybindings")
	}
}

func TestSidebarCommandsFollowNavigationAndFitAfterResize(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{Artwork: true, InitialTray: fixtureTracks}).(*model)
	m.Update(tea.WindowSizeMsg{Width: 112, Height: 26})
	m.Update(prepareLibrary(fixtureTracks, nil))
	sidebar := func() string {
		lines := strings.Split(m.View().Content, "\n")
		for i, line := range lines {
			lines[i] = ansi.Strip(ansi.Cut(line, 0, sidePanelWidth))
		}
		return strings.Join(lines, "\n")
	}
	if got := sidebar(); !strings.Contains(got, "Add to Tray") || !strings.Contains(got, "[t]") {
		t.Fatalf("Songs sidebar did not show the Tray action: %q", got)
	}
	m.Update(keyRune('4'))
	if got := sidebar(); !strings.Contains(got, "Mark") || !strings.Contains(got, "Shuffle Tray") ||
		strings.Contains(got, "Switch mode") {
		t.Fatalf("sidebar did not show both Tray action groups: %q", got)
	}
	if m.groupOf(newRow(string(keymap.MarkTray), "", "")) ==
		m.groupOf(newRow(string(keymap.Shuffle), "", "")) {
		t.Fatal("track edits and whole-Tray actions must use different groups")
	}

	m.Update(keyRune(':'))
	m.Update(tea.PasteMsg{Content: "shuffle tray"})
	view := m.View()
	if view.Cursor == nil || view.Cursor.Y != sidebarCommandInputRow ||
		view.Cursor.X >= sidePanelWidth || !m.showArtworkPanel() || m.page().title != "Tray" {
		t.Fatal("focused sidebar should preserve the main list and artwork")
	}
	m.Update(tea.WindowSizeMsg{Width: 50, Height: 18})
	if !m.paletteInMain() || m.page().title != "Commands" || m.palettePage.filter != "shuffle tray" ||
		len(m.palettePage.rows) != 1 {
		t.Fatal("compact fallback lost the active palette filter")
	}
	m.Update(tea.WindowSizeMsg{Width: 112, Height: 26})
	m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if m.prompt != noPrompt || m.page().title != "Tray" {
		t.Fatal("closing the resized palette should restore the Tray")
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
	var binding string
	for _, item := range m.commandRows() {
		if item.key == string(keymap.Filter) {
			binding = item.detail
		}
	}
	if binding != "f" {
		t.Fatalf("palette filter binding = %q, want f", binding)
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
