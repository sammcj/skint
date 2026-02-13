package commands

import (
	"fmt"
	"net/http"
	"time"

	"github.com/sammcj/skint/internal/config"
	"github.com/sammcj/skint/internal/ui"
	"github.com/spf13/cobra"
)

// NewTestCmd creates the test command
func NewTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test [provider]",
		Short: "Test provider connectivity",
		Long: `Test connectivity to LLM providers by making HTTP requests
to their API endpoints.`,
		RunE: runTest,
	}
}

func runTest(cmd *cobra.Command, args []string) error {
	cc := GetContext(cmd)
	var providersToTest []*config.Provider

	if len(args) > 0 {
		// Test specific provider
		p := cc.Cfg.GetProvider(args[0])
		if p == nil {
			return fmt.Errorf("provider not found: %s", args[0])
		}
		providersToTest = []*config.Provider{p}
	} else {
		// Test all configured providers
		providersToTest = cc.Cfg.Providers
	}

	if len(providersToTest) == 0 {
		ui.Warning("No providers to test")
		return nil
	}

	// JSON output
	if cc.Cfg.OutputFormat == config.FormatJSON {
		results := make([]map[string]any, 0, len(providersToTest))

		for _, p := range providersToTest {
			result := testProvider(p)
			results = append(results, map[string]any{
				"name":        p.Name,
				"reachable":   result.reachable,
				"status_code": result.statusCode,
				"error":       result.errMsg,
			})
		}

		return cc.Output(map[string]any{"results": results})
	}

	// Plain output
	if cc.Cfg.OutputFormat == config.FormatPlain {
		for _, p := range providersToTest {
			result := testProvider(p)
			status := "ok"
			if !result.reachable {
				status = "fail"
			}
			fmt.Printf("%s: %s\n", p.Name, status)
		}
		return nil
	}

	// Human-readable output
	fmt.Println()
	ui.Log("%s", ui.Bold("Testing Providers"))
	ui.Separator(40)

	ok, fail, skip := 0, 0, 0

	for _, p := range providersToTest {
		// Check if configured
		if p.NeedsAPIKey() && p.GetAPIKey() == "" {
			fmt.Printf("  Testing %-15s %s\n", p.Name, ui.Yellow("not configured"))
			fail++
			continue
		}

		// Get test URL
		if p.BaseURL == "" {
			// Native provider
			if p.Type == config.ProviderTypeBuiltin && p.Name == "native" {
				// testProvider will use the default Anthropic URL
			} else {
				fmt.Printf("  Testing %-15s %s\n", p.Name, ui.DimString("skipped"))
				skip++
				continue
			}
		}

		// Test connectivity
		result := testProvider(p)

		if result.reachable {
			fmt.Printf("  Testing %-15s %s %s\n", p.Name, ui.Green(ui.Sym.OK+" reachable"), ui.DimString(fmt.Sprintf("(HTTP %d)", result.statusCode)))
			ok++
		} else {
			if result.errMsg != "" {
				fmt.Printf("  Testing %-15s %s (%s)\n", p.Name, ui.Red(ui.Sym.Error+" unreachable"), result.errMsg)
			} else {
				fmt.Printf("  Testing %-15s %s\n", p.Name, ui.Red(ui.Sym.Error+" unreachable"))
			}
			fail++
		}
	}

	fmt.Println()
	ui.Log("Results: %s, %s", ui.Green(fmt.Sprintf("%d reachable", ok)), ui.Red(fmt.Sprintf("%d failed", fail)))
	if skip > 0 {
		ui.Dim(", %d skipped\n", skip)
	}

	return nil
}

type testResult struct {
	reachable  bool
	statusCode int
	errMsg     string
}

func testProvider(p *config.Provider) testResult {
	testURL := p.BaseURL
	if testURL == "" {
		if p.Type == config.ProviderTypeBuiltin && p.Name == "native" {
			testURL = "https://api.anthropic.com"
		} else {
			return testResult{reachable: false, errMsg: "no URL to test"}
		}
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	// Make request
	resp, err := client.Get(testURL)
	if err != nil {
		return testResult{reachable: false, errMsg: err.Error()}
	}
	defer resp.Body.Close()

	// Any HTTP response means reachable
	return testResult{
		reachable:  true,
		statusCode: resp.StatusCode,
	}
}
