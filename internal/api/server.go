package api

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/ysaisme/x-switch/internal/config"
	"github.com/ysaisme/x-switch/internal/routing"
	"github.com/ysaisme/x-switch/internal/store"
)

type Server struct {
	router *routing.Router
	store  *store.Store
	mux    *http.ServeMux
	webFS  fs.FS
}

func NewServer(router *routing.Router, s *store.Store, webFS fs.FS) *Server {
	srv := &Server{
		router: router,
		store:  s,
		mux:    http.NewServeMux(),
		webFS:  webFS,
	}
	srv.registerRoutes()
	return srv
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/api/v1/routing/switch", s.handleRoutingSwitch)
	s.mux.HandleFunc("/api/v1/routing/current", s.handleRoutingCurrent)
	s.mux.HandleFunc("/api/v1/profiles", s.handleProfiles)
	s.mux.HandleFunc("/api/v1/profiles/create", s.handleProfileCreate)
	s.mux.HandleFunc("/api/v1/profiles/delete", s.handleProfileDelete)
	s.mux.HandleFunc("/api/v1/profiles/rules/add", s.handleProfileRuleAdd)
	s.mux.HandleFunc("/api/v1/profiles/rules/delete", s.handleProfileRuleDelete)
	s.mux.HandleFunc("/api/v1/sites", s.handleSites)
	s.mux.HandleFunc("/api/v1/sites/add", s.handleSiteAdd)
	s.mux.HandleFunc("/api/v1/sites/update", s.handleSiteUpdate)
	s.mux.HandleFunc("/api/v1/sites/delete", s.handleSiteDelete)
	s.mux.HandleFunc("/api/v1/config", s.handleConfig)
	s.mux.HandleFunc("/api/v1/config/reload", s.handleConfigReload)
	s.mux.HandleFunc("/api/v1/health", s.handleHealth)
	s.mux.HandleFunc("/api/v1/logs", s.handleLogs)
	s.mux.HandleFunc("/api/v1/stats", s.handleStats)
	s.mux.HandleFunc("/", s.handleWebUI)
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) handleRoutingSwitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Profile      string `json:"profile,omitempty"`
		Site         string `json:"site,omitempty"`
		Model        string `json:"model,omitempty"`
		SiteForModel string `json:"site_for_model,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var err error
	switch {
	case req.Profile != "":
		err = s.router.SwitchProfile(req.Profile)
	case req.Site != "":
		err = s.router.SwitchSite(req.Site)
	case req.Model != "" && req.SiteForModel != "":
		err = s.router.SwitchModel(req.Model, req.SiteForModel)
	default:
		writeError(w, http.StatusBadRequest, "must specify profile, site, or model+site_for_model")
		return
	}

	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	cfg := s.router.GetConfig()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":        true,
		"active_profile": cfg.Routing.ActiveProfile,
	})
}

func (s *Server) handleRoutingCurrent(w http.ResponseWriter, r *http.Request) {
	cfg := s.router.GetConfig()
	profile := cfg.GetActiveProfile()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"active_profile": cfg.Routing.ActiveProfile,
		"profile":        profile,
		"sites_count":    len(cfg.Sites),
	})
}

func (s *Server) handleProfiles(w http.ResponseWriter, r *http.Request) {
	cfg := s.router.GetConfig()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"profiles":       cfg.Routing.Profiles,
		"active_profile": cfg.Routing.ActiveProfile,
	})
}

func (s *Server) handleProfileCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	cfg := s.router.GetConfig()
	if cfg.FindProfile(req.Name) != nil {
		writeError(w, http.StatusConflict, fmt.Sprintf("profile %s already exists", req.Name))
		return
	}

	cfg.Routing.Profiles = append(cfg.Routing.Profiles, config.Profile{
		Name:  req.Name,
		Rules: []config.Rule{},
	})
	if err := config.Save(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "save config failed")
		return
	}
	s.router.UpdateConfig(cfg)
	writeJSON(w, http.StatusCreated, map[string]interface{}{"success": true, "profile": req.Name})
}

func (s *Server) handleProfileDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	cfg := s.router.GetConfig()
	if cfg.FindProfile(req.Name) == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("profile %s not found", req.Name))
		return
	}
	if cfg.Routing.ActiveProfile == req.Name {
		writeError(w, http.StatusBadRequest, "cannot delete active profile")
		return
	}
	if len(cfg.Routing.Profiles) <= 1 {
		writeError(w, http.StatusBadRequest, "cannot delete the last profile")
		return
	}

	cfg.DeleteProfile(req.Name)
	if err := config.Save(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "save config failed")
		return
	}
	s.router.UpdateConfig(cfg)
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (s *Server) handleProfileRuleAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Profile      string `json:"profile"`
		ModelPattern string `json:"model_pattern"`
		Site         string `json:"site"`
		Fallback     string `json:"fallback,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Profile == "" || req.ModelPattern == "" || req.Site == "" {
		writeError(w, http.StatusBadRequest, "profile, model_pattern, and site are required")
		return
	}

	cfg := s.router.GetConfig()
	profile := cfg.FindProfile(req.Profile)
	if profile == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("profile %s not found", req.Profile))
		return
	}
	if cfg.FindSite(req.Site) == nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("site %s not found", req.Site))
		return
	}

	profile.Rules = append(profile.Rules, config.Rule{
		ModelPattern: req.ModelPattern,
		Site:         req.Site,
		Fallback:     req.Fallback,
	})
	if err := config.Save(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "save config failed")
		return
	}
	s.router.UpdateConfig(cfg)
	writeJSON(w, http.StatusCreated, map[string]interface{}{"success": true})
}

