// Package runtime provides the WebAssembly plugin execution engine.
// It is designed to work with either wazero (recommended) or a lightweight
// custom interpreter — currently stubbed for stdlib-only operation.
package runtime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────
// Errors
// ─────────────────────────────────────────────────────────────

var (
	ErrPluginNotFound = errors.New("plugin not found")
	ErrPluginTimeout  = errors.New("plugin execution timed out")
	ErrMemoryExceeded = errors.New("plugin exceeded memory limit")
	ErrInvalidPlugin  = errors.New("invalid wasm module")
)

// ─────────────────────────────────────────────────────────────
// Plugin descriptor
// ─────────────────────────────────────────────────────────────

// PluginMeta holds metadata parsed from the wasm module's custom section.
type PluginMeta struct {
	Name        string
	Version     string
	Description string
	Exports     []string // exported function names the host can call
}

// PluginConfig controls resource limits for one plugin instance.
type PluginConfig struct {
	MemoryLimitMB int           // max linear memory in MB
	Timeout       time.Duration // per-call timeout
	MaxCalls      int64         // max total calls (0 = unlimited)
	SandboxFS     string        // directory exposed as WASI /
}

// DefaultPluginConfig returns sensible safe defaults.
func DefaultPluginConfig() PluginConfig {
	return PluginConfig{
		MemoryLimitMB: 32,
		Timeout:       5 * time.Second,
		MaxCalls:      0,
		SandboxFS:     os.TempDir(),
	}
}

// ─────────────────────────────────────────────────────────────
// Host functions exposed to plugins
// ─────────────────────────────────────────────────────────────

// HostFuncs registers the functions the host exposes to wasm modules.
// Each function is accessible from wasm via the "host" import namespace.
type HostFuncs struct {
	logFn   func(level, msg string)
	kvGet   func(key string) (string, bool)
	kvSet   func(key, value string)
	httpGet func(url string) ([]byte, error)
}

// NewHostFuncs creates a default set of host functions.
func NewHostFuncs() *HostFuncs {
	kv := make(map[string]string)
	var kvMu sync.RWMutex
	return &HostFuncs{
		logFn: func(level, msg string) { fmt.Printf("[plugin][%s] %s\n", level, msg) },
		kvGet: func(key string) (string, bool) {
			kvMu.RLock()
			defer kvMu.RUnlock()
			v, ok := kv[key]
			return v, ok
		},
		kvSet: func(key, val string) {
			kvMu.Lock()
			defer kvMu.Unlock()
			kv[key] = val
		},
		httpGet: func(rawurl string) ([]byte, error) {
			// TODO: implement HTTP GET with timeout using net/http
			return nil, fmt.Errorf("host.http_get: not yet implemented")
		},
	}
}

// ─────────────────────────────────────────────────────────────
// Plugin instance
// ─────────────────────────────────────────────────────────────

// Plugin is a loaded, sandboxed WebAssembly module instance.
type Plugin struct {
	ID     string
	Meta   PluginMeta
	Config PluginConfig
	calls  int64

	// TODO: hold wazero runtime.Module here once wazero is added.
	// For now hold the raw wasm bytes for deferred instantiation.
	wasmBytes []byte
}

// Call invokes an exported function on the plugin with the given JSON arguments.
// Returns JSON-encoded output or an error.
func (p *Plugin) Call(ctx context.Context, funcName string, argsJSON []byte) ([]byte, error) {
	// TODO: check p.calls against p.Config.MaxCalls
	// TODO: create child context with p.Config.Timeout deadline
	// TODO: find funcName in module exports, marshal args to wasm linear memory,
	//       invoke function, read result from linear memory, return JSON.
	_ = ctx
	return nil, fmt.Errorf("Plugin.Call(%q): not yet implemented — add wazero first", funcName)
}

// Memory returns the current linear memory usage in bytes.
func (p *Plugin) Memory() uint32 {
	// TODO: return module.Memory().Size() once wazero is wired
	return 0
}

// ─────────────────────────────────────────────────────────────
// Engine — manages multiple plugin instances
// ─────────────────────────────────────────────────────────────

// Engine is the top-level plugin runtime.
type Engine struct {
	mu        sync.RWMutex
	plugins   map[string]*Plugin
	hostFuncs *HostFuncs
}

// NewEngine creates a new runtime engine with the given host functions.
func NewEngine(hf *HostFuncs) *Engine {
	if hf == nil {
		hf = NewHostFuncs()
	}
	return &Engine{
		plugins:   make(map[string]*Plugin),
		hostFuncs: hf,
	}
}

// Load reads a .wasm file, validates it, and registers it under id.
func (e *Engine) Load(id, path string, cfg PluginConfig) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("load %q: %w", path, err)
	}

	// Validate wasm magic bytes: \0asm
	if len(data) < 4 || string(data[:4]) != "\x00asm" {
		return ErrInvalidPlugin
	}

	meta, err := parseMeta(data)
	if err != nil {
		return fmt.Errorf("parse meta: %w", err)
	}

	// TODO: instantiate wazero module with cfg.MemoryLimitMB and cfg.SandboxFS
	//       register hostFuncs in the "host" import namespace
	//       store module reference in plugin.module

	e.mu.Lock()
	e.plugins[id] = &Plugin{ID: id, Meta: meta, Config: cfg, wasmBytes: data}
	e.mu.Unlock()
	return nil
}

// Get returns the plugin with the given id.
func (e *Engine) Get(id string) (*Plugin, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	p, ok := e.plugins[id]
	if !ok {
		return nil, ErrPluginNotFound
	}
	return p, nil
}

// Unload removes and destroys a plugin instance.
func (e *Engine) Unload(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.plugins[id]; !ok {
		return ErrPluginNotFound
	}
	// TODO: close wazero module to release memory
	delete(e.plugins, id)
	return nil
}

// List returns all loaded plugin IDs.
func (e *Engine) List() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	ids := make([]string, 0, len(e.plugins))
	for id := range e.plugins {
		ids = append(ids, id)
	}
	return ids
}

// ─────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────

// parseMeta extracts metadata from the wasm module's custom sections.
// The custom section name "metadata" is expected to contain newline-delimited
// KEY=VALUE pairs (name, version, description, exports).
func parseMeta(data []byte) (PluginMeta, error) {
	// TODO: parse wasm binary format (LEB128) to find custom sections
	// For now return a placeholder meta based on file size.
	_ = data
	return PluginMeta{
		Name:    "unknown",
		Version: "0.0.0",
		Exports: []string{"transform", "validate", "enrich"},
	}, nil
}
