package storage

import (
	"testing"
)

func TestGitHelperDisabled(t *testing.T) {
	g := &gitHelper{dataDir: t.TempDir(), enabled: false}

	if err := g.init(); err != nil {
		t.Errorf("init with disabled git: %v", err)
	}
	if err := g.autoCommit("/some/path", "test"); err != nil {
		t.Errorf("autoCommit with disabled git: %v", err)
	}
	if err := g.Push(); err != nil {
		t.Errorf("Push with disabled git: %v", err)
	}
	if err := g.ManualCommit(); err != nil {
		t.Errorf("ManualCommit with disabled git: %v", err)
	}
}

func TestGitHelperInit(t *testing.T) {
	dir := t.TempDir()
	g := newGitHelper(dir)
	// init is idempotent — calling twice should not error
	_ = g.init()
	if err := g.init(); err != nil {
		t.Errorf("second init: %v", err)
	}
}
