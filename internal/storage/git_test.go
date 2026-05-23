package storage

import (
	"os"
	"os/exec"
	"path/filepath"
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

// gitAvailable returns false when git is not on PATH, allowing tests to skip.
func gitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// isolateGitConfig sets GIT_CONFIG_GLOBAL and GIT_CONFIG_SYSTEM to /dev/null so
// that tests are not affected by commit.gpgsign or any other global git settings.
func isolateGitConfig(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
}

func TestManualCommitEnabled(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}
	isolateGitConfig(t)

	dir := t.TempDir()
	g := newGitHelper(dir)
	if err := g.init(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// write a file so there is something to commit
	if err := os.WriteFile(filepath.Join(dir, "data.yaml"), []byte("test: true\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := g.ManualCommit(); err != nil {
		t.Fatalf("ManualCommit: %v", err)
	}

	// verify a commit exists
	out, err := exec.Command("git", "-C", dir, "log", "--oneline").Output()
	if err != nil {
		t.Fatalf("git log: %v", err)
	}
	if len(out) == 0 {
		t.Error("expected at least one commit after ManualCommit")
	}
}

func TestManualCommitNothingToCommit(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}
	isolateGitConfig(t)

	dir := t.TempDir()
	g := newGitHelper(dir)
	if err := g.init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	// nothing staged — git commit exits non-zero
	err := g.ManualCommit()
	if err == nil {
		t.Error("expected error when there is nothing to commit")
	}
}

func TestPushNoRemote(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}
	isolateGitConfig(t)

	dir := t.TempDir()
	g := newGitHelper(dir)
	if err := g.init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	// push on a repo with no remote should fail
	if err := g.Push(); err == nil {
		t.Error("expected error when pushing with no remote configured")
	}
}

func TestPushWithRemote(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}
	isolateGitConfig(t)

	// create a bare repo to act as the remote
	bareDir := t.TempDir()
	if err := exec.Command("git", "init", "--bare", bareDir).Run(); err != nil {
		t.Fatalf("git init --bare: %v", err)
	}

	// working repo
	dir := t.TempDir()
	g := newGitHelper(dir)
	if err := g.init(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// add remote origin pointing to bare repo
	if err := exec.Command("git", "-C", dir, "remote", "add", "origin", bareDir).Run(); err != nil {
		t.Fatalf("git remote add: %v", err)
	}
	// configure push so it works without an upstream tracking branch
	if err := exec.Command("git", "-C", dir, "config", "push.default", "current").Run(); err != nil {
		t.Fatalf("git config push.default: %v", err)
	}

	// create a commit so there is something to push
	if err := os.WriteFile(filepath.Join(dir, "data.yaml"), []byte("test: true\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := g.ManualCommit(); err != nil {
		t.Fatalf("ManualCommit: %v", err)
	}

	if err := g.Push(); err != nil {
		t.Fatalf("Push: %v", err)
	}

	// verify the bare remote received the commit
	out, err := exec.Command("git", "-C", bareDir, "log", "--oneline").Output()
	if err != nil {
		t.Fatalf("git log on bare: %v", err)
	}
	if len(out) == 0 {
		t.Error("bare remote should have at least one commit after Push")
	}
}
