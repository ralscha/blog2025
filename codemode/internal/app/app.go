package app

import (
	"context"
	"fmt"

	anthropicloop "github.com/preblog/codemode/internal/anthropic"
	"github.com/preblog/codemode/internal/catalog"
	"github.com/preblog/codemode/internal/codemode"
	"github.com/preblog/codemode/internal/config"
	"github.com/preblog/codemode/internal/mcpdemo"
	"github.com/preblog/codemode/internal/sandbox"
)

type App struct {
	runtime *mcpdemo.Runtime
	loop    *anthropicloop.Runner
}

func New(cfg config.Config, captureHTTPInTrace bool) (*App, error) {
	ctx := context.Background()
	runtime, err := mcpdemo.NewDemo(ctx)
	if err != nil {
		return nil, err
	}
	items, err := catalog.Load(ctx, runtime.ServerName(), runtime)
	if err != nil {
		if closeErr := runtime.Close(); closeErr != nil {
			return nil, fmt.Errorf("load catalog: %w (close runtime: %v)", err, closeErr)
		}
		return nil, fmt.Errorf("load catalog: %w", err)
	}
	sb := sandbox.New(items, runtime, cfg.EvalTimeout, cfg.MemoryLimitBytes)
	toolset := &codemode.Toolset{Catalog: items, Sandbox: sb}
	loop := anthropicloop.New(cfg.AnthropicAPIKey, cfg.Model, cfg.MaxTurns, cfg.Verbose, cfg.DebugHTTP, captureHTTPInTrace, toolset)
	return &App{runtime: runtime, loop: loop}, nil
}

func (a *App) Run(ctx context.Context, prompt string) (anthropicloop.Result, error) {
	return a.loop.Run(ctx, prompt)
}

func (a *App) Close() error {
	if a.runtime == nil {
		return nil
	}
	return a.runtime.Close()
}
