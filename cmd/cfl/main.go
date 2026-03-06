package main

import (
	"context"
	"fmt"
	"os"

	"github.com/takymt/cflcli/internal/cli"
)

func main() {
	cfg, err := loadClientConfigFromEnv()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stdout, err.Error())
		os.Exit(1)
	}

	client, err := newHTTPClient(cfg, nil)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stdout, err.Error())
		os.Exit(1)
	}

	app := cli.New(client, os.Stdout)
	cwd, err := os.Getwd()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stdout, err.Error())
		os.Exit(1)
	}

	os.Exit(app.Run(context.Background(), os.Args[1:], cwd))
}
