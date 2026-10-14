//go:build live_grok

package provider_test

import "testing"

// TestLiveGrok sends one short prompt to xAI's Grok (0005-PLAN F4-1a).
func TestLiveGrok(t *testing.T) { liveTurn(t, "grok", "grok-4.6", "XAI_API_KEY") }
