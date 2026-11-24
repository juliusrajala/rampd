package main

import (
	"errors"
	"flag"
	"net"
	"strconv"
	"strings"
	"time"

	"rampd/internal/app"
	"rampd/internal/config"
	"rampd/internal/keymap"
	"rampd/internal/mpd"
)

func parseOptions(args []string, env func(string) (string, bool)) (mpd.Client, app.Options, error) {
	var client mpd.Client
	var options app.Options
	defaults := config.Defaults()
	flags := flag.NewFlagSet("rampd", flag.ContinueOnError)
	path := flags.String("config", "", "settings file (default: XDG config directory)")
	hostFlag := flags.String("host", defaults.MPD.Host, "MPD host, host:port, or Unix socket")
	portFlag := flags.Int("port", defaults.MPD.Port, "MPD TCP port for a host without a port")
	timeoutFlag := flags.String("timeout", defaults.MPD.Timeout, "MPD timeout per operation")
	countFlag := flags.Int("dream-count", defaults.Dream.TrackCount, "number of tracks in a Dream preview")
	artworkFlag := flags.Bool("artwork", defaults.UI.Artwork, "show selection details and artwork in wide terminals")
	if err := flags.Parse(args); err != nil {
		return client, options, err
	}
	if flags.NArg() != 0 {
		return client, options, errors.New("unexpected positional arguments")
	}

	cfg, err := config.Load(*path)
	if err != nil {
		return client, options, err
	}
	host, port := cfg.MPD.Host, strconv.Itoa(cfg.MPD.Port)
	if value, _ := env("MPD_HOST"); value != "" {
		host = value
		if !strings.HasPrefix(host, "@") && !strings.HasPrefix(host, "/") {
			if password, address, ok := strings.Cut(host, "@"); ok {
				host, cfg.MPD.Password = address, password
			}
		}
	}
	if value, _ := env("MPD_PORT"); value != "" {
		port = value
	}
	if value, ok := env("MPD_PASSWORD"); ok {
		cfg.MPD.Password = value
	}
	flags.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "host":
			host = *hostFlag
		case "port":
			port = strconv.Itoa(*portFlag)
		case "timeout":
			cfg.MPD.Timeout = *timeoutFlag
		case "dream-count":
			cfg.Dream.TrackCount = *countFlag
		case "artwork":
			cfg.UI.Artwork = *artworkFlag
		}
	})

	if host == "" {
		return client, options, errors.New("MPD host is empty")
	}
	portNum, err := strconv.Atoi(port)
	if err != nil || portNum < 1 || portNum > 65535 {
		return client, options, errors.New("MPD port must be a number between 1 and 65535")
	}
	timeout, err := time.ParseDuration(cfg.MPD.Timeout)
	if err != nil || timeout <= 0 {
		return client, options, errors.New("MPD timeout must be a positive duration such as 5s")
	}
	if cfg.Dream.TrackCount <= 0 {
		return client, options, errors.New("Dream track count must be positive")
	}
	bindings, err := keymap.New(cfg.Keybindings)
	if err != nil {
		return client, options, err
	}
	network, address := endpoint(host, port)
	client = mpd.Client{Network: network, Address: address, Password: cfg.MPD.Password, Timeout: timeout}
	options = app.Options{DreamCount: cfg.Dream.TrackCount, Artwork: cfg.UI.Artwork, Keys: bindings}
	return client, options, nil
}

func endpoint(host, port string) (network, address string) {
	if strings.HasPrefix(host, "/") || strings.HasPrefix(host, "@") {
		return "unix", host
	}
	if _, _, err := net.SplitHostPort(host); err == nil {
		return "tcp", host
	}
	host = strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	return "tcp", net.JoinHostPort(host, port)
}
