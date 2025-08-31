package orch

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/you/swarmone/internal/provider"
)

// Meta returned to HTTP layer (judge-only).
type Meta struct {
	WinnerIndex     int       `json:"winner_index"`
	Runners         int       `json:"runners"`
	Scores          []float64 `json:"scores"`
	IncludedIndices []int     `json:"included_indices"`
	ConsensusID     string    `json:"consensus_id"`
	RunnerErrors    []string  `json:"runner_errors"`
}

type cand struct {
	Orig int
	Text string
}

// Execute: fan-out to runners → judge-only → map scores back → return.
func Execute(ctx context.Context, cfg *Config, keys Keys, instruction string) (string, Meta, error) {
	if cfg == nil {
		return "", Meta{}, errors.New("nil config")
	}
	if len(cfg.Runners) == 0 {
		return "", Meta{}, errors.New("no runners configured")
	}

	// Build clients
	clients := make([]provider.Client, len(cfg.Runners))
	for i, r := range cfg.Runners {
		cl, err := buildClient(r, keys)
		if err != nil {
			return "", Meta{}, fmt.Errorf("build client for runner %d failed: %w", i, err)
		}
		clients[i] = cl
	}

	type res struct {
		idx  int
		text string
		err  error
	}
	answers := make([]string, len(cfg.Runners))
	runnerErrs := make([]string, len(cfg.Runners))
	ch := make(chan res, len(cfg.Runners))

	var wg sync.WaitGroup
	for i, spec := range cfg.Runners {
		wg.Add(1)
		go func(idx int, rs RunnerSpec, cl provider.Client) {
			defer wg.Done()

			rctx := ctx
			if cfg.Server.RunnerTimeout > 0 {
				var cancel context.CancelFunc
				rctx, cancel = context.WithTimeout(ctx, cfg.Server.RunnerTimeout)
				defer cancel()
			}
			t, _, err := cl.Generate(rctx, instruction, rs.MaxTokens)
			if err != nil {
				runnerErrs[idx] = err.Error()
			}
			ch <- res{idx: idx, text: strings.TrimSpace(t), err: err}
		}(i, spec, clients[i])
	}

	go func() { wg.Wait(); close(ch) }()
	for r := range ch {
		if r.err == nil {
			answers[r.idx] = r.text
		}
	}

	// Build candidates (non-empty only)
	var cands []cand
	var included []int
	for i, t := range answers {
		if strings.TrimSpace(t) != "" {
			cands = append(cands, cand{Orig: i, Text: t})
			included = append(included, i)
		}
	}

	consID := randomID()
	if len(cands) == 0 {
		meta := Meta{
			WinnerIndex:     -1,
			Runners:         len(cfg.Runners),
			Scores:          make([]float64, len(cfg.Runners)),
			IncludedIndices: included,
			ConsensusID:     consID,
			RunnerErrors:    runnerErrs,
		}
		return "", meta, fmt.Errorf("all runners failed")
	}

	// Judge-only
	winnerOrig, candScores, err := judgePick(ctx, cfg, keys, instruction, answers, cands)
	if err != nil {
		meta := Meta{
			WinnerIndex:     -1,
			Runners:         len(cfg.Runners),
			Scores:          make([]float64, len(cfg.Runners)),
			IncludedIndices: included,
			ConsensusID:     consID,
			RunnerErrors:    runnerErrs,
		}
		return "", meta, fmt.Errorf("judge error: %w", err)
	}

	// Map candidate scores -> absolute runner indices
	absScores := make([]float64, len(cfg.Runners))
	for i := range absScores {
		absScores[i] = 0
	}
	if len(candScores) == len(cands) {
		for i, c := range cands {
			absScores[c.Orig] = clampRound4(candScores[i])
		}
	}

	meta := Meta{
		WinnerIndex:     winnerOrig,
		Runners:         len(cfg.Runners),
		Scores:          absScores,
		IncludedIndices: included,
		ConsensusID:     consID,
		RunnerErrors:    runnerErrs,
	}
	return answers[winnerOrig], meta, nil
}

