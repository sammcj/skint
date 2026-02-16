package launcher

import "strings"

// ConflictingEnvVars is the list of environment variable names that are
// removed before setting provider-specific values. Both the Launcher and
// the exec command use this list to avoid stale values leaking through.
var ConflictingEnvVars = []string{
	"ANTHROPIC_BASE_URL",
	"ANTHROPIC_AUTH_TOKEN",
	"ANTHROPIC_API_KEY",
	"ANTHROPIC_MODEL",
	"ANTHROPIC_DEFAULT_HAIKU_MODEL",
	"ANTHROPIC_DEFAULT_SONNET_MODEL",
	"ANTHROPIC_DEFAULT_OPUS_MODEL",
	"ANTHROPIC_SMALL_FAST_MODEL",
	"OPENAI_BASE_URL",
	"OPENAI_API_KEY",
	"OPENAI_MODEL",
}

// FilterEnvVars removes the named variables from an environment slice.
// Entries without '=' are preserved as-is.
func FilterEnvVars(env []string, vars ...string) []string {
	varNames := make(map[string]bool, len(vars))
	for _, v := range vars {
		varNames[v] = true
	}

	var result []string
	for _, e := range env {
		name, _, ok := strings.Cut(e, "=")
		if !ok {
			// Entry without '=' -- preserve it
			result = append(result, e)
			continue
		}
		if !varNames[name] {
			result = append(result, e)
		}
	}

	return result
}
