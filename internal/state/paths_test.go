package state

import (
	"path/filepath"
	"testing"
)

func TestStateDirRespectsXDG(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/tmp/xdg")
	got, err := StateDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/tmp/xdg", "orbit")
	if got != want {
		t.Errorf("StateDir() = %q; want %q", got, want)
	}
}

func TestBarePath(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/tmp/xdg")
	got, err := BarePath("repo-test")
	if err != nil {
		t.Fatal(err)
	}
	want := "/tmp/xdg/orbit/repos/repo-test"
	if got != want {
		t.Errorf("BarePath = %q; want %q", got, want)
	}
}
