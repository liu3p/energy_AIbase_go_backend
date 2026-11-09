package handlers

import (
	"os"
	"path/filepath"
	"testing"
)

func withMachineSerialFile(t *testing.T, content string) {
	t.Helper()

	oldPath := machineSerialFile
	machineSerialFile = filepath.Join(t.TempDir(), "machine_serial")
	t.Cleanup(func() {
		machineSerialFile = oldPath
	})

	if content != "" {
		if err := os.WriteFile(machineSerialFile, []byte(content), 0o600); err != nil {
			t.Fatalf("write machine serial fixture: %v", err)
		}
	}
}

func TestResolveSystemSerialPrefersMachineSerial(t *testing.T) {
	withMachineSerialFile(t, "0112-2233-4455-66E2C0\n")

	if got, want := resolveSystemSerial("9876-5432-1000-4321"), "0112-2233-4455-66E2C0"; got != want {
		t.Fatalf("resolveSystemSerial() = %q, want %q", got, want)
	}
}

func TestResolveSystemSerialFallsBackWhenMachineSerialMissing(t *testing.T) {
	withMachineSerialFile(t, "")

	if got, want := resolveSystemSerial("9876-5432-1000-4321"), "9876-5432-1000-4321"; got != want {
		t.Fatalf("resolveSystemSerial() = %q, want fallback %q", got, want)
	}
}
