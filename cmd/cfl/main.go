package main

import (
	"context"
	"fmt"
	"os"

	"github.com/takymt/cflcli/internal/cli"
	"github.com/takymt/cflcli/internal/page"
)

func main() {
	app := cli.NewLazy(func() (page.Client, error) {
		cfg, err := loadClientConfigFromEnv()
		if err != nil {
			return nil, err
		}
		return newHTTPClient(cfg, nil)
	}, os.Stdout)
	cwd, err := os.Getwd()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stdout, err.Error())
		os.Exit(1)
	}

	os.Exit(app.Run(context.Background(), os.Args[1:], cwd))
}
