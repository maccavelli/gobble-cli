// Package permission is the policy contract.
// Stability: beta
//
// 0005-PLAN F6-1 adds project trust: DecideTrust's order and trust.json.
// F6-2a adds the rules: ParseSpec reads one, and Engine decides a call by
// the last that matches over its layers, the defaults before them, the
// strictest of a call's subjects, external_directory for a path outside
// the roots, and gobble's limits after them, which no rule lifts.
// Wire structs use encoding/json/v2. Optional fields are tagged omitzero.
package permission
