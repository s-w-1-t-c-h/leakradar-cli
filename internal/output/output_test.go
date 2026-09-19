package output

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/olekukonko/tablewriter"
)

func TestTerminalSafeEscapesTerminalControls(t *testing.T) {
	in := "normal\x1b]52;c;payload\a\n\r\t\u202etext"
	got := TerminalSafe(in)
	for _, forbidden := range []string{"\x1b", "\a", "\n", "\r", "\t", "\u202e"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("TerminalSafe retained control character %q in %q", forbidden, got)
		}
	}
	for _, want := range []string{`\x1b`, `\x07`, `\n`, `\r`, `\t`, `\u202e`} {
		if !strings.Contains(got, want) {
			t.Errorf("TerminalSafe output %q missing %q", got, want)
		}
	}
}

func TestTableNeutralizesControlSequences(t *testing.T) {
	var buf bytes.Buffer
	Table(&buf, []string{"VALUE"}, [][]string{{"before\x1b[2Jafter"}})
	if strings.ContainsRune(buf.String(), '\x1b') {
		t.Fatalf("table output retained an escape byte: %q", buf.String())
	}
	if !strings.Contains(buf.String(), `\x1b[2J`) {
		t.Fatalf("table output did not render the escape visibly: %q", buf.String())
	}
}

// TestTableOptionsForWidthWrapsWithoutWhitespace guards against the raw
// breach-dump case (SQL/CSV blocks with almost no spaces): word wrapping
// alone can't bound such content, so a narrow terminal must still produce
// rows no wider than its width instead of one long line the terminal itself
// hard-wraps mid-row, corrupting the box-drawing.
func TestTableOptionsForWidthWrapsWithoutWhitespace(t *testing.T) {
	header := []string{"CONTAINER", "FILE", "CATEGORY", "EXT", "SNIPPET"}
	rows := [][]string{
		{"35849", "35849.zip/Dump_public_stuffandthings_none.rar/stuffandthings.sql", "database", "sql",
			"...),(149009434,'alex.florest@gmail.com','2cUQ8BQ6wWXioxG6CatHBw==','Bancario'),(149009438,'silas.barnes@gmail.com'..."},
	}
	const width = 100
	var buf bytes.Buffer
	tt := tablewriter.NewTable(&buf, tableOptionsForWidth(width, header, rows)...)
	tt.Header(header)
	tt.Bulk(rows)
	tt.Render()

	for _, line := range strings.Split(buf.String(), "\n") {
		if n := utf8.RuneCountInString(line); n > width {
			t.Errorf("line exceeds terminal width %d (got %d): %q", width, n, line)
		}
	}
}

// TestTableOptionsForWidthKeepsShortColumnsNatural ensures columns that
// already fit their fair share of the width aren't capped just because a
// sibling column (SNIPPET) needs more room.
func TestTableOptionsForWidthKeepsShortColumnsNatural(t *testing.T) {
	header := []string{"CONTAINER", "CATEGORY", "EXT", "UNLOCKED", "SNIPPET"}
	longSnippet := strings.Repeat("nowhitespacecontenttowrapon", 10)
	rows := [][]string{{"35849", "database", "sql", "true", longSnippet}}

	opts := tableOptionsForWidth(120, header, rows)
	if len(opts) == 0 {
		t.Fatal("expected capping options for oversized content, got none")
	}
	var buf bytes.Buffer
	tt := tablewriter.NewTable(&buf, opts...)
	tt.Header(header)
	tt.Bulk(rows)
	tt.Render()

	for _, want := range []string{"CONTAINER", "CATEGORY", "EXT", "UNLOCKED", "database", "true", "sql"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("short column value %q was mangled/wrapped, not left natural:\n%s", want, buf.String())
		}
	}
}

// TestTableOptionsForWidthNoOpWhenContentFits ensures no capping is applied
// (and thus no unnecessary wrapping) when the content already fits.
func TestTableOptionsForWidthNoOpWhenContentFits(t *testing.T) {
	header := []string{"A", "B"}
	rows := [][]string{{"short", "also short"}}
	if opts := tableOptionsForWidth(200, header, rows); opts != nil {
		t.Fatalf("expected no options when content fits the width, got %v", opts)
	}
}
