package cmd

import (
	"fmt"
	"io"
)

func runPageList(out io.Writer, opts *PageListOptions) error {
	cfg, err := loadConfig("")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return RunPageListWithConfig(out, opts, cfg)
}

func runPageGet(out io.Writer, opts *pageGetOptions) error {
	cfg, err := loadConfig("")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return RunPageGetWithConfig(out, opts.PageID, cfg)
}

func runPageCreate(out io.Writer, opts *pageCreateOptions) error {
	cfg, err := loadConfig("")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return RunPageCreateWithConfig(out, opts, cfg)
}

func runPageUpdate(out io.Writer, opts *pageUpdateOptions) error {
	cfg, err := loadConfig("")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return RunPageUpdateWithConfig(out, opts, cfg)
}

func runPageDelete(out io.Writer, opts *pageDeleteOptions) error {
	cfg, err := loadConfig("")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return RunPageDeleteWithConfig(out, opts.PageID, cfg)
}
