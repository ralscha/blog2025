package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Options struct {
	Model         string
	MaxTurns      int
	EvalTimeoutS  int
	MemoryLimitMB int
	Verbose       bool
	DebugHTTP     bool
}

type Config struct {
	AnthropicAPIKey  string
	Model            string
	MaxTurns         int
	EvalTimeout      time.Duration
	MemoryLimitBytes uintptr
	Verbose          bool
	DebugHTTP        bool
}

func Load(opts Options) (Config, error) {
	_ = godotenv.Load()

	apiKey := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
	if apiKey == "" {
		return Config{}, fmt.Errorf("ANTHROPIC_API_KEY is required")
	}
	if opts.MaxTurns <= 0 {
		opts.MaxTurns = 6
	}
	if opts.EvalTimeoutS <= 0 {
		opts.EvalTimeoutS = 10
	}
	if opts.MemoryLimitMB <= 0 {
		opts.MemoryLimitMB = 32
	}
	model := strings.TrimSpace(opts.Model)
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	return Config{
		AnthropicAPIKey:  apiKey,
		Model:            model,
		MaxTurns:         opts.MaxTurns,
		EvalTimeout:      time.Duration(opts.EvalTimeoutS) * time.Second,
		MemoryLimitBytes: uintptr(opts.MemoryLimitMB) * 1024 * 1024,
		Verbose:          opts.Verbose,
		DebugHTTP:        opts.DebugHTTP,
	}, nil
}
