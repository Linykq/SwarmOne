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

// Gemini client via REST (no external deps).
// Endpoint: https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent?key=API_KEY
// When blocked by safety, returns an error with blockReason instead of silent "".
type Gemini struct {
	Model string
	Key   string
	HTTP  *http.Client
}

func NewGemini(model, key string) Client {
	return &Gemini{Model: model, Key: key, HTTP: nil}
}

func (g *Gemini) ensureHTTP() {
	if g.HTTP != nil {
		return
	}
	timeout := 18 * time.Second
	if t := os.Getenv("GEMINI_HTTP_TIMEOUT"); t != "" {
		if d, err := time.ParseDuration(t); err == nil {
			timeout = d
		}
	}
	cl := &http.Client{}
	cl.Timeout = timeout
	g.HTTP = cl
}

func (g *Gemini) Generate(ctx context.Context, prompt string, maxTokens int) (string, string, error) {
	if g.Key == "" {
		return "", "", errors.New("gemini api key missing")
	}
	g.ensureHTTP()

	type part struct {
		Text string `json:"text,omitempty"`
	}
	type content struct {
		Role  string `json:"role,omitempty"`
		Parts []part `json:"parts,omitempty"`
	}
	body := map[string]any{
		"contents": []content{{Role: "user", Parts: []part{{Text: prompt}}}},
	}
	if maxTokens > 0 {
		body["generationConfig"] = map[string]any{"maxOutputTokens": maxTokens}
	}

	b, _ := json.Marshal(body)
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.Model, g.Key)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.HTTP.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("gemini http %d: %s", resp.StatusCode, string(raw))
	}

	var jr map[string]any
	if err := json.Unmarshal(raw, &jr); err != nil {
		return "", "", fmt.Errorf("gemini decode error: %v; body=%s", err, string(raw))
	}

	// If blocked by safety, the API often returns promptFeedback.blockReason
	if pf, ok := jr["promptFeedback"].(map[string]any); ok {
		if br, ok := pf["blockReason"].(string); ok && br != "" {
			return "", "", fmt.Errorf("gemini safety block: %s", br)
		}
	}

	var out strings.Builder
	if cands, ok := jr["candidates"].([]any); ok {
		for _, c := range cands {
			cm, ok := c.(map[string]any)
			if !ok {
				continue
			}
			cont, _ := cm["content"].(map[string]any)
			if cont == nil {
				continue
			}
			if parts, ok := cont["parts"].([]any); ok {
				for _, p := range parts {
					if pm, ok := p.(map[string]any); ok {
						if s, ok := pm["text"].(string); ok {
							if ts := strings.TrimSpace(s); ts != "" {
								if out.Len() > 0 {
									out.WriteByte('\n')
								}
								out.WriteString(ts)
							}
						}
					}
				}
			}
		}
	}

	txt := strings.TrimSpace(out.String())
	if txt == "" {
		// Nothing usable and no explicit blockReason → let caller see a generic error
		return "", "", errors.New("gemini empty output")
	}
	return txt, "", nil
}
