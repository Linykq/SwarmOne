package provider

import (
    "context"
    "errors"
)

// Null client returns error; used for unknown providers.
type Null struct{}

func (*Null) Generate(ctx context.Context, instruction string, maxTokens int) (string, string, error) {
    return "", "", errors.New("unknown provider")
}
