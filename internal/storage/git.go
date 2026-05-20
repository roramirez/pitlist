package storage

import (
	"os"
	"os/exec"
	"path/filepath"
)

type gitHelper struct {
	dataDir string
	enabled bool
}

// GitEnv returns the environment variables required for git commits.
func GitEnv() []string {
	return append(os.Environ(),
		"GIT_AUTHOR_NAME=pitlist",
		"GIT_AUTHOR_EMAIL=pitlist@local",
		"GIT_COMMITTER_NAME=pitlist",
		"GIT_COMMITTER_EMAIL=pitlist@local",
	)
}

func newGitHelper(dataDir string) *gitHelper {
	_, err := exec.LookPath("git")
	return &gitHelper{dataDir: dataDir, enabled: err == nil}
}

func (g *gitHelper) init() error {
	if !g.enabled {
		return nil
	}
	gitDir := filepath.Join(g.dataDir, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		return nil
	}
	cmd := exec.Command("git", "init", g.dataDir)
	return cmd.Run()
}

func (g *gitHelper) autoCommit(path, message string) error {
	if !g.enabled {
		return nil
	}
	add := exec.Command("git", "-C", g.dataDir, "add", path)
	if err := add.Run(); err != nil {
		return err
	}
	commit := exec.Command("git", "-C", g.dataDir, "commit", "-m", message)
	commit.Env = GitEnv()
	return commit.Run()
}

func (g *gitHelper) Push() error {
	if !g.enabled {
		return nil
	}
	cmd := exec.Command("git", "-C", g.dataDir, "push")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (g *gitHelper) ManualCommit() error {
	if !g.enabled {
		return nil
	}
	add := exec.Command("git", "-C", g.dataDir, "add", ".")
	if err := add.Run(); err != nil {
		return err
	}
	commit := exec.Command("git", "-C", g.dataDir, "commit", "-m", "pitlist: manual sync")
	commit.Env = GitEnv()
	return commit.Run()
}
