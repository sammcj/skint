package ui

import "testing"

func TestMaskKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{name: "empty string", key: "", want: ""},
		{name: "short key hides everything", key: "abc123", want: "****"},
		{name: "exactly 12 chars hides everything", key: "123456789012", want: "****"},
		{name: "13 chars shows first and last 4", key: "1234567890123", want: "1234****0123"},
		{name: "typical API key", key: "sk-ant-api03-abcdefgh", want: "sk-a****efgh"},
		{name: "single char", key: "x", want: "****"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := MaskKey(tc.key)
			if got != tc.want {
				t.Errorf("MaskKey(%q) = %q, want %q", tc.key, got, tc.want)
			}
		})
	}
}
