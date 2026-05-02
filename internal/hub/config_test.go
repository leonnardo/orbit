package hub

import (
	"path/filepath"
	"testing"
	"time"
)

func TestWriteAndRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ConfigFilename)

	in := &Config{
		Version:   1,
		Project:   "repo-test",
		Remote:    "https://github.com/x/repo-test",
		CreatedAt: time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
	}
	if err := Write(path, in); err != nil {
		t.Fatal(err)
	}

	out, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}

	if out.Version != in.Version || out.Project != in.Project || out.Remote != in.Remote {
		t.Errorf("roundtrip mismatch: in=%+v out=%+v", in, out)
	}
	if !out.CreatedAt.Equal(in.CreatedAt) {
		t.Errorf("CreatedAt mismatch: in=%v out=%v", in.CreatedAt, out.CreatedAt)
	}
}

func TestReadMissingFile(t *testing.T) {
	_, err := Read(filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
