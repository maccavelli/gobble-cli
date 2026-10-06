// Package mcptest is an MCP server fixture for tests, built on the go-sdk
// server: the same tools over stdio, from the test binary itself, and over
// streamable HTTP.
// Stability: beta
//
// A test package that starts the stdio fixture calls ServeStdioIfAsked
// first in its TestMain; StdioServer is then the command that runs the
// test binary as the server.
package mcptest
