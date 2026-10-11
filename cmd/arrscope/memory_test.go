package main

import "testing"

func TestContainerMemoryHeadroom(t *testing.T) {
	for _, test := range []struct {
		name       string
		configured int64
		container  string
		want       int64
	}{
		{"legacy Unraid target", 192 << 20, "268435456\n", 96 << 20},
		{"previous default target", 128 << 20, "268435456", 96 << 20},
		{"smaller explicit target", 64 << 20, "268435456", 64 << 20},
		{"larger container", 192 << 20, "1073741824", 192 << 20},
		{"unlimited container", 192 << 20, "max", 192 << 20},
		{"native host", 128 << 20, "", 128 << 20},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := memoryTarget(test.configured, test.container); got != test.want {
				t.Fatalf("memory target = %d, want %d", got, test.want)
			}
		})
	}
}
