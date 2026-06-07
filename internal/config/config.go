// Package config handles persistent configuration for KnowsWell.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// PromptPreset is a named system-prompt string that can be saved and reused.
type PromptPreset struct {
	Name string `json:"name"`
	Text string `json:"text"`
}

// Config holds all user-tunable settings for the application.
type Config struct {
	APIEndpoint  string         `json:"api_endpoint"`
	APIKey       string         `json:"api_key"`
	Model        string         `json:"model"`
	MaxTokens    int            `json:"max_tokens"`
	ToolbarX     int            `json:"toolbar_x"`
	ToolbarY     int            `json:"toolbar_y"`
	SystemPrompt string         `json:"system_prompt"`
	GhostMode    bool           `json:"ghost_mode"`
	Opacity      int            `json:"opacity"` // 10–100, default 100
	Prompts      []PromptPreset `json:"prompts"`
}

func defaultConfig() *Config {
	return &Config{
		APIEndpoint: "https://api.openai.com/v1/chat/completions",
		APIKey:      os.Getenv("OPENAI_API_KEY"),
		Model:       "gpt-4o",
		MaxTokens:   4096,
		ToolbarX:    200,
		ToolbarY:    8,
		GhostMode:   true,
		Opacity:     100,
		Prompts: []PromptPreset{
			{Name: "Assistant", Text: "You are a helpful assistant."},
		},
	}
}

func configPath() string {
	exe, err := os.Executable()
	if err != nil {
		wd, _ := os.Getwd()
		return filepath.Join(wd, "knowswell.json")
	}
	return filepath.Join(filepath.Dir(exe), "knowswell.json")
}

// Load reads the config from disk and merges it with defaults.
func Load() *Config {
	cfg := defaultConfig()

	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return cfg
	}

	if envKey := os.Getenv("OPENAI_API_KEY"); envKey != "" {
		cfg.APIKey = envKey
	}
	if envModel := os.Getenv("OPENAI_MODEL"); envModel != "" {
		cfg.Model = envModel
	}
	if len(cfg.Prompts) == 0 {
		cfg.Prompts = defaultConfig().Prompts
	}
	if cfg.Opacity < 10 || cfg.Opacity > 100 {
		cfg.Opacity = 100
	}
	return cfg
}

// Save writes the config to disk, pretty-printed for human readability.
func (c *Config) Save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0644)
}
