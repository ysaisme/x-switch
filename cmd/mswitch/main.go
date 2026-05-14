package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/ysaisme/mswitch/internal/api"
	"github.com/ysaisme/mswitch/internal/config"
	"github.com/ysaisme/mswitch/internal/proxy"
	"github.com/ysaisme/mswitch/internal/routing"
	"github.com/ysaisme/mswitch/internal/security"
	"github.com/ysaisme/mswitch/internal/store"
)

var rootCmd = &cobra.Command{
	Use:   "mswitch",
	Short: "Model API hot-switch proxy",
	Long:  "mswitch is a proxy tool that allows hot-switching between different LLM API providers",
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the mswitch proxy server",
	RunE:  runStart,
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the mswitch proxy server",
	RunE:  runStop,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show mswitch proxy status",
	RunE:  runStatus,
}

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show current active configuration",
	RunE:  runCurrent,
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize mswitch configuration",
	RunE:  runInit,
}

var useCmd = &cobra.Command{
	Use:   "use [profile|site <id>|model <model> <site>]",
	Short: "Hot-switch to a different profile, site, or model routing",
	RunE:  runUse,
}

var siteListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured sites",
	RunE:  runSiteList,
}

var siteAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new API site",
	RunE:  runSiteAdd,
}

var siteTestCmd = &cobra.Command{
	Use:   "test [site_id]",
	Short: "Test connectivity to a site",
	RunE:  runSiteTest,
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	RunE:  runProfileList,
}

var profileCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new profile",
	RunE:  runProfileCreate,
}

var balanceCmd = &cobra.Command{
	Use:   "balance [site_id]",
	Short: "Check site balance",
	RunE:  runBalance,
}

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View request logs",
	RunE:  runLogs,
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open config file in editor",
	RunE:  runConfigEdit,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE:  runConfigShow,
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(currentCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(useCmd)
	rootCmd.AddCommand(balanceCmd)
	rootCmd.AddCommand(logsCmd)

	siteCmd := &cobra.Command{Use: "site", Short: "Manage API sites"}
	siteCmd.AddCommand(siteListCmd)
	siteCmd.AddCommand(siteAddCmd)
	siteCmd.AddCommand(siteTestCmd)
	rootCmd.AddCommand(siteCmd)

	profileCmd := &cobra.Command{Use: "profile", Short: "Manage routing profiles"}
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileCreateCmd)
	rootCmd.AddCommand(profileCmd)

	configCmd := &cobra.Command{Use: "config", Short: "Manage configuration"}
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configShowCmd)
	rootCmd.AddCommand(configCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	s, err := store.New(config.ConfigDir())
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer s.Close()

	if cfg.Logging.Enabled && cfg.Logging.MaxDays > 0 {
		s.CleanOldLogs(cfg.Logging.MaxDays)
	}

	router := routing.NewRouter(cfg)
	failover := routing.NewFailoverManager(cfg)
	px := proxy.New(router, s, failover)
	apiServer := api.NewServer(router, s, webUIFS())

	proxyMux := http.NewServeMux()

	authMiddleware := security.NewAuthMiddleware(cfg.Security.AccessToken, cfg.Security.AllowedIPs)
	rateLimiter := security.NewRateLimiter(cfg.Security.RateLimit.GlobalRPM, cfg.Security.RateLimit.PerSiteRPM)

	proxyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isAPIPath(r.URL.Path) {
			apiServer.Handler().ServeHTTP(w, r)
			return
		}
		px.HandleAny(w, r)
	})

	protectedHandler := rateLimiter.Middleware(authMiddleware.Wrap(proxyHandler))
	proxyMux.Handle("/", protectedHandler)

	proxyAddr := cfg.Proxy.Listen
	webAddr := cfg.Proxy.WebListen

	log.Printf("[mswitch] proxy server starting on %s", proxyAddr)
	log.Printf("[mswitch] api server starting on %s", webAddr)
	log.Printf("[mswitch] active profile: %s", cfg.Routing.ActiveProfile)
	log.Printf("[mswitch] configured sites: %d", len(cfg.Sites))
	log.Printf("[mswitch] logging: enabled=%v max_days=%d", cfg.Logging.Enabled, cfg.Logging.MaxDays)

	pidFile := filepath.Join(config.ConfigDir(), "mswitch.pid")
	os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0644)

	go func() {
		if err := http.ListenAndServe(proxyAddr, proxyMux); err != nil {
			log.Fatalf("[mswitch] proxy server error: %v", err)
		}
	}()

	go func() {
		if err := http.ListenAndServe(webAddr, apiServer.Handler()); err != nil {
			log.Fatalf("[mswitch] api server error: %v", err)
		}
	}()

	failoverStop := failover.StartHealthCheck()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	close(failoverStop)
	os.Remove(pidFile)
	log.Println("[mswitch] shutting down...")
	return nil
}

