package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/viper"
)

type GitConfig struct {
	AutoCommit bool `mapstructure:"auto_commit"`
}

type TUIConfig struct {
	ShowDoneTasks bool `mapstructure:"show_done_tasks"`
}

// Profile overrides selected fields from the base config for a named scope.
type Profile struct {
	DataDir  string   `mapstructure:"data_dir"`
	Contexts []string `mapstructure:"contexts"`
}

type Config struct {
	DataDir   string             `mapstructure:"data_dir"`
	Editor    string             `mapstructure:"editor"`
	WeekStart string             `mapstructure:"week_start"`
	Contexts  []string           `mapstructure:"contexts"`
	Git       GitConfig          `mapstructure:"git"`
	TUI       TUIConfig          `mapstructure:"tui"`
	Profiles  map[string]Profile `mapstructure:"profiles"`
}

// ApplyScope merges the named profile into the config, overriding DataDir and
// Contexts. Returns an error if the scope name is not found in Profiles.
func (c *Config) ApplyScope(scope string) error {
	if scope == "" {
		return nil
	}
	p, ok := c.Profiles[scope]
	if !ok {
		return fmt.Errorf("scope %q not defined in config (available: %s)", scope, availableScopes(c.Profiles))
	}
	if p.DataDir != "" {
		c.DataDir = expandHome(p.DataDir)
	}
	if len(p.Contexts) > 0 {
		c.Contexts = p.Contexts
	}
	return nil
}

func availableScopes(profiles map[string]Profile) string {
	if len(profiles) == 0 {
		return "(none)"
	}
	names := make([]string, 0, len(profiles))
	for k := range profiles {
		names = append(names, k)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.config/pitlist")
	viper.AddConfigPath(configDir())
	viper.SetEnvPrefix("PITLIST")
	viper.AutomaticEnv()

	viper.SetDefault("data_dir", defaultDataDir())
	viper.SetDefault("editor", os.Getenv("EDITOR"))
	viper.SetDefault("week_start", "monday")
	viper.SetDefault("git.auto_commit", true)
	viper.SetDefault("tui.show_done_tasks", false)
	viper.SetDefault("contexts", []string{"work", "personal", "other"})

	_ = viper.ReadInConfig()

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	if cfg.DataDir == "" {
		cfg.DataDir = defaultDataDir()
	}
	cfg.DataDir = expandHome(cfg.DataDir)

	return cfg, nil
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "pitlist")
}

func defaultDataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "pitlist")
}

func expandHome(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
