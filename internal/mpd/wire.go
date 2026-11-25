// Package mpd connects the player to Music Player Daemon.
package mpd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

type Client struct {
	Network, Address, Password string
	Timeout                    time.Duration
}

type wire struct {
	net.Conn
	input *bufio.Reader
	stop  func() bool
}

type field struct {
	key, value string
}

type ackError string

func (e ackError) Error() string { return string(e) }

var escape = strings.NewReplacer(`\`, `\\`, `"`, `\"`)

func (c Client) open(ctx context.Context) (*wire, error) {
	dialCtx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(dialCtx, c.Network, c.Address)
	if err != nil {
		return nil, err
	}
	deadline, _ := dialCtx.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		conn.Close()
		return nil, err
	}
	// The deadline bounds requests. Closing the connection also stops a request on exit.
	w := &wire{Conn: conn, input: bufio.NewReader(conn), stop: context.AfterFunc(ctx, func() { conn.Close() })}
	greeting, err := w.line()
	if err != nil || !strings.HasPrefix(greeting, "OK MPD ") {
		w.Close()
		if err != nil {
			return nil, err
		}
		return nil, errors.New("invalid MPD greeting")
	}
	if c.Password != "" {
		if err := w.run("password", c.Password); err != nil {
			w.Close()
			// A server can include the contents of a command in an error.
			return nil, errors.New("MPD password authentication failed")
		}
	}
	return w, nil
}

func (w *wire) Close() error {
	w.stop()
	return w.Conn.Close()
}

func (w *wire) line() (string, error) {
	line, err := w.input.ReadString('\n')
	return strings.TrimSuffix(line, "\n"), err
}

type command struct {
	name string
	args []string
}

func (c command) line(out *strings.Builder) error {
	out.WriteString(c.name)
	for _, arg := range c.args {
		if strings.ContainsAny(arg, "\x00\r\n") {
			return errors.New("MPD argument contains a line break or NUL")
		}
		out.WriteString(` "`)
		out.WriteString(escape.Replace(arg))
		out.WriteByte('"')
	}
	out.WriteByte('\n')
	return nil
}

func (w *wire) runList(commands []command) error {
	_, err := w.queryList(commands)
	return err
}

func (w *wire) queryList(commands []command) ([]field, error) {
	if len(commands) == 0 {
		return nil, nil
	}
	var out strings.Builder
	out.WriteString("command_list_begin\n")
	for _, c := range commands {
		if err := c.line(&out); err != nil {
			return nil, err
		}
	}
	out.WriteString("command_list_end\n")
	return w.exchange(out.String(), "command list")
}

func (w *wire) query(name string, args ...string) ([]field, error) {
	var out strings.Builder
	if err := (command{name, args}).line(&out); err != nil {
		return nil, err
	}
	return w.exchange(out.String(), name)
}

func (w *wire) exchange(request, name string) ([]field, error) {
	if _, err := io.WriteString(w, request); err != nil {
		return nil, err
	}
	var fields []field
	for {
		line, err := w.line()
		switch {
		case err != nil:
			return nil, err
		case line == "OK":
			return fields, nil
		case strings.HasPrefix(line, "ACK "):
			// Earlier commands in a list may already have returned data.
			return fields, ackError(line)
		}
		key, value, ok := strings.Cut(line, ": ")
		if !ok {
			return nil, fmt.Errorf("invalid MPD response to %s", name)
		}
		fields = append(fields, field{key, value})
	}
}

func (w *wire) run(name string, args ...string) error {
	_, err := w.query(name, args...)
	return err
}

func get(fields []field, key string) string {
	for _, f := range fields {
		if f.key == key {
			return f.value
		}
	}
	return ""
}