func isAPIPath(path string) bool {
	return len(path) >= 5 && path[:5] == "/api/"
}

func runUse(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage:")
		fmt.Println("  mswitch use <profile>              - Switch to a profile")
		fmt.Println("  mswitch use site <site_id>         - Route all to a site")
		fmt.Println("  mswitch use model <model> <site>   - Route a model to a site")
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	router := routing.NewRouter(cfg)

	switch args[0] {
	case "site":
		if len(args) < 2 {
			return fmt.Errorf("usage: mswitch use site <site_id>")
		}
		if err := router.SwitchSite(args[1]); err != nil {
			return err
		}
		fmt.Printf("Switched: all models -> %s\n", args[1])

	case "model":
		if len(args) < 3 {
			return fmt.Errorf("usage: mswitch use model <model> <site_id>")
		}
		if err := router.SwitchModel(args[1], args[2]); err != nil {
			return err
		}
		fmt.Printf("Switched: %s -> %s\n", args[1], args[2])

	default:
		if err := router.SwitchProfile(args[0]); err != nil {
			return err
		}
		fmt.Printf("Switched to profile: %s\n", args[0])
	}

	if isRunning() {
		notifyReload()
	}

	return nil
}

func runStop(cmd *cobra.Command, args []string) error {
	pidFile := filepath.Join(config.ConfigDir(), "mswitch.pid")
	data, err := os.ReadFile(pidFile)
	if err != nil {
		fmt.Println("mswitch is not running (pid file not found)")
		return nil
	}

	var pid int
	fmt.Sscanf(string(data), "%d", &pid)

	proc, err := os.FindProcess(pid)
	if err != nil {
		fmt.Println("mswitch is not running")
		return nil
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		fmt.Println("failed to stop mswitch:", err)
		return nil
	}

	os.Remove(pidFile)
	fmt.Println("mswitch stopped")
	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	pidFile := filepath.Join(config.ConfigDir(), "mswitch.pid")
	data, err := os.ReadFile(pidFile)
	if err != nil {
		fmt.Println("mswitch is not running")
		return nil
	}

	var pid int
	fmt.Sscanf(string(data), "%d", &pid)

	proc, err := os.FindProcess(pid)
	if err != nil {
		fmt.Println("mswitch is not running")
		return nil
	}

	if err := proc.Signal(syscall.Signal(0)); err != nil {
		fmt.Println("mswitch is not running (stale pid file)")
		os.Remove(pidFile)
		return nil
	}

	cfg, _ := config.Load()
	fmt.Printf("mswitch is running (PID: %d)\n", pid)
	fmt.Printf("  proxy:     %s\n", cfg.Proxy.Listen)
	fmt.Printf("  web ui:    %s\n", cfg.Proxy.WebListen)
	fmt.Printf("  profile:   %s\n", cfg.Routing.ActiveProfile)
	fmt.Printf("  sites:     %d\n", len(cfg.Sites))
	return nil
}

func runCurrent(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	fmt.Printf("Active profile: %s\n\n", cfg.Routing.ActiveProfile)

	profile := cfg.GetActiveProfile()
	if profile == nil {
		fmt.Println("No active profile found")
		return nil
	}

	fmt.Println("Routing rules:")
	for _, rule := range profile.Rules {
		site := cfg.FindSite(rule.Site)
		siteName := rule.Site
		if site != nil {
			siteName = site.Name
		}
		fallback := ""
		if rule.Fallback != "" {
			fallback = fmt.Sprintf(" (fallback: %s)", rule.Fallback)
		}
		fmt.Printf("  %s -> %s%s\n", rule.ModelPattern, siteName, fallback)
	}

	return nil
}

