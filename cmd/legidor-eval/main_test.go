package main

import (
	"os"
	"strings"
	"testing"
)

func TestExpensiveModelsRequireOptInBeforeAnyExecution(t *testing.T) {
	previous := os.Args
	t.Cleanup(func() { os.Args = previous })
	for _, args := range [][]string{
		{"run", "-claude-model", "claude-opus-5-5"},
		{"run", "-claude-models", "claude-sonnet-5-5,claude-opus-5-5"},
		{"run", "-agents", "codex", "-codex-model", "gpt-6-astra"},
		{"run", "-agents", "codex", "-codex-models", "gpt-6.1-sol,gpt-6-astra"},
		{"grade", "-model", "claude-opus-5-5"},
		{"grade", "-agent", "codex", "-model", "gpt-6-astra"},
		{"campaign", "-judge-model", "claude-opus-5-5"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			os.Args = append([]string{"legidor-eval"}, args...)
			err := run()
			if err == nil || !strings.Contains(err.Error(), "-allow-expensive-models") {
				t.Fatalf("expected model policy rejection before input validation, got %v", err)
			}
		})
	}
}

func TestExpensiveModelOptInReachesInputValidation(t *testing.T) {
	previous := os.Args
	t.Cleanup(func() { os.Args = previous })
	os.Args = []string{"legidor-eval", "run", "-claude-model", "claude-opus-5-5", "-allow-expensive-models"}
	if err := run(); err == nil || !strings.Contains(err.Error(), "output") {
		t.Fatalf("explicit opt-in did not reach input validation: %v", err)
	}
}
