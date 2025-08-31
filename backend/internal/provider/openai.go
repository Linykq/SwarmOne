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

// OpenAI client for Responses API (stdlib HTTP).
// Text extraction: output_text -> output[].content[].text -> recursive "text" fields.
// When no text found, we surface status/finish_reason to help debugging.
type OpenAI struct {
	Model string
	Key   string
	HTTP  *http.Client
}

func NewOpenAI(model, key string) Client {
	return &OpenAI{Model: model, Key: key, HTTP: nil}
}

func (c *OpenAI) ensureHTTP() {
	if c.HTTP != nil {
		return
	}
	timeout := 18 * time.Second
	if t := os.Getenv("OPENAI_HTTP_TIMEOUT"); t != "" {
		if d, err := time.ParseDuration(t); err == nil {
			timeout = d
		}
	}
	cl := &http.Client{}
	cl.Timeout = timeout
	c.HTTP = cl
}

// (text, meta, error). meta 未使用，返回 ""。
func (c *OpenAI) Generate(ctx context.Context, prompt string, maxTokens int) (string, string, error) {
	if c.Key == "" {
		return "", "", errors.New("openai api key missing")
	}
	c.ensureHTTP()

	payload := map[string]any{
		"model": c.Model,
		"input": prompt,
	}
	if maxTokens > 0 {
		payload["max_output_tokens"] = maxTokens
	}

	bodyBytes, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/responses", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.Key)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("openai http %d: %s", resp.StatusCode, string(respBody))
	}

	var raw map[string]any
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return "", "", fmt.Errorf("openai decode error: %v; body=%s", err, string(respBody))
	}

	// 1) Prefer "output_text"
	if s, ok := raw["output_text"].(string); ok {
		if ts := strings.TrimSpace(s); ts != "" {
			return ts, "", nil
		}
	}

	// 2) Common: output[].content[].text
	if out, ok := raw["output"].([]any); ok {
		if acc := collectAllText(out); acc != "" {
			return acc, "", nil
		}
	}

	// 3) Last resort: recursive scan for any "text"
	if acc := collectAllText(raw); acc != "" {
		return acc, "", nil
	}

	// No usable text → surface diagnostics (status/finish_reasons)
	status := asString(raw["status"])
	var reasons []string
	if out, ok := raw["output"].([]any); ok {
		for _, it := range out {
			if m, ok := it.(map[string]any); ok {
				if fr := asString(m["finish_reason"]); fr != "" {
					reasons = append(reasons, fr)
				}
			}
		}
	}
	return "", "", fmt.Errorf("openai empty output (status=%q, finish_reasons=%v)", status, reasons)
}

// ---------- helpers ----------

func collectAllText(v any) string {
	var buf []string
	var walk func(any)
	walk = func(x any) {
		switch t := x.(type) {
		case map[string]any:
			if s, ok := t["text"].(string); ok {
				if ts := strings.TrimSpace(s); ts != "" {
					buf = append(buf, ts)
				}
			}
			for _, vv := range t {
				walk(vv)
			}
		case []any:
			for _, it := range t {
				walk(it)
			}
		}
	}
	walk(v)
	if len(buf) == 0 {
		return ""
	}
	var b strings.Builder
	for i, s := range buf {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(s)
	}
	return b.String()
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	if b, err := json.Marshal(v); err == nil {
		return string(b)
	}
	return ""
}
