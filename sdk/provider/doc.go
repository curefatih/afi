// Package provider documents the AFI gateway ChatProvider contract for
// multi-model extensibility.
//
// Runtime adapters live in github.com/curefatih/afi/internal/dataplane and
// register into a Registry used by the gateway pipeline. Out-of-tree adapters
// should mirror the ChatProvider interface below and register at process
// startup (blank-import or explicit Register in cmd/gateway).
//
// gRPC and WASM plugin runtimes are not shipped yet; in-process registration
// is the supported extension path. See also docs/development/providers.md.
package provider
