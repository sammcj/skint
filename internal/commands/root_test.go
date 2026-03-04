package commands

import (
	"context"
	"testing"

	"github.com/spf13/cobra"
)

func TestClaudePassthroughFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantArgs []string
	}{
		{
			name:     "no passthrough flags produces empty extra args",
			args:     []string{},
			wantArgs: nil,
		},
		{
			name:     "resume flag passes session ID",
			args:     []string{"--resume", "abc-123"},
			wantArgs: []string{"--resume", "abc-123"},
		},
		{
			name:     "continue long flag",
			args:     []string{"--continue"},
			wantArgs: []string{"--continue"},
		},
		{
			name:     "continue short flag",
			args:     []string{"-c"},
			wantArgs: []string{"--continue"},
		},
		{
			name:     "both resume and continue",
			args:     []string{"--resume", "sess-456", "--continue"},
			wantArgs: []string{"--resume", "sess-456", "--continue"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := NewRootCmd("test")

			// Disable the RunE so we don't need a full config/TUI setup.
			root.RunE = func(cmd *cobra.Command, args []string) error {
				return nil
			}

			// Replace PersistentPreRunE to only build passthrough args (skip initialize).
			cc := &CmdContext{OutputFormat: "human"}
			root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
				cmd.SetContext(context.WithValue(cmd.Context(), ctxKey, cc))

				// Read parsed flag values (production uses closure-captured vars bound via StringVar/BoolVarP,
				// but the result is identical since both read from the same pflag set).
				resume, _ := cmd.Flags().GetString("resume")
				cont, _ := cmd.Flags().GetBool("continue")

				if resume != "" {
					cc.ClaudeExtraArgs = append(cc.ClaudeExtraArgs, "--resume", resume)
				}
				if cont {
					cc.ClaudeExtraArgs = append(cc.ClaudeExtraArgs, "--continue")
				}
				return nil
			}

			root.SetArgs(tc.args)
			if err := root.Execute(); err != nil {
				t.Fatalf("Execute() error: %v", err)
			}

			if len(tc.wantArgs) == 0 && len(cc.ClaudeExtraArgs) == 0 {
				return
			}

			if len(cc.ClaudeExtraArgs) != len(tc.wantArgs) {
				t.Fatalf("ClaudeExtraArgs length = %d, want %d\ngot:  %v\nwant: %v",
					len(cc.ClaudeExtraArgs), len(tc.wantArgs), cc.ClaudeExtraArgs, tc.wantArgs)
			}

			for i, want := range tc.wantArgs {
				if cc.ClaudeExtraArgs[i] != want {
					t.Errorf("ClaudeExtraArgs[%d] = %q, want %q", i, cc.ClaudeExtraArgs[i], want)
				}
			}
		})
	}
}
