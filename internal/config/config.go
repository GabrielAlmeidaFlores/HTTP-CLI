package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type KeybindingEntry struct {
	Keys        []string `json:"keys"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Visible     bool     `json:"visible"`
	Tab         string   `json:"tab,omitempty"`
}

type HintsConfig struct {
	Enabled          bool   `json:"enabled"`
	Position         string `json:"position"`
	Height           int    `json:"height"`
	ShowDescriptions bool   `json:"show_descriptions"`
	HighlightKeys    bool   `json:"highlight_keys"`
	KeyColor         string `json:"key_color"`
	DescriptionColor string `json:"description_color"`
	Separator        string `json:"separator"`
}

type LayoutConfig struct {
	LeftPanelWidthRatio float64 `json:"left_panel_width_ratio"`
	BorderStyle         string  `json:"border_style"`
	ShowStatusBar       bool    `json:"show_status_bar"`
}

type ThemeConfig struct {
	Primary      string `json:"primary"`
	Secondary    string `json:"secondary"`
	Success      string `json:"success"`
	Error        string `json:"error"`
	Warning      string `json:"warning"`
	FocusBorder  string `json:"focus_border"`
	BlurBorder   string `json:"blur_border"`
	MethodGet    string `json:"method_get"`
	MethodPost   string `json:"method_post"`
	MethodPut    string `json:"method_put"`
	MethodDelete string `json:"method_delete"`
	MethodPatch  string `json:"method_patch"`
}

type UIConfig struct {
	Hints  HintsConfig  `json:"hints"`
	Layout LayoutConfig `json:"layout"`
	Theme  ThemeConfig  `json:"theme"`
}

type RequestDefaults struct {
	TimeoutSeconds  int    `json:"timeout_seconds"`
	FollowRedirects bool   `json:"follow_redirects"`
	VerifySSL       bool   `json:"verify_ssl"`
	UserAgent       string `json:"user_agent"`
}

type StorageConfig struct {
	HistoryLimit int  `json:"history_limit"`
	AutoSave     bool `json:"auto_save"`
}

type DebugConfig struct {
	LogLevel string `json:"log_level"`
	Verbose  bool   `json:"verbose"`
}

type Config struct {
	Version         string                                `json:"version"`
	Keybindings     map[string]map[string]KeybindingEntry `json:"keybindings"`
	UI              UIConfig                              `json:"ui"`
	RequestDefaults RequestDefaults                       `json:"request_defaults"`
	Storage         StorageConfig                         `json:"storage"`
	Debug           DebugConfig                           `json:"debug"`
}

type Manager struct {
	mu        sync.RWMutex
	config    *Config
	listeners []func(*Config)
}

func NewManager() *Manager {
	return &Manager{
		listeners: make([]func(*Config), 0),
	}
}

func (m *Manager) Load(projectConfigPath, userConfigPath string) error {
	base, err := loadFile(projectConfigPath)
	if err != nil {
		return fmt.Errorf("loading project config: %w", err)
	}

	if userConfigPath != "" {
		if user, err := loadFile(userConfigPath); err == nil {
			merge(base, user)
		}
	}

	m.mu.Lock()
	m.config = base
	listeners := make([]func(*Config), len(m.listeners))
	copy(listeners, m.listeners)
	m.mu.Unlock()

	for _, fn := range listeners {
		fn(base)
	}

	return nil
}

func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

func (m *Manager) OnChange(fn func(*Config)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listeners = append(m.listeners, fn)
}

func loadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &cfg, nil
}

func merge(base, override *Config) {
	if override.UI.Hints.Position != "" {
		base.UI.Hints = override.UI.Hints
	}
	if override.UI.Layout.BorderStyle != "" {
		base.UI.Layout = override.UI.Layout
	}
	if override.UI.Theme.Primary != "" {
		base.UI.Theme = override.UI.Theme
	}
	if override.RequestDefaults.UserAgent != "" {
		base.RequestDefaults = override.RequestDefaults
	}
	if override.Debug.LogLevel != "" {
		base.Debug = override.Debug
	}
	for section, bindings := range override.Keybindings {
		if base.Keybindings == nil {
			base.Keybindings = make(map[string]map[string]KeybindingEntry)
		}
		if base.Keybindings[section] == nil {
			base.Keybindings[section] = make(map[string]KeybindingEntry)
		}
		for action, entry := range bindings {
			base.Keybindings[section][action] = entry
		}
	}
}

func DefaultUserConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "http-cli", "config.json")
}
