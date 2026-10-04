// Package archtest enforces the module import boundaries.
// Stability: internal
//
// The test runs go list -deps -json ./... and checks the eight rules in
// 0004-MADR. Rules 4 and 8 name their edges even though those imports are
// not in the tree yet.
package archtest
