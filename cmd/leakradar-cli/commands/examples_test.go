package commands

import (
	"strings"
	"testing"
)

// TestAllExamples_ParseAgainstRealCommandTree resolves every example in
// exampleSections against the live cobra command tree — the same RootCmd
// every other test and the real binary use — and validates its flags and
// positional argument count. It does not execute RunE (no network calls),
// only the parsing/validation cobra does before running a command. This is
// what "syntactically correct" in the examples command's own --help text
// actually means: if a flag is renamed or removed, this test fails at
// build/test time instead of the examples silently going stale.
//
// Flag parsing writes into the same package-level option structs the real
// commands use (adv, cadv, rawOpts, ...) — harmless here, since each
// subtest only asserts whether parsing/arg-count validation succeeded, not
// any resulting value, and each cobra.Command owns its own pflag.FlagSet
// even when several commands bind flags to the same struct.
func TestAllExamples_ParseAgainstRealCommandTree(t *testing.T) {
	for _, section := range exampleSections {
		for _, ex := range section.Examples {
			ex := ex
			t.Run(section.Title+"/"+ex.Cmd, func(t *testing.T) {
				args := strings.Fields(ex.Cmd)
				if len(args) == 0 {
					t.Fatalf("empty example command")
				}

				target, remaining, err := RootCmd.Find(args)
				if err != nil {
					t.Fatalf("resolving command %q: %v", ex.Cmd, err)
				}
				if target == RootCmd {
					t.Fatalf("example %q did not resolve to a subcommand", ex.Cmd)
				}

				if err := target.ParseFlags(remaining); err != nil {
					t.Fatalf("parsing flags for %q: %v", ex.Cmd, err)
				}

				if target.Args != nil {
					if err := target.Args(target, target.Flags().Args()); err != nil {
						t.Fatalf("positional args for %q: %v", ex.Cmd, err)
					}
				}
			})
		}
	}
}
