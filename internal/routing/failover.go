package routing

import (
	"log"
	"sync"
	"time"

	"github.com/ysaisme/x-switch/internal/config"
)

type SiteHealth struct {
	SiteID     string
	Model      string
	Healthy    bool
	LastCheck  time.Time
	FailCount  int
	RecoverAt  time.Time
}

type FailoverManager struct {
	mu      sync.RWMutex
	health map[string]*SiteHealth
	cfg     *config.Config

	MaxFailCount    int
	RecoveryWait    time.Duration
	CheckInterval   time.Duration
}

func NewFailoverManager(cfg *config.Config) *FailoverManager {
	return &FailoverManager{
		health:       make(map[string]*SiteHealth),
		cfg:          cfg,
		MaxFailCount: 3,
		RecoveryWait: 30 * time.Second,
		CheckInterval: 10 * time.Second,
	}
}

func (f *FailoverManager) key(siteID, model string) string {
	return siteID + ":" + model
}

func (f *FailoverManager) RecordSuccess(siteID, model string) {
	k := f.key(siteID, model)
	f.mu.Lock()
	defer f.mu.Unlock()

	if h, ok := f.health[k]; ok {
		h.FailCount = 0
		h.Healthy = true
		h.LastCheck = time.Now()
	}
}

func (f *FailoverManager) RecordFailure(siteID, model string) {
	k := f.key(siteID, model)
	f.mu.Lock()
	defer f.mu.Unlock()

	h, ok := f.health[k]
	if !ok {
		h = &SiteHealth{
			SiteID:  siteID,
			Model:   model,
			Healthy: true,
		}
		f.health[k] = h
	}

	h.FailCount++
	h.LastCheck = time.Now()

	if h.FailCount >= f.MaxFailCount {
		h.Healthy = false
		h.RecoverAt = time.Now().Add(f.RecoveryWait)
		log.Printf("[xswitch] FAILOVER: %s/%s marked as degraded (fail_count=%d)", siteID, model, h.FailCount)
	}
}

func (f *FailoverManager) IsHealthy(siteID, model string) bool {
	k := f.key(siteID, model)
	f.mu.RLock()
	defer f.mu.RUnlock()

	h, ok := f.health[k]
	if !ok {
		return true
	}

	if !h.Healthy {
		if time.Now().After(h.RecoverAt) {
			return true
		}
		return false
	}

	return true
}

func (f *FailoverManager) FindFallback(model string, cfg *config.Config) *config.Site {
	profile := cfg.GetActiveProfile()
	if profile == nil {
		return nil
	}

	for _, rule := range profile.Rules {
		if matchModelPattern(rule.ModelPattern, model) && rule.Fallback != "" {
			if f.IsHealthy(rule.Fallback, model) {
				site := cfg.FindSite(rule.Fallback)
				if site != nil {
					log.Printf("[xswitch] FAILOVER: %s -> fallback %s for model %s", rule.Site, rule.Fallback, model)
					return site
				}
			}
		}
	}

	return nil
}

func (f *FailoverManager) GetHealthStatus() []SiteHealth {
	f.mu.RLock()
	defer f.mu.RUnlock()

	result := make([]SiteHealth, 0, len(f.health))
	for _, h := range f.health {
		result = append(result, *h)
	}
	return result
}

func (f *FailoverManager) TryRecover() {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := time.Now()
	for k, h := range f.health {
		if !h.Healthy && now.After(h.RecoverAt) {
			h.Healthy = true
			h.FailCount = 0
			log.Printf("[xswitch] FAILOVER: %s recovered", k)
		}
	}
}

func (f *FailoverManager) StartHealthCheck() chan struct{} {
	stop := make(chan struct{})

	go func() {
		ticker := time.NewTicker(f.CheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				f.TryRecover()
			case <-stop:
				return
			}
		}
	}()

	return stop
}
