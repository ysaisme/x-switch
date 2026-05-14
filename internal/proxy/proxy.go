package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ysaisme/mswitch/internal/adapter"
	"github.com/ysaisme/mswitch/internal/routing"
	"github.com/ysaisme/mswitch/internal/store"
)

type ChatRequest struct {
	Model    string          `json:"model"`
	Messages json.RawMessage `json:"messages"`
	Stream   bool            `json:"stream,omitempty"`
}

type Proxy struct {
	router   *routing.Router
	failover *routing.FailoverManager
	client   *http.Client
	store    *store.Store
	logCh    chan *store.RequestLog
	reqCount atomic.Int64
}

func New(router *routing.Router, s *store.Store, fo *routing.FailoverManager) *Proxy {
	p := &Proxy{
		router:   router,
		failover: fo,
		client:   &http.Client{Timeout: 300 * time.Second},
		store:    s,
		logCh:    make(chan *store.RequestLog, 1024),
	}

	if s != nil {
		go p.drainLogs()
	}

	return p
}

func (p *Proxy) drainLogs() {
	for l := range p.logCh {
		if err := p.store.InsertLog(l); err != nil {
			log.Printf("[mswitch] log write error: %v", err)
		}
	}
}

func (p *Proxy) HandleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read request body failed", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var chatReq ChatRequest
	if err := json.Unmarshal(body, &chatReq); err != nil {
		http.Error(w, "parse request body failed", http.StatusBadRequest)
		return
	}

	site := p.router.FindSiteForModel(chatReq.Model)
	if site == nil {
		http.Error(w, fmt.Sprintf("no site configured for model: %s", chatReq.Model), http.StatusBadRequest)
		return
	}

	if p.failover != nil && !p.failover.IsHealthy(site.ID, chatReq.Model) {
		cfg := p.router.GetConfig()
		if fallback := p.failover.FindFallback(chatReq.Model, cfg); fallback != nil {
			site = fallback
		}
	}

	adp := adapter.GetAdapter(site.Protocol)

	unifiedReq := &adapter.UnifiedRequest{
		Model:  chatReq.Model,
		Stream: chatReq.Stream,
		Body:   body,
	}

	upstreamReq, err := adp.ConvertRequest(site.BaseURL, site.APIKey, unifiedReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("convert request failed: %v", err), http.StatusInternalServerError)
		return
	}

	reqID := fmt.Sprintf("req-%d", p.reqCount.Add(1))
	start := time.Now()

	log.Printf("[mswitch] %s -> %s (model: %s, stream: %v)", r.RemoteAddr, site.ID, chatReq.Model, chatReq.Stream)

	resp, err := p.client.Do(upstreamReq)
	latency := time.Since(start).Milliseconds()

	reqLog := &store.RequestLog{
		RequestID: reqID,
		Timestamp: start,
		SiteID:    site.ID,
		Model:     chatReq.Model,
		Protocol:  site.Protocol,
		IsStream:  chatReq.Stream,
		LatencyMs: int(latency),
		ClientIP:  r.RemoteAddr,
	}

	if err != nil {
		reqLog.StatusCode = http.StatusBadGateway
		reqLog.Error = err.Error()
		p.asyncLog(reqLog)
		if p.failover != nil {
			p.failover.RecordFailure(site.ID, chatReq.Model)
		}
		http.Error(w, fmt.Sprintf("upstream request failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	reqLog.StatusCode = resp.StatusCode

	if resp.StatusCode >= 500 {
		if p.failover != nil {
			p.failover.RecordFailure(site.ID, chatReq.Model)
		}
	} else if resp.StatusCode < 400 && p.failover != nil {
		p.failover.RecordSuccess(site.ID, chatReq.Model)
	}

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)

	if chatReq.Stream {
		p.streamResponse(w, resp)
	} else {
		respBody, _ := io.ReadAll(resp.Body)
		w.Write(respBody)
		p.parseUsage(respBody, reqLog)
	}

	p.asyncLog(reqLog)
}

func (p *Proxy) parseUsage(body []byte, reqLog *store.RequestLog) {
	var resp struct {
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if json.Unmarshal(body, &resp) == nil {
		reqLog.InputTokens = resp.Usage.PromptTokens
		reqLog.OutputTokens = resp.Usage.CompletionTokens
	}
}

func (p *Proxy) asyncLog(l *store.RequestLog) {
	if p.store != nil {
		select {
		case p.logCh <- l:
		default:
		}
	}
}

func (p *Proxy) streamResponse(w http.ResponseWriter, resp *http.Response) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		io.Copy(w, resp.Body)
		return
	}

	reader := resp.Body
	buf := make([]byte, 4096)

	for {
		n, err := reader.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
			flusher.Flush()
		}
		if err != nil {
			if err != io.EOF {
				log.Printf("[mswitch] stream read error: %v", err)
			}
			break
		}
	}
}

func (p *Proxy) HandleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg := p.router.GetConfig()

	type Model struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	}

	var models []Model
	now := time.Now().Unix()

	for _, site := range cfg.Sites {
		for _, m := range site.Models {
			models = append(models, Model{
				ID:      m,
				Object:  "model",
				Created: now,
				OwnedBy: site.ID,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"object": "list",
		"data":   models,
	})
}

func (p *Proxy) HandleAny(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if strings.HasSuffix(path, "/v1/chat/completions") || path == "/chat/completions" {
		p.HandleChatCompletions(w, r)
		return
	}

	if strings.HasSuffix(path, "/v1/models") || path == "/models" {
		p.HandleModels(w, r)
		return
	}

	http.Error(w, fmt.Sprintf("path not supported: %s", path), http.StatusNotFound)
}
