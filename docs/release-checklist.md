# First release checklist

Run these checks in a fresh checkout. Do not use a live music server for tests that change playback or playlists.

## Build and test

1. Use the Go version in `go.mod` or later.
2. Run `go mod verify`, `go test ./...`, and `go vet ./...`.
3. On Linux, install MPD and FFmpeg. Run `RAMPD_MPD_INTEGRATION=1 go test -race -count=1 ./...`.
4. Install tmux and Python 3. Run `bash scripts/smoke.sh` for a terminal test with its own MPD instance.
5. Try wide and narrow terminals. Test navigation, filter input, playlist edits, Dream, and recovery after MPD restarts.
6. Build the Linux binaries for the release. Test each binary on its target architecture before publishing.

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o dist/linux-amd64/rampd ./cmd/rampd
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -o dist/linux-arm64/rampd ./cmd/rampd
```

## Before publishing

- Add the chosen open-source license.
- Verify the version, release notes, and installation steps.
- Package each binary with the license and README. Record a SHA-256 checksum for each archive.
- Inspect the files in the source checkout and archives. Do not include personal settings, private notes, or generated music fixtures.
- Test at least one TCP setup. Check authenticated MPD and a remote terminal session before claiming support for those setups.
