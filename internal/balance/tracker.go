package balance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/ysaisme/x-switch/internal/adapter"
	"github.com/ysaisme/x-switch/internal/config"
	"github.com/ysaisme/x-switch/internal/store"
)

type ModelPricing struct {
	InputPer1M  float64
	OutputPer1M float64
}

var pricingTable = map[string]ModelPricing{
	"gpt-4o":                    {InputPer1M: 2.5, OutputPer1M: 10.0},
	"gpt-4o-mini":               {InputPer1M: 0.15, OutputPer1M: 0.60},
	"gpt-4-turbo":               {InputPer1M: 10.0, OutputPer1M: 30.0},
	"o1":                        {InputPer1M: 15.0, OutputPer1M: 60.0},
	"o3-mini":                   {InputPer1M: 1.1, OutputPer1M: 4.4},
	"claude-sonnet-4-20250514":  {InputPer1M: 3.0, OutputPer1M: 15.0},
	"claude-haiku-4-20250414":   {InputPer1M: 0.80, OutputPer1M: 4.0},
	"claude-opus-4-20250514":    {InputPer1M: 15.0, OutputPer1M: 75.0},
	"gemini-2.0-flash":          {InputPer1M: 0.10, OutputPer1M: 0.40},
	"gemini-2.5-pro":            {InputPer1M: 1.25, OutputPer1M: 10.0},
}

type BalanceTracker struct {
	mu       sync.RWMutex
	balances map[string]*adapter.BalanceInfo
	cfg      *config.Config
	store    *store.Store
	client   *http.Client
	alerts   []AlertConfig
}

type AlertConfig struct {
	SiteID     string
	Threshold  float64
	Notify     string
	WebhookURL string
}

func NewTracker(cfg *config.Config, s *store.Store) *BalanceTracker {
	return &BalanceTracker{
		balances: make(map[string]*adapter.BalanceInfo),
		cfg:      cfg,
		store:    s,
		client:   &http.Client{Timeout: 15 * time.Second},
	}
}

func (t *BalanceTracker) SetAlerts(alerts []AlertConfig) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.alerts = alerts
}

func (t *BalanceTracker) GetBalance(siteID string) *adapter.BalanceInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.balances[siteID]
}

func (t *BalanceTracker) GetAllBalances() map[string]*adapter.BalanceInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()
	result := make(map[string]*adapter.BalanceInfo)
	for k, v := range t.balances {
		result[k] = v
	}
	return result
}

func (t *BalanceTracker) RefreshAll() {
	for _, site := range t.cfg.Sites {
		if site.BalanceAPI == "" {
			continue
		}
		t.refreshSite(site)
	}
}

func (t *BalanceTracker) refreshSite(site config.Site) {
	req, err := http.NewRequest("GET", site.BalanceAPI, nil)
	if err != nil {
		log.Printf("[xswitch] balance request error for %s: %v", site.ID, err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+site.APIKey)

	resp, err := t.client.Do(req)
	if err != nil {
		log.Printf("[xswitch] balance fetch error for %s: %v", site.ID, err)
		return
	}
	defer resp.Body.Close()

	body := make([]byte, 4096)
	n, _ := resp.Body.Read(body)
	body = body[:n]

	adp := adapter.GetAdapter(site.Protocol)
	info, err := adp.ParseBalance(body)
	if err != nil {
		log.Printf("[xswitch] balance parse error for %s: %v", site.ID, err)
		return
	}

	info.SiteID = site.ID

	t.mu.Lock()
	t.balances[site.ID] = info
	t.mu.Unlock()

	if t.store != nil {
		t.store.InsertBalance(site.ID, info.Balance, info.Currency)
	}

	t.checkAlerts(site.ID, info)
}

func (t *BalanceTracker) checkAlerts(siteID string, info *adapter.BalanceInfo) {
	t.mu.RLock()
	alerts := t.alerts
	t.mu.RUnlock()

	for _, alert := range alerts {
		if alert.SiteID == siteID && info.Balance < alert.Threshold {
			log.Printf("[xswitch] BALANCE ALERT: %s balance %.2f %s below threshold %.2f",
				siteID, info.Balance, info.Currency, alert.Threshold)

			if alert.Notify == "webhook" && alert.WebhookURL != "" {
				t.sendWebhook(alert.WebhookURL, siteID, info)
			}
		}
	}
}

func (t *BalanceTracker) sendWebhook(url string, siteID string, info *adapter.BalanceInfo) {
	payload, _ := json.Marshal(map[string]interface{}{
		"site_id":  siteID,
		"balance":  info.Balance,
		"currency": info.Currency,
		"alert":    "balance_below_threshold",
	})

	resp, err := http.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		log.Printf("[xswitch] webhook error: %v", err)
		return
	}
	resp.Body.Close()
}

func EstimateCost(model string, inputTokens, outputTokens int) float64 {
	pricing, ok := pricingTable[model]
	if !ok {
		return 0
	}

	inputCost := float64(inputTokens) / 1_000_000 * pricing.InputPer1M
	outputCost := float64(outputTokens) / 1_000_000 * pricing.OutputPer1M

	return inputCost + outputCost
}

func GetPricingTable() map[string]ModelPricing {
	return pricingTable
}

func (t *BalanceTracker) StartPeriodicRefresh(interval time.Duration) chan struct{} {
	stop := make(chan struct{})

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		t.RefreshAll()

		for {
			select {
			case <-ticker.C:
				t.RefreshAll()
			case <-stop:
				return
			}
		}
	}()

	return stop
}

func FormatCost(cost float64) string {
	if cost < 0.01 {
		return fmt.Sprintf("$%.4f", cost)
	}
	return fmt.Sprintf("$%.2f", cost)
}
