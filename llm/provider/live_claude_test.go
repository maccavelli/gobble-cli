//go:build live_claude

package provider_test

import "testing"

// TestLiveClaude sends one short prompt to Claude (0005-PLAN F4-1a).
func TestLiveClaude(t *testing.T) { liveTurn(t, "claude", "claude-haiku-5-5", "ANTHROPIC_API_KEY") }
