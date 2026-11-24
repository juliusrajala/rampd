package mpd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const maxArtwork = 16 << 20

func (c Client) Artwork(ctx context.Context, uri string) ([]byte, error) {
	w, err := c.open(ctx)
	if err != nil {
		return nil, err
	}
	defer w.Close()

	for _, name := range []string{"readpicture", "albumart"} {
		data, err := readArtwork(w, name, uri)
		if err != nil {
			var ack ackError
			if errors.As(err, &ack) &&
				(strings.HasPrefix(string(ack), "ACK [50@") || strings.HasPrefix(string(ack), "ACK [5@")) {
				continue
			}
			return nil, err
		}
		if len(data) != 0 {
			return data, nil
		}
	}
	return nil, nil
}

func readArtwork(w *wire, name, uri string) ([]byte, error) {
	var data []byte
	for {
		size, chunk, err := artworkChunk(w, name, uri, len(data))
		if err != nil {
			return nil, err
		}
		if size == 0 && len(data) == 0 {
			return nil, nil
		}
		if len(chunk) == 0 || size < len(data)+len(chunk) {
			return nil, errors.New("incomplete MPD artwork")
		}
		data = append(data, chunk...)
		if len(data) == size {
			return data, nil
		}
	}
}

func artworkChunk(w *wire, name, uri string, offset int) (size int, chunk []byte, err error) {
	var out strings.Builder
	if err := (command{name, []string{uri, strconv.Itoa(offset)}}).line(&out); err != nil {
		return 0, nil, err
	}
	if _, err := io.WriteString(w, out.String()); err != nil {
		return 0, nil, err
	}

	for {
		line, err := w.line()
		switch {
		case err != nil:
			return 0, nil, err
		case line == "OK":
			return size, chunk, nil
		case strings.HasPrefix(line, "ACK "):
			return 0, nil, ackError(line)
		}
		key, value, ok := strings.Cut(line, ": ")
		if !ok {
			return 0, nil, errors.New("invalid MPD artwork response")
		}
		switch key {
		case "size":
			size, err = strconv.Atoi(value)
			if err != nil || size < 0 || size > maxArtwork {
				return 0, nil, fmt.Errorf("MPD artwork exceeds the %d MiB limit or has an invalid size", maxArtwork>>20)
			}
		case "binary":
			n, err := strconv.Atoi(value)
			if err != nil || n <= 0 || n > size-offset || chunk != nil {
				return 0, nil, errors.New("invalid MPD artwork chunk size")
			}
			chunk = make([]byte, n)
			if _, err := io.ReadFull(w.input, chunk); err != nil {
				return 0, nil, err
			}
			// MPD sends a line break after the binary payload, before the next protocol line.
			separator, err := w.input.ReadByte()
			if err != nil || separator != '\n' {
				return 0, nil, errors.New("invalid MPD artwork chunk separator")
			}
		}
	}
}
