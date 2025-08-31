package orch

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"
)

// Keys holds provider API keys.
type Keys struct {
	OpenAI    string
	Google    string
	Anthropic string
}

// RunnerSpec describes a worker model.
type RunnerSpec struct {
	Name      string `json:"name"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	MaxTokens int    `json:"max_tokens"`
}

// JudgeSpec defines the arbitrator model.
type JudgeSpec struct {
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	MaxTokens int    `json:"max_tokens"`
}

// Consensus keeps judge configuration (we only support judge-only now).
type Consensus struct {
	Judge JudgeSpec `json:"judge"`
}

// Server options.
type Server struct {
	Addr           string        // e.g. ":8080"
	RequestTimeout time.Duration // overall request budget
	RunnerTimeout  time.Duration // per-runner budget
}

// Config is the whole runtime config used by the orchestrator.
type Config struct {
	Server    Server       `json:"server"`
	Runners   []RunnerSpec `json:"runners"`
	Consensus Consensus    `json:"consensus"`
}

// Load builds Config and Keys from environment variables with safe defaults.
// This keeps dev bootstrap simple; you can switch to YAML later without changing callsites.
func Load() (*Config, Keys, error) {
	// Keys
	keys := Keys{
		OpenAI:    os.Getenv("OPENAI_API_KEY"),
		Google:    os.Getenv("GOOGLE_API_KEY"),
		Anthropic: os.Getenv("ANTHROPIC_API_KEY"),
	}

	// Server
	addr := os.Getenv("SERVER_ADDR")
	if strings.TrimSpace(addr) == "" {
		addr = ":8080"
	}
	reqTO := parseDurDefault(os.Getenv("REQUEST_TIMEOUT"), 25*time.Second)
	runTO := parseDurDefault(os.Getenv("RUNNER_TIMEOUT"), 12*time.Second)

	// Runners: from SWARMONE_RUNNERS (JSON array) or sensible defaults.
	var runners []RunnerSpec
	if raw := strings.TrimSpace(os.Getenv("SWARMONE_RUNNERS")); raw != "" {
		_ = json.Unmarshal([]byte(raw), &runners) // if malformed we fall back below
	}
	if len(runners) == 0 {
		// defaults try to mirror your previous setup
		runners = []RunnerSpec{
			{Name: "runner-openai", Provider: "openai", Model: "gpt-5-nano-2025-08-07", MaxTokens: 512},
			{Name: "runner-gemini", Provider: "gemini", Model: "gemini-2.5-flash", MaxTokens: 512},
			{Name: "runner-claude", Provider: "anthropic", Model: "claude-3-5-haiku-20241022", MaxTokens: 512},
		}
	}

	// Judge: env overrides or default to Anthropic (strong & stable).
	judgeProv := firstNonEmpty(os.Getenv("JUDGE_PROVIDER"), "anthropic")
	judgeModel := firstNonEmpty(os.Getenv("JUDGE_MODEL"), "claude-3-5-sonnet-20241022")
	judgeMax := parseIntDefault(os.Getenv("JUDGE_MAX_TOKENS"), 384)

	cfg := &Config{
		Server: Server{
			Addr:           addr,
			RequestTimeout: reqTO,
			RunnerTimeout:  runTO,
		},
		Runners: runners,
		Consensus: Consensus{
			Judge: JudgeSpec{
				Provider:  judgeProv,
				Model:     judgeModel,
				MaxTokens: judgeMax,
			},
		},
	}
	return cfg, keys, nil
}

func parseDurDefault(s string, d time.Duration) time.Duration {
	if strings.TrimSpace(s) == "" {
		return d
	}
	if v, err := time.ParseDuration(s); err == nil {
		return v
	}
	return d
}

func parseIntDefault(s string, d int) int {
	if strings.TrimSpace(s) == "" {
		return d
	}
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return d
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
