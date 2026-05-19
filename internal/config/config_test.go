package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandHomeWithTilde(t *testing.T) {
	home, _ := os.UserHomeDir()
	got := expandHome("~/pitlist")
	want := filepath.Join(home, "pitlist")
	if got != want {
		t.Errorf("expandHome(~/pitlist) = %q, want %q", got, want)
	}
}

func TestExpandHomeWithoutTilde(t *testing.T) {
	got := expandHome("/absolute/path")
	if got != "/absolute/path" {
		t.Errorf("expandHome(/absolute/path) = %q, want unchanged", got)
	}
}

func TestExpandHomeEmpty(t *testing.T) {
	got := expandHome("")
	if got != "" {
		t.Errorf("expandHome('') = %q, want empty string", got)
	}
}

func TestDefaultDataDir(t *testing.T) {
	home, _ := os.UserHomeDir()
	got := defaultDataDir()
	if !strings.HasPrefix(got, home) {
		t.Errorf("defaultDataDir() = %q, expected prefix %q", got, home)
	}
}

func TestConfigDir(t *testing.T) {
	home, _ := os.UserHomeDir()
	got := configDir()
	if !strings.HasPrefix(got, home) {
		t.Errorf("configDir() = %q, expected prefix %q", got, home)
	}
	if !strings.Contains(got, "pitlist") {
		t.Errorf("configDir() = %q, expected to contain 'pitlist'", got)
	}
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DataDir == "" {
		t.Error("DataDir should not be empty")
	}
	if cfg.WeekStart != "monday" {
		t.Errorf("WeekStart default = %q, want 'monday'", cfg.WeekStart)
	}
	if !cfg.Git.AutoCommit {
		t.Error("Git.AutoCommit default should be true")
	}
	if cfg.TUI.Pager != "day" {
		t.Errorf("TUI.Pager default = %q, want 'day'", cfg.TUI.Pager)
	}
	if cfg.TUI.ShowDoneTasks {
		t.Error("TUI.ShowDoneTasks default should be false")
	}
}

func TestLoadEnvOverride(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("PITLIST_DATA_DIR", tmpDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DataDir != tmpDir {
		t.Errorf("DataDir from env = %q, want %q", cfg.DataDir, tmpDir)
	}
}

func TestLoadEmptyDataDirFallsBackToDefault(t *testing.T) {
	// Setting PITLIST_DATA_DIR="" causes viper to emit an empty string,
	// triggering the cfg.DataDir == "" fallback in Load().
	t.Setenv("PITLIST_DATA_DIR", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DataDir == "" {
		t.Error("DataDir should have been set to defaultDataDir() as fallback")
	}
}
