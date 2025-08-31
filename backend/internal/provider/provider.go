package provider

import "context"

// Client is a minimal LLM provider interface.
type Client interface {
    // Generate sends a single user instruction and returns (answer, finishReason, error).
    Generate(ctx context.Context, instruction string, maxTokens int) (string, string, error)
}

type Keys struct {
    OpenAI    string
    Google    string
    Anthropic string
}

func NewClient(provider, model string, keys Keys) Client {
    switch provider {
    case "openai":
        return &OpenAI{Model: model, Key: keys.OpenAI}
    case "gemini", "google":
        return &Gemini{Model: model, Key: keys.Google}
    case "anthropic", "claude":
        return &Anthropic{Model: model, Key: keys.Anthropic}
    default:
        return &Null{}
    }
}
