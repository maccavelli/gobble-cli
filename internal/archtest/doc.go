// Package archtest enforces the module import boundaries.
// Stability: internal
//
// The test runs go list -e -deps -test -json ./... and checks twelve rules:
// rules 1 to 9 are 0004-MADR's import rules, as amended; rule 10 keeps
// jsonx a leaf, and rule 11 allows a []byte field with a json tag only
// where listed (0039-MADR D24); rule 12 keeps the nested module acpext
// to the ACP edge (0042-MADR D4). docs/architecture.md lists them.
package archtest
