# Theme settings

Put color overrides in the `[theme]` table in `config.toml`. Use `#RRGGBB` strings. Start the player again after editing the file. The player uses the built-in Synthwave colors for settings you omit.

This table lists the default values. Copy only the lines you want to change.

```toml
[theme]
heading_text = "#FFE0F7"
heading_background = "#40225E" # Also the inactive row highlight
muted = "#B29AC9"
accent = "#FF8CC9"
shortcut = "#77F5E5"
selection_text = "#180E2A"
selection_background = "#FF8CC9"
active_tab_text = "#180E2A"
active_tab_background = "#77F5E5"
song = "#F5EFFB"
artist = "#CEB5F5"
album = "#EFB69C"
panel_border = "#D8B4FE"
divider = "#8D61AE"
online = "#77F5E5"
offline = "#FFAA82"
progress_played = "#77F5E5" # Also used for times and known lengths
progress_unplayed = "#3A3743"
artwork_background = "#180E2A"
motif_top = "#F95DB5"
motif_middle = "#FF7FB2"
motif_bottom = "#FFAA82"
```

Choose selection text and background colors with good contrast. The selected row uses these two colors instead of the song, artist, album, and length colors.
