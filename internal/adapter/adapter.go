package adapter

import (
	"encoding/json"
	"net/http"
)

type UnifiedRequest struct {
	Model    string          `json:"model"`
	Messages json.RawMessage `json:"messages"`
	Stream   bool            `json:"stream,omitempty"`
	Body     json.RawMessage `json:"-"`
}

type UnifiedResponse struct {
	ID      string          `json:"id"`
	Object  string          `json:"object"`
	Model   string          `json:"model"`
	Choices json.RawMessage `json:"choices"`
	Usage   json.RawMessage `json:"usage,omitempty"`
	Body    json.RawMessage `json:"-"`
}

type ConnectivityResult struct {
	Ok             bool   `json:"ok"`
	LatencyMs      int64  `json:"latency_ms"`
	Error          string `json:"error,omitempty"`
	ModelsAvailable int   `json:"models_available,omitempty"`
	SiteID         string `json:"site_id,omitempty"`
}

type ModelInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	Created int64  `json:"created,omitempty"`
}

type ProtocolAdapter interface {
	ConvertRequest(baseURL string, apiKey string, req *UnifiedRequest) (*http.Request, error)
	ConvertResponse(resp *http.Response) (*UnifiedResponse, error)
	ConvertStreamChunk(chunk []byte) ([]byte, error)
	TestConnectivity(baseURL string, apiKey string) (*ConnectivityResult, error)
	ListModels(baseURL string, apiKey string) ([]ModelInfo, error)
	Name() string
}

func GetAdapter(protocol string) ProtocolAdapter {
	switch protocol {
	case "openai":
		return &OpenAIAdapter{}
	case "anthropic":
		return &AnthropicAdapter{}
	case "gemini":
		return &GeminiAdapter{}
	default:
		return &OpenAIAdapter{}
	}
}