func runSiteList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if len(cfg.Sites) == 0 {
		fmt.Println("No sites configured. Run 'mswitch init' to get started.")
		return nil
	}

	for _, site := range cfg.Sites {
		fmt.Printf("  %-20s %-15s %s\n", site.ID, site.Protocol, site.Name)
		for _, m := range site.Models {
			fmt.Printf("    - %s\n", m)
		}
	}

	return nil
}

func runSiteAdd(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	fmt.Println("Add a new API site:")
	fmt.Println()

	var id, name, baseURL, apiKey, modelsStr, protocol string

	fmt.Print("Site ID (e.g. openai-official): ")
	fmt.Scanln(&id)

	fmt.Print("Site name (e.g. OpenAI Official): ")
	fmt.Scanln(&name)

	fmt.Print("Base URL (e.g. https://api.openai.com): ")
	fmt.Scanln(&baseURL)

	fmt.Print("Protocol (openai/anthropic/gemini) [openai]: ")
	fmt.Scanln(&protocol)
	if protocol == "" {
		protocol = "openai"
	}

	fmt.Print("API Key: ")
	fmt.Scanln(&apiKey)

	fmt.Print("Models (comma-separated, e.g. gpt-4o,gpt-4o-mini): ")
	fmt.Scanln(&modelsStr)

	models := splitComma(modelsStr)

	for _, existing := range cfg.Sites {
		if existing.ID == id {
			return fmt.Errorf("site %s already exists", id)
		}
	}

	cfg.Sites = append(cfg.Sites, config.Site{
		ID:       id,
		Name:     name,
		BaseURL:  baseURL,
		Protocol: protocol,
		APIKey:   apiKey,
		Models:   models,
	})

	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Printf("\nSite '%s' added successfully.\n", name)
	return nil
}

func runSiteTest(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mswitch site test <site_id>")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	site := cfg.FindSite(args[0])
	if site == nil {
		return fmt.Errorf("site %s not found", args[0])
	}

	fmt.Printf("Testing connection to %s (%s)...\n", site.Name, site.BaseURL)

	client := &http.Client{Timeout: 10}
	resp, err := client.Get(site.BaseURL)
	if err != nil {
		fmt.Printf("  FAILED: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	fmt.Printf("  Status: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Printf("  OK\n")
	return nil
}

func runProfileList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	for _, p := range cfg.Routing.Profiles {
		active := ""
		if p.Name == cfg.Routing.ActiveProfile {
			active = " (active)"
		}
		fmt.Printf("  %s%s - %d rules\n", p.Name, active, len(p.Rules))
		for _, rule := range p.Rules {
			fmt.Printf("    %s -> %s\n", rule.ModelPattern, rule.Site)
		}
	}

	return nil
}

func runProfileCreate(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mswitch profile create <name>")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	for _, p := range cfg.Routing.Profiles {
		if p.Name == args[0] {
			return fmt.Errorf("profile %s already exists", args[0])
		}
	}

	cfg.Routing.Profiles = append(cfg.Routing.Profiles, config.Profile{
		Name:  args[0],
		Rules: []config.Rule{},
	})

	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Printf("Profile '%s' created.\n", args[0])
	return nil
}

func runBalance(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	sites := cfg.Sites
	if len(args) > 0 {
		site := cfg.FindSite(args[0])
		if site == nil {
			return fmt.Errorf("site %s not found", args[0])
		}
		sites = []config.Site{*site}
	}

	for _, site := range sites {
		if site.BalanceAPI == "" {
			fmt.Printf("  %-20s  (no balance API configured)\n", site.ID)
			continue
		}
		fmt.Printf("  %-20s  checking...\n", site.ID)
	}

	return nil
}

