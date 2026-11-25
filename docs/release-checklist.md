# First release checklist

Run these checks in a fresh checkout. Do not use a live music server for tests that change playback or playlists.

## Build and test

1. Use the Go version in `go.mod` or later.
2. Run `go mod verify`, `go test ./...`, and `go vet ./...`.
3. On Linux, install MPD and FFmpeg. Run `RAMPD_MPD_INTEGRATION=1 go test -race -count=1 ./...`.
4. Install tmux and Python 3. Run `bash scripts/smoke.sh` for a terminal test with its own MPD instance.
5. Try wide and narrow terminals. Test navigation, filter input, playlist edits, Dream, and recovery after MPD restarts.
6. Build the Linux and macOS release archives. The script packages x86-64 and ARM64 binaries with the README and license. It also generates SHA-256 checksums.
7. Run the installer checks against the built archives. Then test the client and guided MPD setup on Linux and macOS before publishing.

```sh
bash scripts/package-release.sh
RAMPD_INSTALL_MPD_TEST=1 bash scripts/test-install.sh dist/release
```

## Before publishing

- Verify the version, release notes, license, and installation steps.
- Attach `dist/release/` contents to the GitHub release. The installer expects `install.sh`, `SHA256SUMS`, and the platform archives under their generated names.
- Inspect the files in the source checkout and archives. Do not include personal settings, private notes, or generated music fixtures.
- Test at least one TCP setup. Check authenticated MPD and a remote terminal session before claiming support for those setups.
