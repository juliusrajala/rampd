package theme

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestFromConfigOverridesOnlyRequestedColors(t *testing.T) {
	base := Default()
	styles, err := FromConfig(map[string]string{
		"artist": "#123abc", "selection_background": "#223344", "shortcut": "#112233",
	})
	if err != nil {
		t.Fatal(err)
	}
	if styles.ArtistName.GetForeground() != lipgloss.Color("#123abc") ||
		styles.Selection.GetBackground() != lipgloss.Color("#223344") ||
		styles.Input.Cursor.Color != lipgloss.Color("#112233") ||
		styles.Input.Focused.Prompt.GetForeground() != lipgloss.Color("#112233") {
		t.Fatal("overrides did not reach the expected styles")
	}
	if styles.SongName.GetForeground() != base.SongName.GetForeground() ||
		styles.Selection.GetForeground() != base.Selection.GetForeground() ||
		!styles.OutlinePanels || Default().ArtistName.GetForeground() != base.ArtistName.GetForeground() {
		t.Fatal("overrides changed the default styles")
	}
}

func TestFromConfigRejectsInvalidColors(t *testing.T) {
	for _, value := range []string{"", "#abc", "red", "#zz4466", "#22334455", "\x1b[31m"} {
		if _, err := FromConfig(map[string]string{"artist": value}); err == nil ||
			!strings.Contains(err.Error(), "theme.artist") {
			t.Errorf("expected a role-specific error for %q, got %v", value, err)
		}
	}
	if _, err := FromConfig(map[string]string{"artsit": "#112233"}); err == nil ||
		!strings.Contains(err.Error(), "artsit") {
		t.Fatalf("expected an error for an unknown role, got %v", err)
	}
}