func runLogs(cmd *cobra.Command, args []string) error {
	s, err := store.New(config.ConfigDir())
	if err != nil {
		return err
	}
	defer s.Close()

	filter := store.LogFilter{Limit: 50, Since: time.Now().AddDate(0, 0, -1)}

	logs, err := s.QueryLogs(filter)
	if err != nil {
		return err
	}

	if len(logs) == 0 {
		fmt.Println("No recent logs.")
		return nil
	}

	for _, l := range logs {
		stream := ""
		if l.IsStream {
			stream = " [stream]"
		}
		errStr := ""
		if l.Error != "" {
			errStr = fmt.Sprintf(" ERROR: %s", l.Error)
		}
		fmt.Printf("  %s  %-15s %-25s %dms %d/%d tokens%s%s\n",
			l.Timestamp.Format("15:04:05"), l.SiteID, l.Model,
			l.LatencyMs, l.InputTokens, l.OutputTokens, stream, errStr)
	}

	return nil
}

func runConfigEdit(cmd *cobra.Command, args []string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	cfgPath := config.ConfigPath()
	fmt.Printf("Opening %s with %s\n", cfgPath, editor)

	ecmd := exec.Command(editor, cfgPath)
	ecmd.Stdin = os.Stdin
	ecmd.Stdout = os.Stdout
	ecmd.Stderr = os.Stderr
	return ecmd.Run()
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	for _, site := range cfg.Sites {
		maskedKey := maskKey(site.APIKey)
		fmt.Printf("  %-20s  %s  key=%s  models=%v\n", site.ID, site.BaseURL, maskedKey, site.Models)
	}

	fmt.Printf("\nActive profile: %s\n", cfg.Routing.ActiveProfile)
	fmt.Printf("Proxy: %s\n", cfg.Proxy.Listen)
	fmt.Printf("Web:   %s\n", cfg.Proxy.WebListen)
	return nil
}

func runInit(cmd *cobra.Command, args []string) error {
	cfgPath := config.ConfigPath()
	if _, err := os.Stat(cfgPath); err == nil {
		fmt.Printf("Config already exists at %s\n", cfgPath)
		return nil
	}

	cfg := config.DefaultConfig()

	fmt.Println("Welcome to mswitch! Let's set up your first API site.")
	fmt.Println()

	var id, name, baseURL, apiKey, modelsStr string

	fmt.Print("Site ID (e.g. openai-official): ")
	fmt.Scanln(&id)

	fmt.Print("Site name (e.g. OpenAI Official): ")
	fmt.Scanln(&name)

	fmt.Print("Base URL (e.g. https://api.openai.com): ")
	fmt.Scanln(&baseURL)

	fmt.Print("API Key: ")
	fmt.Scanln(&apiKey)

	fmt.Print("Models (comma-separated, e.g. gpt-4o,gpt-4o-mini): ")
	fmt.Scanln(&modelsStr)

	models := splitComma(modelsStr)

	cfg.Sites = append(cfg.Sites, config.Site{
		ID:       id,
		Name:     name,
		BaseURL:  baseURL,
		Protocol: "openai",
		APIKey:   apiKey,
		Models:   models,
	})

	cfg.Routing.Profiles[0].Rules = append(cfg.Routing.Profiles[0].Rules, config.Rule{
		ModelPattern: "*",
		Site:         id,
	})

	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Printf("\nConfig saved to %s\n", cfgPath)
	fmt.Printf("Run 'mswitch start' to start the proxy server.\n")
	return nil
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func isRunning() bool {
	pidFile := filepath.Join(config.ConfigDir(), "mswitch.pid")
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return false
	}

	var pid int
	fmt.Sscanf(string(data), "%d", &pid)

	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	return proc.Signal(syscall.Signal(0)) == nil
}

func notifyReload() {
	cfg, _ := config.Load()
	webAddr := cfg.Proxy.WebListen
	url := fmt.Sprintf("http://%s/api/v1/config/reload", webAddr)

	body, _ := json.Marshal(map[string]bool{"reload": true})
	http.Post(url, "application/json", bytes.NewReader(body))
}

func splitComma(s string) []string {
	var result []string
	for _, v := range splitByRune(s, ',') {
		v = trimSpace(v)
		if v != "" {
			result = append(result, v)
		}
	}
	return result
}

func splitByRune(s string, r rune) []string {
	var result []string
	start := 0
	for i, c := range s {
		if c == r {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func webUIFS() fs.FS {
	sub, err := fs.Sub(webFS, "web_dist")
	if err != nil {
		log.Printf("[mswitch] warning: web ui assets not available: %v", err)
		return nil
	}
	return sub
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
