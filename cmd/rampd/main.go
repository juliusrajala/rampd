package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"rampd/internal/app"
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err = tea.NewProgram(app.New(ctx, client, options), tea.WithContext(ctx)).Run()
	return err
}
