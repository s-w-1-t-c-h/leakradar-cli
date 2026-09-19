package commands

import (
	"os"
	"strings"
	"testing"
)

func TestRedactedCommandLine_HidesAPIKeyValue(t *testing.T) {
	orig := os.Args
	defer func() { os.Args = orig }()

	cases := [][]string{
		{"leakradar-cli", "profile", "--api-key", "super-secret-jwt"},
		{"leakradar-cli", "profile", "--api-key=super-secret-jwt"},
	}
	for _, args := range cases {
		os.Args = args
		got := redactedCommandLine()
		if strings.Contains(got, "super-secret-jwt") {
			t.Fatalf("redactedCommandLine() leaked the key: %q", got)
		}
		if !strings.Contains(got, "REDACTED") {
			t.Fatalf("redactedCommandLine() = %q, expected a REDACTED marker", got)
		}
	}
}

func TestRedactedCommandLine_LeavesOtherArgsIntact(t *testing.T) {
	orig := os.Args
	defer func() { os.Args = orig }()

	os.Args = []string{"leakradar-cli", "domain", "example.com", "--category", "employees"}
	got := redactedCommandLine()
	want := "leakradar-cli domain example.com --category employees"
	if got != want {
		t.Fatalf("redactedCommandLine() = %q, want %q", got, want)
	}
}
