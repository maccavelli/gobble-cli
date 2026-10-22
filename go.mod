module github.com/maccavelli/gobble-cli

go 1.27.3

tool (
	golang.org/x/exp/cmd/apidiff
	golang.org/x/lint/golint
	golang.org/x/vuln/cmd/govulncheck
	honnef.co/go/tools/cmd/staticcheck
)

require (
	github.com/alecthomas/kong v1.16.1
	github.com/google/jsonschema-go v0.4.3
	github.com/maccavelli/acp-go-sdk2 v1.20.0
	github.com/maccavelli/go-llmprovider-sdk v1.21.0
	github.com/maccavelli/go-selfupdate-lib v1.17.0
	github.com/modelcontextprotocol/go-sdk v1.8.0
	go.yaml.in/yaml/v3 v3.0.5
	golang.org/x/sys v0.49.0
	golang.org/x/term v0.47.0
	golang.org/x/text v0.43.0
	mvdan.cc/sh/v3 v3.14.1
)

require (
	github.com/BurntSushi/toml v1.4.1-0.20240526193622-a339e1f7089c // indirect
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	golang.org/x/exp v0.0.0-20261014143641-f0251eb4aa3e // indirect
	golang.org/x/exp/typeparams v0.0.0-20231108232855-2478ac86f678 // indirect
	golang.org/x/lint v0.0.0-20241112194109-818c5a804067 // indirect
	golang.org/x/mod v0.42.0 // indirect
	golang.org/x/oauth2 v0.35.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/telemetry v0.0.0-20261010193011-f71dc92da45e // indirect
	golang.org/x/time v0.15.0 // indirect
	golang.org/x/tools v0.52.0 // indirect
	golang.org/x/vuln v1.9.0 // indirect
	honnef.co/go/tools v0.8.2 // indirect
)
