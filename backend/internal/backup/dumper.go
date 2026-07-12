package backup

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type DumpResult struct {
	Path string
	Name string
	Size int64
}

type Dumper struct {
	DatabaseURL string
	TmpDir      string
	Now         func() time.Time
}

func (d *Dumper) Dump(ctx context.Context) (DumpResult, error) {
	if d.DatabaseURL == "" {
		return DumpResult{}, fmt.Errorf("database url is empty")
	}
	if err := os.MkdirAll(d.TmpDir, 0o700); err != nil {
		return DumpResult{}, fmt.Errorf("create backup tmp dir: %w", err)
	}

	now := time.Now
	if d.Now != nil {
		now = d.Now
	}

	name := fmt.Sprintf("movies-%s.dump", now().UTC().Format("2006-01-02-150405"))
	path := filepath.Join(d.TmpDir, name)

	cmd := exec.CommandContext(ctx, "pg_dump", "-Fc", "--dbname", d.DatabaseURL, "--file", path)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		_ = os.Remove(path)
		return DumpResult{}, fmt.Errorf("run pg_dump: %w: %s", err, stderr.String())
	}

	stat, err := os.Stat(path)
	if err != nil {
		_ = os.Remove(path)
		return DumpResult{}, fmt.Errorf("stat dump file: %w", err)
	}

	return DumpResult{Path: path, Name: name, Size: stat.Size()}, nil
}
