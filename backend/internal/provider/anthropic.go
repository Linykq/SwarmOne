package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Minimal Anthropic Messages API client (stdlib).
// POST https://api.anthropic.com/v1/messages
// Headers: x-api-key, anthropic-version: 2023-06-01
type Anthropic struct {
	Model string
	Key   string
	HTTP  *http.Client
}

func NewAnthropic(model, key string) Client {
	return &Anthropic{Model: model, Key: key}
}

func (a *Anthropic) ensureHTTP() {
	if a.HTTP != nil {
		return
	}
	timeout := 18 * time.Second
	if t := os.Getenv("ANTHROPIC_HTTP_TIMEOUT"); t != "" {
		if d, err := time.ParseDuration(t); err == nil {
			timeout = d
		}
	}
	a.HTTP = &http.Client{Timeout: timeout}
}

func (a *Anthropic) Generate(ctx context.Context, prompt string, maxTokens int) (string, string, error) {
	if a.Key == "" {
		return "", "", errors.New("anthropic api key missing")
	}
	a.ensureHTTP()
	if maxTokens <= 0 {
		maxTokens = 256
	}

	payload := map[string]any{
		"model":       a.Model,
		"max_tokens":  maxTokens,
		"messages":    []map[string]any{{"role": "user", "content": prompt}},
		"temperature": 0,
	}
	b, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(b))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.Key)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.HTTP.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("anthropic http %d: %s", resp.StatusCode, string(raw))
	}

	// content is an array of blocks; we concatenate text blocks
	var jr struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &jr); err != nil {
		return "", "", fmt.Errorf("anthropic decode error: %v; body=%s", err, string(raw))
	}
	var sb strings.Builder
	for _, p := range jr.Content {
		if strings.ToLower(p.Type) == "text" && strings.TrimSpace(p.Text) != "" {
			if sb.Len() > 0 {
				sb.WriteByte('\n')
			}
			sb.WriteString(strings.TrimSpace(p.Text))
		}
	}
	txt := strings.TrimSpace(sb.String())
	if txt == "" {
		return "", "", errors.New("anthropic empty output")
	}
	return txt, "", nil
}