func (s *Server) handleProfileRuleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Profile string `json:"profile"`
		Index   int    `json:"index"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Profile == "" {
		writeError(w, http.StatusBadRequest, "profile is required")
		return
	}

	cfg := s.router.GetConfig()
	profile := cfg.FindProfile(req.Profile)
	if profile == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("profile %s not found", req.Profile))
		return
	}
	if req.Index < 0 || req.Index >= len(profile.Rules) {
		writeError(w, http.StatusBadRequest, "invalid rule index")
		return
	}

	profile.Rules = append(profile.Rules[:req.Index], profile.Rules[req.Index+1:]...)
	if err := config.Save(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "save config failed")
		return
	}
	s.router.UpdateConfig(cfg)
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (s *Server) handleSites(w http.ResponseWriter, r *http.Request) {
	cfg := s.router.GetConfig()

	sites := cfg.Sites
	if sites == nil {
		sites = []config.Site{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"sites": sites,
	})
}

func (s *Server) handleSiteAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var site config.Site
	if err := json.NewDecoder(r.Body).Decode(&site); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if site.ID == "" || site.BaseURL == "" || site.APIKey == "" {
		writeError(w, http.StatusBadRequest, "id, base_url, and api_key are required")
		return
	}

	cfg := s.router.GetConfig()
	for _, existing := range cfg.Sites {
		if existing.ID == site.ID {
			writeError(w, http.StatusConflict, fmt.Sprintf("site %s already exists", site.ID))
			return
		}
	}

	cfg.Sites = append(cfg.Sites, site)
	if err := config.Save(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "save config failed")
		return
	}

	s.router.UpdateConfig(cfg)
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"site_id": site.ID,
	})
}

func (s *Server) handleSiteUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var site config.Site
	if err := json.NewDecoder(r.Body).Decode(&site); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if site.ID == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	cfg := s.router.GetConfig()
	existing := cfg.FindSite(site.ID)
	if existing == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("site %s not found", site.ID))
		return
	}

	if site.Name == "" {
		site.Name = existing.Name
	}
	if site.BaseURL == "" {
		site.BaseURL = existing.BaseURL
	}
	if site.APIKey == "" {
		site.APIKey = existing.APIKey
	}
	if site.Protocol == "" {
		site.Protocol = existing.Protocol
	}
	if site.Models == nil {
		site.Models = existing.Models
	}

	cfg.UpdateSite(site)
	if err := config.Save(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "save config failed")
		return
	}
	s.router.UpdateConfig(cfg)
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (s *Server) handleSiteDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ID == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	cfg := s.router.GetConfig()
	if cfg.FindSite(req.ID) == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("site %s not found", req.ID))
		return
	}

	cfg.DeleteSite(req.ID)
	if err := config.Save(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "save config failed")
		return
	}
	s.router.UpdateConfig(cfg)
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	cfg := s.router.GetConfig()

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"proxy":    cfg.Proxy,
			"security": cfg.Security,
			"logging":  cfg.Logging,
		})

	case http.MethodPatch:
		var updates map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		cfg := s.router.GetConfig()

		if proxy, ok := updates["proxy"].(map[string]interface{}); ok {
			if listen, ok := proxy["listen"].(string); ok && listen != "" {
				cfg.Proxy.Listen = listen
			}
			if webListen, ok := proxy["web_listen"].(string); ok && webListen != "" {
				cfg.Proxy.WebListen = webListen
			}
		}

		if security, ok := updates["security"].(map[string]interface{}); ok {
			if token, ok := security["access_token"].(string); ok {
				cfg.Security.AccessToken = token
			}
			if ips, ok := security["allowed_ips"].([]interface{}); ok {
				var newIPs []string
				for _, ip := range ips {
					if s, ok := ip.(string); ok {
						newIPs = append(newIPs, s)
					}
				}
				cfg.Security.AllowedIPs = newIPs
			}
			if rl, ok := security["rate_limit"].(map[string]interface{}); ok {
				if rpm, ok := rl["global_rpm"].(float64); ok {
					cfg.Security.RateLimit.GlobalRPM = int(rpm)
				}
			}
		}

		if logging, ok := updates["logging"].(map[string]interface{}); ok {
			if enabled, ok := logging["enabled"].(bool); ok {
				cfg.Logging.Enabled = enabled
			}
			if maxDays, ok := logging["max_days"].(float64); ok {
				cfg.Logging.MaxDays = int(maxDays)
			}
			if logBody, ok := logging["log_body"].(bool); ok {
				cfg.Logging.LogBody = logBody
			}
		}

		if err := config.Save(cfg); err != nil {
			writeError(w, http.StatusInternalServerError, "save config failed")
			return
		}
		s.router.UpdateConfig(cfg)
		writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleConfigReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := s.router.ReloadConfig(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "ok",
	})
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"logs": []interface{}{}})
		return
	}

	filter := store.LogFilter{
		SiteID:     r.URL.Query().Get("site"),
		Model:      r.URL.Query().Get("model"),
		OnlyErrors: r.URL.Query().Get("errors") == "true",
		Limit:      100,
	}

	if days := r.URL.Query().Get("days"); days != "" {
		d, _ := time.ParseDuration(days + "h")
		filter.Since = time.Now().Add(-d * 24)
	}

	logs, err := s.store.QueryLogs(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if logs == nil {
		logs = []store.RequestLog{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"logs": logs,
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{})
		return
	}

	days := 1
	if d := r.URL.Query().Get("days"); d != "" {
		fmt.Sscanf(d, "%d", &days)
	}

	since := time.Now().AddDate(0, 0, -days)
	stats, err := s.store.GetStats(since)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]interface{}{
		"error": msg,
	})
}

func (s *Server) handleWebUI(w http.ResponseWriter, r *http.Request) {
	if s.webFS == nil {
		http.NotFound(w, r)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}

	if _, err := fs.Stat(s.webFS, path); err != nil {
		r.URL.Path = "/"
	}

	http.FileServer(http.FS(s.webFS)).ServeHTTP(w, r)
}
