# AGENTS.md — rämpd player

## Project context

**rämpd player** is a keyboard-first terminal music player for MPD. Use Go, Bubble Tea, Bubbles, and Lip Gloss. The primary target is Linux terminals, including over SSH. Keep the code portable where practical.

Check the code and tests before treating a planned feature as implemented.

Deliver complete, testable slices. Start with a terminal shell that connects to MPD, shows status, and handles basic playback keys. Build library browsing, queue operations, playlists, and Dream on that foundation. Good keyboard interaction is more important than feature count.

## Go conventions

Write idiomatic Go. Prefer small packages with clear responsibilities and the simplest design that meets the requirement. Use [Effective Go](https://go.dev/doc/effective_go) and [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) as references.

- Use short, descriptive package names. Avoid catch-all packages such as `utils` or `common`.
- Keep exported APIs small. Use concrete types by default and typed structs for known data.
- Define small interfaces where callers need a real boundary. Do not add an interface for every dependency.
- Return errors for expected failures. Add useful context and wrap errors with `%w` when callers need the cause. Do not log and return the same error at the same layer.
- Pass `context.Context` to operations that can block or need cancellation. Make resource ownership and cleanup clear.
- Avoid global mutable state. Prefer direct control flow. Add concurrency only when it serves a clear purpose.
- Prefer the standard library and existing dependencies where they fit. Use an established Go MPD library where practical. Check its support for timeouts, `idle`, and binary responses before relying on it for those operations.

## Architecture and MPD

- Keep `main` small. It should load configuration, construct dependencies, and start the application. Add internal packages as concrete needs arise.
- Let Bubble Tea own terminal input, navigation, and presentation. Keep library, queue, playlist, and Dream operations testable without a terminal.
- Keep MPD commands, response parsing, connection management, and library-specific types inside the MPD adapter. Pass application types across that boundary.
- Treat MPD as the source of truth for playback, the active queue, and stored playlists. Do not add an audio engine, a second MPRIS service, or a separate music database. Never modify source music files.
- Use stable MPD song IDs for active queue operations where supported. Do not treat a cached queue position as a stable identity. Keep the active queue distinct from stored playlists.
- Use MPD `idle` notifications for external changes. Refresh affected state only and preserve selection when possible. Use a separate idle connection if needed. Use bounded backoff for reconnects and resync state after reconnecting.
- Configure the MPD endpoint, credentials, and timeouts. Default to `127.0.0.1:6600`. Follow XDG conventions on Linux and do not include passwords in logs or user-facing errors.

## Terminal interaction

- Keep `Update` and `View` fast. Run network I/O, Dream generation, and artwork processing outside those synchronous paths. Return results as messages. Cancel obsolete work or ignore stale results, especially for search and artwork.
- Keep keybindings configurable. Give focused text inputs their keystrokes without triggering global playback commands.
- Preserve selection and scroll position when users return to a view. Make the interface usable in narrow terminals and without artwork.
- Confirm destructive actions such as clearing the queue or overwriting or deleting a stored playlist.
- Fetch artwork through MPD. Handle missing artwork and chunked responses. Cache processed artwork outside the render path. Use Unicode half-blocks as the default renderer. Defer Kitty and Sixel support until the base UI is stable.

## Dream

- Keep recommendation strategies independent of Bubble Tea, the MPD library, and playback commands. The application supplies library tracks and options to the selected strategy.
- Keep Dream offline. The baseline strategy selects playable tracks, avoids duplicate entries, and avoids consecutive tracks by the same artist when possible. Default to 25 tracks and handle small or sparsely tagged libraries.
- Opening Dream, generating a session, and regenerating it are preview operations. They must not change the queue or stored playlists. Let the user explicitly play, append, replace, or save the preview.
- Keep the strategy replaceable without changes to the Dream view. Do not add listening history, external services, or advanced recommendation signals to the MVP.

## Testing and changes

- Use Go's `testing` package. Prefer table-driven tests where they clarify related cases. Test observable behavior and errors, not only helpers.
- Use fakes at the MPD boundary for unit tests. Test key dispatch, selection preservation, stale messages, and Dream preview behavior without a terminal or live MPD server.
- Keep MPD integration tests opt-in and use a disposable instance with fixture music. Never run mutating tests against a user's live MPD instance.
- Test resize behavior and output without truecolor when changing the UI. Test Dream against small libraries and with a substitute strategy. Avoid sleeps in tests where explicit synchronization works.
- Keep changes focused. Add regression tests for bug fixes and update tests when behavior changes.
- Run `gofmt` on changed Go files, then `go test ./...` and `go vet ./...` when Go code changes. Run `go test -race ./...` for concurrency changes. Report checks that could not run.
- Explain why non-obvious behavior exists. Keep installation and usage documentation separate from design notes. Do not describe planned features as available.
