//go:build windows

// knowswell — a portable Win32 on screen AI assistant.
//
// Build with:
//
//	go build -ldflags="-H windowsgui -s -w" -o knowswell.exe .
package main

import (
	"runtime"
	"runtime/debug"

	"knowswell/internal/config"
	"knowswell/internal/ui"
)

func init() {
	runtime.LockOSThread()
	// Convert memory-access faults (nil deref, stale pointer) into Go panics
	// so the per-window recover() guards can catch them instead of crashing.
	debug.SetPanicOnFault(true)
}

func main() {
	cfg := config.Load()
	app := ui.NewApp(cfg)
	app.Run()
	_ = cfg.Save()
}
