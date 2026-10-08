// Package tool is the tool contract.
// Stability: stable
//
// New builds a Tool from a typed function: the input JSON Schema is derived
// from the input type, and Run validates and decodes the model's arguments
// before calling it. The schema it publishes may say more than the one it
// validates: WithSchema adds bounds, defaults and minItems for the model,
// and the tool's code enforces them. A tool that changes files reports
// Diffs, and every tool may title its calls (Describer) for the client's
// display.
//
// Wire structs use encoding/json/v2. Optional fields are tagged omitzero.
package tool
