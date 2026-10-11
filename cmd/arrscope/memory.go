package main

import (
	"os"
	"runtime/debug"
	"strconv"
	"strings"
)

func memoryTarget(configured int64, containerLimit string) int64 {
	if limit, err := strconv.ParseInt(strings.TrimSpace(containerLimit), 10, 64); err == nil && limit > 0 && configured > limit/8*3 {
		return limit / 8 * 3
	}
	return configured
}

func configureMemoryLimit() {
	configured := debug.SetMemoryLimit(-1)
	if os.Getenv("GOMEMLIMIT") == "" {
		configured = 96 << 20
	}
	// Go's soft target excludes kernel page cache and some process allocations.
	// Reserve five eighths of the container budget for those, including legacy
	// Unraid templates that still set GOMEMLIMIT=192MiB.
	limit, err := os.ReadFile("/sys/fs/cgroup/memory.max")
	if err != nil {
		limit, _ = os.ReadFile("/sys/fs/cgroup/memory/memory.limit_in_bytes")
	}
	debug.SetMemoryLimit(memoryTarget(configured, string(limit)))
}
