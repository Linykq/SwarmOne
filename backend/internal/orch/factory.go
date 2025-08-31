package orch

import (
	"fmt"
	"strings"

	"github.com/you/swarmone/internal/provider"
)

// buildClient creates a provider.Client from RunnerSpec + Keys.
// Requires provider package to expose NewOpenAI / NewGemini / NewAnthropic.
func buildClient(r RunnerSpec, keys Keys) (provider.Client, error) {
	switch strings.ToLower(strings.TrimSpace(r.Provider)) {
	case "openai":
		return provider.NewOpenAI(r.Model, keys.OpenAI), nil
	case "gemini", "google", "googleai":
		return provider.NewGemini(r.Model, keys.Google), nil
	case "anthropic", "claude":
		return provider.NewAnthropic(r.Model, keys.Anthropic), nil
	default:
		return nil, fmt.Errorf("unknown provider %q", r.Provider)
	}
}
