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

type BalanceInfo struct {
	SiteID    string  `json:"site_id"`
	Balance   float64 `json:"balance"`
	Currency  string  `json:"currency"`
	Raw       string  `json:"raw,omitempty"`
}

type ProtocolAdapter interface {
	ConvertRequest(baseURL string, apiKey string, req *UnifiedRequest) (*http.Request, error)
	ConvertResponse(resp *http.Response) (*UnifiedResponse, error)
	ConvertStreamChunk(chunk []byte) ([]byte, error)
	ParseBalance(body []byte) (*BalanceInfo, error)
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
