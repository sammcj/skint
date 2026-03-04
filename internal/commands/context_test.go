package commands

import (
	"testing"

	"github.com/sammcj/skint/internal/config"
)

func TestClaudeExtraArgsMergedWithClaudeArgs(t *testing.T) {
	tests := []struct {
		name       string
		claudeArgs []string
		extraArgs  []string
		want       []string
	}{
		{
			name:       "extra args appended to config args",
			claudeArgs: []string{"--verbose"},
			extraArgs:  []string{"--resume", "abc-123"},
			want:       []string{"--verbose", "--resume", "abc-123"},
		},
		{
			name:       "no config args with extra args",
			claudeArgs: nil,
			extraArgs:  []string{"--continue"},
			want:       []string{"--continue"},
		},
		{
			name:       "config args with no extra args",
			claudeArgs: []string{"--verbose"},
			extraArgs:  nil,
			want:       []string{"--verbose"},
		},
		{
			name:       "both empty",
			claudeArgs: nil,
			extraArgs:  nil,
			want:       nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{
				ClaudeArgs: tc.claudeArgs,
			}

			// Replicate the merging logic from LaunchClaude.
			args := append([]string{}, cfg.ClaudeArgs...)
			args = append(args, tc.extraArgs...)

			if len(tc.want) == 0 && len(args) == 0 {
				return
			}

			if len(args) != len(tc.want) {
				t.Fatalf("merged args length = %d, want %d\ngot:  %v\nwant: %v",
					len(args), len(tc.want), args, tc.want)
			}

			for i, want := range tc.want {
				if args[i] != want {
					t.Errorf("args[%d] = %q, want %q", i, args[i], want)
				}
			}
		})
	}
}
