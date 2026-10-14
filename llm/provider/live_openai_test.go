//go:build live_openai

package provider_test

import "testing"

// TestLiveOpenAI sends one short prompt to OpenAI (0005-PLAN F4-1a).
func TestLiveOpenAI(t *testing.T) { liveTurn(t, "openai", "gpt-6-luna", "OPENAI_API_KEY") }
