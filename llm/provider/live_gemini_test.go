//go:build live_gemini

package provider_test

import "testing"

// TestLiveGemini sends one short prompt to Gemini (0005-PLAN F4-1a).
func TestLiveGemini(t *testing.T) { liveTurn(t, "gemini", "gemini-3.5-flash-lite", "GEMINI_API_KEY") }
