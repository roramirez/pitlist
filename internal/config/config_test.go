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

func TestApplyScopeEmpty(t *testing.T) {
	cfg := &Config{DataDir: "/base", Contexts: []string{"work"}}
	if err := cfg.ApplyScope(""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DataDir != "/base" {
		t.Errorf("DataDir changed unexpectedly: %v", cfg.DataDir)
	}
}

func TestApplyScopeUnknown(t *testing.T) {
	cfg := &Config{Profiles: map[string]Profile{"work": {DataDir: "/work"}}}
	err := cfg.ApplyScope("personal")
	if err == nil {
		t.Fatal("expected error for unknown scope")
	}
	if !strings.Contains(err.Error(), "work") {
		t.Errorf("error should list available scopes, got: %v", err)
	}
}

func TestApplyScopeOverridesDataDir(t *testing.T) {
	cfg := &Config{
		DataDir:  "/base",
		Contexts: []string{"all"},
		Profiles: map[string]Profile{
			"work": {DataDir: "/work"},
		},
	}
	if err := cfg.ApplyScope("work"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DataDir != "/work" {
		t.Errorf("DataDir = %q, want /work", cfg.DataDir)
	}
	if len(cfg.Contexts) != 1 || cfg.Contexts[0] != "all" {
		t.Errorf("Contexts should be unchanged, got %v", cfg.Contexts)
	}
}

func TestApplyScopeOverridesContexts(t *testing.T) {
	cfg := &Config{
		DataDir:  "/base",
		Contexts: []string{"all"},
		Profiles: map[string]Profile{
			"work": {Contexts: []string{"work", "meetings"}},
		},
	}
	if err := cfg.ApplyScope("work"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Contexts) != 2 || cfg.Contexts[0] != "work" {
		t.Errorf("Contexts = %v, want [work meetings]", cfg.Contexts)
	}
	if cfg.DataDir != "/base" {
		t.Errorf("DataDir changed unexpectedly: %v", cfg.DataDir)
	}
}

func TestApplyScopeNoProfiles(t *testing.T) {
	cfg := &Config{}
	err := cfg.ApplyScope("work")
	if err == nil {
		t.Fatal("expected error when no profiles defined")
	}
	if !strings.Contains(err.Error(), "(none)") {
		t.Errorf("error should mention no profiles, got: %v", err)
	}
}
