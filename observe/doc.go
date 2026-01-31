// Package observe provides observability primitives for tool execution.
//
// It is a pure instrumentation library: no execution, no transport, no I/O
// beyond exporter setup. Consumers wire the observer into toolrun/toolruntime
// or server middleware.
package observe
