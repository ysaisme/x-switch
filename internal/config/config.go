package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Sites    []Site    `yaml:"sites"`
	Routing  Routing   `yaml:"routing"`
	Proxy    Proxy     `yaml:"proxy"`
	Security Security  `yaml:"security"`
	Logging  Logging   `yaml:"logging"`
}

type Site struct {
	ID        string   `yaml:"id"`
	Name      string   `yaml:"name"`
	BaseURL   string   `yaml:"base_url"`
	Protocol  string   `yaml:"protocol"`
	APIKey    string   `yaml:"api_key"`
	Models    []string `yaml:"models"`
	BalanceAPI string  `yaml:"balance_api,omitempty"`
}

type Routing struct {
	ActiveProfile string     `yaml:"active_profile"`
	Profiles      []Profile  `yaml:"profiles"`
}

type Profile struct {
	Name  string `yaml:"name"`
	Rules []Rule `yaml:"rules"`
}

type Rule struct {
	ModelPattern string `yaml:"model_pattern"`
	Site         string `yaml:"site"`
	Fallback     string `yaml:"fallback,omitempty"`
}

type Proxy struct {
	Listen    string `yaml:"listen"`
	WebListen string `yaml:"web_listen"`
}

type Security struct {
	APIKeyEncryption bool      `yaml:"api_key_encryption"`
	AccessToken      string    `yaml:"access_token"`
	AllowedIPs       []string  `yaml:"allowed_ips,omitempty"`
	RateLimit        RateLimit `yaml:"rate_limit"`
}

type RateLimit struct {
	GlobalRPM  int            `yaml:"global_rpm"`
	PerSiteRPM map[string]int `yaml:"per_site_rpm,omitempty"`
}

type Logging struct {
	Enabled bool   `yaml:"enabled"`
	MaxDays int    `yaml:"max_days"`
	LogBody bool   `yaml:"log_body"`
}

var (
	instance *Config
	once     sync.RWMutex
)

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
	return filepath.Join(home, ".mswitch")
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

func Load() (*Config, error) {
	once.RLock()
	if instance != nil {
		c := *instance
		once.RUnlock()
		return &c, nil
	}
	once.RUnlock()

	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig()
			if saveErr := Save(cfg); saveErr != nil {
				return nil, fmt.Errorf("create default config: %w", saveErr)
			}
			once.Lock()
			instance = cfg
			once.Unlock()
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	once.Lock()
	instance = &cfg
	once.Unlock()
	return &cfg, nil
}

func Save(cfg *Config) error {
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

	once.Lock()
	instance = cfg
	once.Unlock()
	return nil
}

func Reload() (*Config, error) {
	once.Lock()
	instance = nil
	once.Unlock()
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
