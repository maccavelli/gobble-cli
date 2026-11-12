package acpext

// AuthProvider is a provider gobble's ACP authentication names (0016-MADR
// A2). ID is the id of the agent auth method gobble advertises for it. Name
// is its label. Command runs the provider's bare sign-in, or, ending in
// "--api-key -", reads a key from standard input. Login means the provider
// has a way to sign in other than an API key; it never means a device code,
// and a provider with a sign-in may still take a key, as OpenAI's offers
// its ChatGPT plan sign-in and its API key (gobble-cli 0031-MADR D7). A
// client reads each sign-in method's facts from the agent's
// _meta.gobble.methods once connected (gobble-cli 0051-MADR D7).
type AuthProvider struct {
	ID, Name, Command string
	Login             bool
}

// LoginSuffix ends a terminal auth method's id, <provider>-login. gobble
// offers terminal methods only to a client that advertises auth.terminal.
const LoginSuffix = "-login"

// AuthProviders returns the providers gobble's ACP authentication offers,
// in the order of gobble's provider registry. gobble checks that this list
// equals the one it builds from the registry, so a client may show it
// before it connects. Each call returns a new slice.
func AuthProviders() []AuthProvider {
	return []AuthProvider{
		{ID: "gemini", Name: "Gemini (Google)", Command: "gobble auth login gemini --api-key -"},
		{ID: "openai", Name: "OpenAI", Command: "gobble auth login openai", Login: true},
		{ID: "claude", Name: "Claude (Anthropic)", Command: "gobble auth login claude --api-key -"},
		{ID: "grok", Name: "Grok (xAI)", Command: "gobble auth login grok", Login: true},
		{ID: "opencode-zen", Name: "OpenCode Zen", Command: "gobble auth login opencode-zen --api-key -"},
		{ID: "opencode-go", Name: "OpenCode Go", Command: "gobble auth login opencode-go --api-key -"},
		{ID: "huggingface", Name: "Hugging Face", Command: "gobble auth login huggingface --api-key -"},
		{ID: "kilo", Name: "Kilo Gateway", Command: "gobble auth login kilo", Login: true},
		{ID: "together", Name: "Together AI", Command: "gobble auth login together --api-key -"},
	}
}
