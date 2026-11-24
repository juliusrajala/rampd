package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"rampd/internal/app"
	"rampd/internal/music"
	"rampd/internal/traystore"
)

func main() {
	if err := run(); err != nil && !errors.Is(err, flag.ErrHelp) {
		fmt.Fprintln(os.Stderr, "rampd:", err)
		os.Exit(1)
	}
}

func run() error {
	client, options, err := parseOptions(os.Args[1:], os.LookupEnv)
	if err != nil {
		return err
	}
	path, err := traystore.Path()
	if err != nil {
		return fmt.Errorf("locate saved Tray: %w", err)
	}
	options.InitialTray, err = traystore.Load(path)
	if err != nil {
		return fmt.Errorf("load saved Tray: %w (file was not changed)", err)
	}
	options.SaveTray = func(tracks []music.Track) error {
		return traystore.Save(path, tracks)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err = tea.NewProgram(app.New(ctx, client, options), tea.WithContext(ctx)).Run()
	return err
}
