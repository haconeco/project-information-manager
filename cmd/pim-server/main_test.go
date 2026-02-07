package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/haconeco/project-information-manager/internal/config"
)

func TestEnsureDataDirs(t *testing.T) {
	baseDir := t.TempDir()
	cfg := &config.Config{DataDir: baseDir}

	if err := ensureDataDirs(cfg); err != nil {
		t.Fatalf("ensureDataDirs failed: %v", err)
	}

	paths := []string{
		baseDir,
		filepath.Join(baseDir, "stocks"),
	}

	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			t.Fatalf("expected path to exist: %s: %v", p, err)
		}
		if !info.IsDir() {
			t.Fatalf("expected directory: %s", p)
		}
	}
}

func TestEnsureDataDirsFailsWhenDataDirIsFile(t *testing.T) {
	baseDir := t.TempDir()
	filePath := filepath.Join(baseDir, "not-a-dir")
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	cfg := &config.Config{DataDir: filePath}
	if err := ensureDataDirs(cfg); err == nil {
		t.Fatalf("expected error when data dir is a file")
	}
}
