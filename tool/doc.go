// Package tool is the tool contract.
// Stability: stable
//
// New builds a Tool from a typed function: the input JSON Schema is derived
// from the input type, and Run validates and decodes the model's arguments
// before calling it. A tool that changes files reports Diffs, and every
// tool may title its calls (Describer) for the client's display.
//
// Wire structs use encoding/json/v2. Optional fields are tagged omitzero.
package tool
