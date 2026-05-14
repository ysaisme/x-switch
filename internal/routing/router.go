package routing

import (
	"sync"

	"github.com/ysaisme/mswitch/internal/config"
)

type Router struct {
	mu     sync.RWMutex
	cfg    *config.Config
}

func NewRouter(cfg *config.Config) *Router {
	return &Router{cfg: cfg}
}

func (r *Router) GetConfig() *config.Config {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.cfg
}

func (r *Router) SwitchProfile(profileName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	found := false
	for _, p := range r.cfg.Routing.Profiles {
		if p.Name == profileName {
			found = true
			break
		}
	}
	if !found {
		return ErrProfileNotFound
	}

	r.cfg.Routing.ActiveProfile = profileName
	return config.Save(r.cfg)
}

func (r *Router) SwitchSite(siteID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	site := r.cfg.FindSite(siteID)
	if site == nil {
		return ErrSiteNotFound
	}

	profile := r.cfg.GetActiveProfile()
	if profile == nil {
		return ErrNoActiveProfile
	}

	profile.Rules = []config.Rule{
		{
			ModelPattern: "*",
			Site:         siteID,
		},
	}

	return config.Save(r.cfg)
}

func (r *Router) SwitchModel(model, siteID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	site := r.cfg.FindSite(siteID)
	if site == nil {
		return ErrSiteNotFound
	}

	profile := r.cfg.GetActiveProfile()
	if profile == nil {
		return ErrNoActiveProfile
	}

	replaced := false
	for i, rule := range profile.Rules {
		if matchModelPattern(rule.ModelPattern, model) {
			profile.Rules[i].Site = siteID
			replaced = true
			break
		}
	}

	if !replaced {
		newRules := make([]config.Rule, 0, len(profile.Rules)+1)
		newRules = append(newRules, config.Rule{
			ModelPattern: model,
			Site:         siteID,
		})
		newRules = append(newRules, profile.Rules...)
		profile.Rules = newRules
	}

	return config.Save(r.cfg)
}

func (r *Router) ReloadConfig() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	r.cfg = cfg
	return nil
}

func (r *Router) UpdateConfig(cfg *config.Config) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cfg = cfg
}

func (r *Router) FindSiteForModel(model string) *config.Site {
	r.mu.RLock()
	defer r.mu.RUnlock()

	profile := r.cfg.GetActiveProfile()
	if profile == nil {
		return nil
	}

	for _, rule := range profile.Rules {
		if matchModelPattern(rule.ModelPattern, model) {
			return r.cfg.FindSite(rule.Site)
		}
	}
	return nil
}

func matchModelPattern(pattern, model string) bool {
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
