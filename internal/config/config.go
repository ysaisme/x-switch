package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Sites    []Site    `yaml:"sites" json:"sites"`
	Routing  Routing   `yaml:"routing" json:"routing"`
	Proxy    Proxy     `yaml:"proxy" json:"proxy"`
	Security Security  `yaml:"security" json:"security"`
	Logging  Logging   `yaml:"logging" json:"logging"`
}

type Site struct {
	ID         string   `yaml:"id" json:"id"`
	Name       string   `yaml:"name" json:"name"`
	BaseURL    string   `yaml:"base_url" json:"base_url"`
	Protocol   string   `yaml:"protocol" json:"protocol"`
	APIKey     string   `yaml:"api_key" json:"api_key"`
	Models     []string `yaml:"models" json:"models"`
	BalanceAPI string   `yaml:"balance_api,omitempty" json:"balance_api,omitempty"`
}

type Routing struct {
	ActiveProfile string    `yaml:"active_profile" json:"active_profile"`
	Profiles      []Profile `yaml:"profiles" json:"profiles"`
}

type Profile struct {
	Name  string `yaml:"name" json:"name"`
	Rules []Rule `yaml:"rules" json:"rules"`
}

type Rule struct {
	ModelPattern string `yaml:"model_pattern" json:"model_pattern"`
	Site         string `yaml:"site" json:"site"`
	Fallback     string `yaml:"fallback,omitempty" json:"fallback,omitempty"`
}

type Proxy struct {
	Listen    string `yaml:"listen" json:"listen"`
	WebListen string `yaml:"web_listen" json:"web_listen"`
}

type Security struct {
	APIKeyEncryption bool      `yaml:"api_key_encryption" json:"api_key_encryption"`
	AccessToken      string    `yaml:"access_token" json:"access_token"`
	AllowedIPs       []string  `yaml:"allowed_ips,omitempty" json:"allowed_ips,omitempty"`
	RateLimit        RateLimit `yaml:"rate_limit" json:"rate_limit"`
}

type RateLimit struct {
	GlobalRPM  int            `yaml:"global_rpm" json:"global_rpm"`
	PerSiteRPM map[string]int `yaml:"per_site_rpm,omitempty" json:"per_site_rpm,omitempty"`
}

type Logging struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	MaxDays int    `yaml:"max_days" json:"max_days"`
	LogBody bool   `yaml:"log_body" json:"log_body"`
}

var (
	instance *Config
	mu       sync.Mutex
)

func saveConfig(cfg *Config) error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	path := ConfigPath()
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

func DefaultConfig() *Config {
	return &Config{
		Sites: []Site{},
		Routing: Routing{
			ActiveProfile: "default",
			Profiles: []Profile{
				{
					Name: "default",
					Rules: []Rule{},
				},
			},
		},
		Proxy: Proxy{
			Listen:    "127.0.0.1:9090",
			WebListen: "127.0.0.1:9091",
		},
		Security: Security{
			APIKeyEncryption: true,
			AccessToken:      "",
			AllowedIPs:       []string{"127.0.0.1"},
			RateLimit: RateLimit{
				GlobalRPM:  60,
				PerSiteRPM: map[string]int{},
			},
		},
		Logging: Logging{
			Enabled: true,
			MaxDays: 30,
			LogBody: false,
		},
	}
}

func ConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".xswitch")
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

func Load() (*Config, error) {
	mu.Lock()
	defer mu.Unlock()

	if instance != nil {
		c := *instance
		return &c, nil
	}

	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig()
			if saveErr := saveConfig(cfg); saveErr != nil {
				return nil, fmt.Errorf("create default config: %w", saveErr)
			}
			instance = cfg
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	instance = &cfg
	return &cfg, nil
}

func Save(cfg *Config) error {
	if err := saveConfig(cfg); err != nil {
		return err
	}
	mu.Lock()
	instance = cfg
	mu.Unlock()
	return nil
}

func Reload() (*Config, error) {
	mu.Lock()
	instance = nil
	mu.Unlock()
	return Load()
}

func (c *Config) FindSite(id string) *Site {
	for i := range c.Sites {
		if c.Sites[i].ID == id {
			return &c.Sites[i]
		}
	}
	return nil
}

func (c *Config) FindSiteForModel(model string) *Site {
	profile := c.GetActiveProfile()
	if profile == nil {
		return nil
	}

	for _, rule := range profile.Rules {
		if matchPattern(rule.ModelPattern, model) {
			return c.FindSite(rule.Site)
		}
	}
	return nil
}

func (c *Config) GetActiveProfile() *Profile {
	for i := range c.Routing.Profiles {
		if c.Routing.Profiles[i].Name == c.Routing.ActiveProfile {
			return &c.Routing.Profiles[i]
		}
	}
	return nil
}

func matchPattern(pattern, model string) bool {
	if pattern == "*" {
		return true
	}
	if pattern == model {
		return true
	}
	if len(pattern) > 1 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		if len(model) >= len(prefix) && model[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func (c *Config) UpdateSite(site Site) {
	for i := range c.Sites {
		if c.Sites[i].ID == site.ID {
			c.Sites[i] = site
			return
		}
	}
}

func (c *Config) DeleteSite(id string) {
	for i := range c.Sites {
		if c.Sites[i].ID == id {
			c.Sites = append(c.Sites[:i], c.Sites[i+1:]...)
			return
		}
	}
}

func (c *Config) FindProfile(name string) *Profile {
	for i := range c.Routing.Profiles {
		if c.Routing.Profiles[i].Name == name {
			return &c.Routing.Profiles[i]
		}
	}
	return nil
}

func (c *Config) DeleteProfile(name string) {
	for i := range c.Routing.Profiles {
		if c.Routing.Profiles[i].Name == name {
			c.Routing.Profiles = append(c.Routing.Profiles[:i], c.Routing.Profiles[i+1:]...)
			return
		}
	}
}
