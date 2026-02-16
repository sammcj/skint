package launcher

import (
	"slices"
	"testing"
)

// envEqual reports whether two environment slices contain the same entries
// in the same order. Both nil and empty slices are treated as equivalent.
func envEqual(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("length mismatch: got %d, want %d\ngot:  %v\nwant: %v", len(got), len(want), got, want)
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("mismatch at index %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestFilterEnvVars(t *testing.T) {
	tests := []struct {
		name string
		env  []string
		vars []string
		want []string
	}{
		{
			name: "matching vars are removed",
			env:  []string{"FOO=bar", "BAZ=qux", "KEEP=yes"},
			vars: []string{"FOO", "BAZ"},
			want: []string{"KEEP=yes"},
		},
		{
			name: "non-matching vars are preserved",
			env:  []string{"ALPHA=1", "BRAVO=2", "CHARLIE=3"},
			vars: []string{"DELTA", "ECHO"},
			want: []string{"ALPHA=1", "BRAVO=2", "CHARLIE=3"},
		},
		{
			name: "entries without equals sign are preserved",
			env:  []string{"NOEQUALS", "HAS=value", "ANOTHER"},
			vars: []string{"HAS"},
			want: []string{"NOEQUALS", "ANOTHER"},
		},
		{
			name: "empty env returns nil",
			env:  []string{},
			vars: []string{"ANYTHING"},
			want: nil,
		},
		{
			name: "nil env returns nil",
			env:  nil,
			vars: []string{"ANYTHING"},
			want: nil,
		},
		{
			name: "multiple vars removed at once",
			env: []string{
				"ANTHROPIC_BASE_URL=https://example.com",
				"ANTHROPIC_API_KEY=sk-secret",
				"HOME=/home/user",
				"OPENAI_API_KEY=oai-secret",
				"PATH=/usr/bin",
			},
			vars: []string{"ANTHROPIC_BASE_URL", "ANTHROPIC_API_KEY", "OPENAI_API_KEY"},
			want: []string{"HOME=/home/user", "PATH=/usr/bin"},
		},
		{
			name: "duplicate entries with same key are all removed",
			env:  []string{"DUP=first", "KEEP=yes", "DUP=second", "DUP=third"},
			vars: []string{"DUP"},
			want: []string{"KEEP=yes"},
		},
		{
			name: "no vars to remove preserves everything",
			env:  []string{"A=1", "B=2"},
			vars: []string{},
			want: []string{"A=1", "B=2"},
		},
		{
			name: "value containing equals sign is handled correctly",
			env:  []string{"CONN=host=localhost;port=5432", "DROP=me"},
			vars: []string{"DROP"},
			want: []string{"CONN=host=localhost;port=5432"},
		},
		{
			name: "key name is a substring of another key",
			env:  []string{"FOO=1", "FOOBAR=2", "FOO_BAR=3"},
			vars: []string{"FOO"},
			want: []string{"FOOBAR=2", "FOO_BAR=3"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Take a copy so we can verify the original slice is not mutated.
			var orig []string
			if tc.env != nil {
				orig = slices.Clone(tc.env)
			}

			got := FilterEnvVars(tc.env, tc.vars...)

			envEqual(t, got, tc.want)

			// Verify the input slice was not mutated.
			if orig != nil {
				envEqual(t, tc.env, orig)
			}
		})
	}
}

func TestConflictingEnvVars(t *testing.T) {
	// Verify the shared list contains the expected variables.
	expected := map[string]bool{
		"ANTHROPIC_BASE_URL":             true,
		"ANTHROPIC_AUTH_TOKEN":           true,
		"ANTHROPIC_API_KEY":              true,
		"ANTHROPIC_MODEL":                true,
		"ANTHROPIC_DEFAULT_HAIKU_MODEL":  true,
		"ANTHROPIC_DEFAULT_SONNET_MODEL": true,
		"ANTHROPIC_DEFAULT_OPUS_MODEL":   true,
		"ANTHROPIC_SMALL_FAST_MODEL":     true,
		"OPENAI_BASE_URL":                true,
		"OPENAI_API_KEY":                 true,
		"OPENAI_MODEL":                   true,
	}

	if len(ConflictingEnvVars) != len(expected) {
		t.Fatalf("ConflictingEnvVars has %d entries, expected %d", len(ConflictingEnvVars), len(expected))
	}

	for _, v := range ConflictingEnvVars {
		if !expected[v] {
			t.Errorf("unexpected entry in ConflictingEnvVars: %s", v)
		}
	}
}
