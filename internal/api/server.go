package api

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/ysaisme/mswitch/internal/config"
	"github.com/ysaisme/mswitch/internal/routing"
	"github.com/ysaisme/mswitch/internal/store"
)

type Server struct {
	router  *routing.Router
	store   *store.Store
	mux     *http.ServeMux
	webFS   fs.FS
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
	s.mux.HandleFunc("/api/v1/sites", s.handleSites)
	s.mux.HandleFunc("/api/v1/sites/add", s.handleSiteAdd)
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

func (s *Server) handleSites(w http.ResponseWriter, r *http.Request) {
	cfg := s.router.GetConfig()

	type SiteView struct {
		ID       string   `json:"id"`
		Name     string   `json:"name"`
		BaseURL  string   `json:"base_url"`
		Protocol string   `json:"protocol"`
		Models   []string `json:"models"`
	}

	var sites []SiteView
	for _, site := range cfg.Sites {
		sites = append(sites, SiteView{
			ID:       site.ID,
			Name:     site.Name,
			BaseURL:  site.BaseURL,
			Protocol: site.Protocol,
			Models:   site.Models,
		})
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
