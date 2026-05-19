package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type GitConfig struct {
	AutoCommit bool `mapstructure:"auto_commit"`
}

type TUIConfig struct {
	ShowDoneTasks bool   `mapstructure:"show_done_tasks"`
	Pager         string `mapstructure:"pager"`
}

type Config struct {
	DataDir   string    `mapstructure:"data_dir"`
	Editor    string    `mapstructure:"editor"`
	WeekStart string    `mapstructure:"week_start"`
	Contexts  []string  `mapstructure:"contexts"`
	Git       GitConfig `mapstructure:"git"`
	TUI       TUIConfig `mapstructure:"tui"`
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
	viper.SetDefault("tui.pager", "day")
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
