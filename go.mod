module github.com/maccavelli/gobble-cli

go 1.27.1

tool (
	golang.org/x/exp/cmd/apidiff
	golang.org/x/lint/golint
	golang.org/x/vuln/cmd/govulncheck
	honnef.co/go/tools/cmd/staticcheck
)

require (
	github.com/coder/acp-go-sdk v0.13.5
	golang.org/x/sys v0.48.0
	golang.org/x/term v0.46.0
)

require (
	github.com/BurntSushi/toml v1.4.1-0.20240526193622-a339e1f7089c // indirect
	golang.org/x/exp v0.0.0-20260908205506-85c1c2202aba // indirect
	golang.org/x/exp/typeparams v0.0.0-20231108232855-2478ac86f678 // indirect
	golang.org/x/lint v0.0.0-20241112194109-818c5a804067 // indirect
	golang.org/x/mod v0.41.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/telemetry v0.0.0-20260908163034-4bcc4b2ee518 // indirect
	golang.org/x/tools v0.50.0 // indirect
	golang.org/x/vuln v1.8.0 // indirect
	honnef.co/go/tools v0.8.1 // indirect
)

replace github.com/coder/acp-go-sdk => github.com/maccavelli/acp-go-sdk v0.13.6-mcr.1
