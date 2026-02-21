# Project 05 — WebAssembly Plugin Runtime

> **Difficulty**: Expert · **Domain**: Runtime Systems, WASM, Plugin Architecture
> **Real-world analog**: Envoy WASM filters, Shopify Script (WASM), Cloudflare Workers, OPA

---

## Why This Project Exists

Plugin systems that use `plugin.Open()` or dynamic `.so` files are fragile and OS-specific.
The industry has converged on **WebAssembly** as the universal, sandboxed, language-agnostic
plugin format. This project builds a real WASM plugin runtime where plugins are Go programs
compiled to WASM that extend a host application safely.

---

## Folder Structure

```
05-wasm-plugin-runtime/
├── go.mod
├── main.go                          # Host application CLI
├── runtime/
│   ├── engine.go                    # WASM engine wrapper (using wazero)
│   ├── plugin.go                    # Plugin lifecycle: load, call, unload
│   ├── host_funcs.go                # Host functions exposed to WASM plugins
│   ├── memory.go                    # Linear memory management (ptr passing)
│   └── sandbox.go                   # Resource limits: memory, CPU time, syscalls
├── sdk/
│   └── plugin_sdk.go                # SDK Go file that plugin authors import
├── plugins/
│   ├── transformer/
│   │   └── main.go                  # Example plugin: JSON field transformer
│   ├── validator/
│   │   └── main.go                  # Example plugin: request validator
│   └── enricher/
│       └── main.go                  # Example plugin: data enricher
├── registry/
│   └── registry.go                  # Plugin registry: name → WASM binary
└── config/
    └── plugins.yaml                 # Plugin configuration
```

---

## Technology

Use **wazero** — the pure-Go, zero-dependency WASM runtime:
```
go get github.com/tetratelabs/wazero
```

wazero is used by production systems (Dapr, TiKV, OPA) and has zero CGO requirements.

---

## Implementation Guide

### Phase 1 — Host/Plugin ABI Design (Week 1)

The hardest part of WASM plugins is **crossing the memory boundary**. WASM only understands
integers. To pass a string, you must:
1. Write the string to WASM linear memory
2. Pass the pointer (int32) + length (int32) to the WASM function
3. The WASM function reads from its linear memory using those values

**Host → Plugin ABI:**
```
// Plugin exports these functions:
transform(inputPtr, inputLen int32) int32  // returns result_ptr
get_output_len() int32                     // returns last result length
malloc(size int32) int32                   // plugin-side allocator
free(ptr int32)
```

**Plugin SDK** (Go, compiles to WASM):
```go
//go:build wasip1
package main

//export transform
func transform(inputPtr, inputLen int32) int32 { ... }
```

---

### Phase 2 — Host Function Exposure (Week 1-2)

Host functions are functions the host exposes to WASM plugins (reverse direction):

```go
// What plugins CAN call:
log(msgPtr, msgLen int32)                        // structured logging
get_config(keyPtr, keyLen, outPtr, outMax int32) // read plugin config
http_get(urlPtr, urlLen, outPtr, outMax int32) int32  // sandboxed HTTP
set_header(keyPtr, keyLen, valPtr, valLen int32) // modify response headers
```

Implement `host_funcs.go` that registers these as wazero host module functions.
The `http_get` function must enforce an **allowlist** of domains from config.

---

### Phase 3 — Resource Sandbox (Week 2)

A runaway plugin must not crash the host. Implement:

```go
type SandboxConfig struct {
    MaxMemoryPages  uint32        // 64KB per page; default 16 = 1MB
    ExecutionTimeout time.Duration // max wall time per call
    MaxHTTPCalls    int           // per invocation
    AllowedDomains  []string
}
```

**Steps**:
1. Set `wazero.RuntimeConfig.WithMemoryLimitPages(maxPages)` for memory isolation
2. Wrap each `plugin.Call()` in a `context.WithTimeout` — wazero respects context cancellation
3. Count host function calls and return error when over budget

---

### Phase 4 — Plugin Pipeline (Week 3)

Chain multiple plugins into a processing pipeline:

```
Request → [validator] → [enricher] → [transformer] → Response
```

```go
pipeline := runtime.NewPipeline()
pipeline.Add("validator",   "plugins/validator.wasm",   cfg["validator"])
pipeline.Add("enricher",    "plugins/enricher.wasm",    cfg["enricher"])
pipeline.Add("transformer", "plugins/transformer.wasm", cfg["transformer"])
result, err := pipeline.Execute(ctx, inputJSON)
```

Each plugin receives the output of the previous one as its input. If any plugin returns
an error, the pipeline short-circuits.

---

### Phase 5 — Hot Reload (Week 3-4)

Watch plugin WASM files for changes and reload without restarting the host:

```go
watcher := registry.NewWatcher(pluginDir)
watcher.OnReload(func(name string, newBinary []byte) {
    pipeline.Reload(name, newBinary)
})
watcher.Start(ctx)
```

**Steps**:
1. Poll plugin file mtimes every 2 seconds (or use `fsnotify`)
2. On change: compile new WASM module, swap in atomically under RWMutex
3. Drain in-flight calls to old module before unloading

---

### Writing Plugins (Guest SDK)

```go
//go:build wasip1
// compile: GOARCH=wasm GOOS=wasip1 go build -o validator.wasm ./plugins/validator

package main

import (
    "encoding/json"
    "unsafe"
)

//go:linkname hostLog
func hostLog(ptr, length uint32)

//export validate
func validate(inputPtr, inputLen uint32) uint32 {
    input := readMemory(inputPtr, inputLen)
    var req map[string]any
    if err := json.Unmarshal(input, &req); err != nil {
        return writeError("invalid JSON")
    }
    if _, ok := req["user_id"]; !ok {
        return writeError("missing user_id")
    }
    return writeOK(input)
}

func main() {} // required for wasip1
```

---

## Acceptance Criteria

- [ ] Plugin isolation: a panicking plugin does not crash the host
- [ ] Memory limit enforced: allocating beyond limit returns error
- [ ] Timeout enforced: infinite loop plugin is terminated within `executionTimeout`
- [ ] Pipeline of 3 plugins processes 10,000 req/sec

---

## Stretch Goals

- Implement a **WASM component model** compatible ABI
- Build a **plugin marketplace**: download and pin plugins by content hash
- Add **per-plugin tracing** using W3C trace context propagation
