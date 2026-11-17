// Package mcptest is an MCP server fixture for tests, built on the go-sdk
// server: the same tools over stdio, from the test binary itself, and over
// streamable HTTP.
// Stability: beta
//
// A test package that starts the stdio fixture calls ServeStdioIfAsked
// first in its TestMain; StdioServer is then the command that runs the
// test binary as the server.
//
// NewUnionServer is another server, whose tools carry root-level unions,
// registered through Server.AddTool and listed as written, for the rule of
// 0060-MADR D2: the real schema of magictools' execute_pipeline, and one the
// rule does not cover. UnionHTTPServer serves it over streamable HTTP.
package mcptest