// judgePick asks the judge model to score each candidate ([0,1], 4 decimals) and pick a winner.
// NOTE: it now accepts []cand to match call-site type exactly.
func judgePick(
	ctx context.Context,
	cfg *Config,
	keys Keys,
	instruction string,
	answers []string,
	cands []cand,
) (int, []float64, error) {
	// Build judge client
	if cfg.Consensus.Judge.Provider == "" || cfg.Consensus.Judge.Model == "" {
		return 0, nil, errors.New("judge provider/model not configured")
	}
	jSpec := RunnerSpec{
		Name:      "judge",
		Provider:  cfg.Consensus.Judge.Provider,
		Model:     cfg.Consensus.Judge.Model,
		MaxTokens: cfg.Consensus.Judge.MaxTokens,
	}
	jc, err := buildClient(jSpec, keys)
	if err != nil {
		return 0, nil, fmt.Errorf("build judge client: %w", err)
	}

	// Prepare JSON payload for judge
	type jcand struct {
		Index int    `json:"index"`
		Text  string `json:"text"`
	}
	jcands := make([]jcand, 0, len(cands))
	for i, c := range cands {
		jcands = append(jcands, jcand{Index: i, Text: c.Text})
	}
	req := map[string]any{
		"task":        "score each candidate and choose a single best one",
		"instruction": instruction,
		"candidates":  jcands,
		"schema": map[string]any{
			"scores": "array of numbers in [0,1] with 4 decimals, length == number of candidates",
			"winner": "integer candidate index",
		},
		"criteria": []string{
			"Task match / completeness",
			"Clarity / organization",
			"Factuality / safety",
			"Tone / style follows Language",
		},
		"format": "Return ONLY JSON: {\"scores\":[...], \"winner\": <int>}",
	}
	b, _ := json.Marshal(req)
	prompt := "You are a strict impartial judge.\n" +
		"Score every candidate between 0 and 1 (4 decimals). Higher is better.\n" +
		"Choose ONE winner. Return ONLY JSON as specified.\n\n" + string(b)

	// Dedicated timeout for judge
	var jctx context.Context = ctx
	var cancel context.CancelFunc
	if cfg.Server.RequestTimeout > 0 {
		if dl, ok := ctx.Deadline(); ok {
			rem := time.Until(dl)
			d := rem * 8 / 10
			if d < 10*time.Second {
				d = 10 * time.Second
			}
			if d > 30*time.Second {
				d = 30 * time.Second
			}
			jctx, cancel = context.WithTimeout(ctx, d)
		} else {
			jctx, cancel = context.WithTimeout(ctx, 20*time.Second)
		}
	} else {
		jctx, cancel = context.WithTimeout(ctx, 20*time.Second)
	}
	defer cancel()

	maxTok := cfg.Consensus.Judge.MaxTokens
	if maxTok <= 0 {
		maxTok = 256
	}
	out, _, err := jc.Generate(jctx, prompt, maxTok)
	if err != nil {
		return 0, nil, err
	}
	txt := strings.TrimSpace(out)
	if txt == "" {
		return 0, nil, errors.New("judge returned empty content")
	}
	txt = stripCodeFence(txt)

	// Parse/repair JSON
	var jr struct {
		Scores []float64 `json:"scores"`
		Winner *int      `json:"winner"`
	}
	if err := json.Unmarshal([]byte(txt), &jr); err != nil {
		nums := extractNumbers(txt)
		if len(nums) >= len(cands) {
			jr.Scores = nums[:len(cands)]
		}
		if jr.Winner == nil {
			if w := findWinnerIndex(txt); w >= 0 && w < len(cands) {
				jr.Winner = &w
			}
		}
		if len(jr.Scores) != len(cands) || jr.Winner == nil {
			return 0, nil, fmt.Errorf("judge unparsable: %s", truncate(txt, 500))
		}
	}
	if len(jr.Scores) != len(cands) {
		return 0, nil, fmt.Errorf("judge scores length mismatch: got %d, want %d", len(jr.Scores), len(cands))
	}

	w := 0
	if jr.Winner != nil {
		w = *jr.Winner
	}
	if w < 0 || w >= len(cands) {
		w = argmax(jr.Scores)
	}
	for i := range jr.Scores {
		jr.Scores[i] = clampRound4(jr.Scores[i])
	}
	return cands[w].Orig, jr.Scores, nil
}

// -------- helpers --------

func clampRound4(x float64) float64 {
	if x != x || x > 1 {
		x = 1
	}
	if x < 0 {
		x = 0
	}
	return float64(int64(x*10000+0.5)) / 10000
}

func randomID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func stripCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		if i := strings.Index(s[3:], "\n"); i >= 0 {
			s = s[3+i+1:]
		}
	}
	if strings.HasSuffix(s, "```") {
		s = strings.TrimSuffix(s, "```")
	}
	return strings.TrimSpace(s)
}

var numRe = regexp.MustCompile(`[-+]?\d+(\.\d+)?`)

func extractNumbers(s string) []float64 {
	m := numRe.FindAllString(s, -1)
	out := make([]float64, 0, len(m))
	for _, mm := range m {
		var v float64
		_, err := fmt.Sscanf(mm, "%f", &v)
		if err == nil {
			out = append(out, v)
		}
	}
	return out
}

func findWinnerIndex(s string) int {
	re := regexp.MustCompile(`"winner"\s*:\s*(\d+)`)
	ms := re.FindStringSubmatch(s)
	if len(ms) == 2 {
		var w int
		_, err := fmt.Sscanf(ms[1], "%d", &w)
		if err == nil {
			return w
		}
	}
	return -1
}

func argmax(a []float64) int {
	if len(a) == 0 {
		return 0
	}
	best := 0
	bv := a[0]
	for i := 1; i < len(a); i++ {
		if a[i] > bv {
			bv, best = a[i], i
		}
	}
	return best
}

func truncate(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	if n > 3 {
		return s[:n-3] + "..."
	}
	return s[:n]
}
