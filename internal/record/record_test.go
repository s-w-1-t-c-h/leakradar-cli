package record

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSanitize_NeutralizesPathTraversal(t *testing.T) {
	cases := []struct{ in, want string }{
		{"example.com", "example.com"},
		{"user@example.com", "user@example.com"},
		{"../../etc/passwd", ".._.._etc_passwd"},
		{"..", "__"},
		{".", "_"},
		{"a/b\\c", "a_b_c"},
		{`weird"name<>:|?*`, "weird_name______"},
	}
	for _, c := range cases {
		got := Sanitize(c.in)
		if got != c.want {
			t.Errorf("Sanitize(%q) = %q, want %q", c.in, got, c.want)
		}
		if strings.ContainsAny(got, `<>:"/\|?*`) {
			t.Errorf("Sanitize(%q) = %q still contains an unsafe character", c.in, got)
		}
	}
}

func TestDomainPath_StaysUnderDomainsDir(t *testing.T) {
	// "../../evil" sanitizes to a single path segment with no slashes left
	// (".._.._evil"), so even though it still contains the substring "..",
	// filepath.Clean can't interpret it as a parent-directory reference —
	// only a literal ".." segment (no surrounding characters) does that.
	got := DomainPath("../../evil", "report.json")
	if !strings.HasPrefix(got, "domains"+string(filepath.Separator)) {
		t.Fatalf("DomainPath escaped its root: %q", got)
	}
	segments := strings.Split(got, string(filepath.Separator))
	for _, seg := range segments {
		if seg == ".." {
			t.Fatalf("DomainPath produced a literal '..' path segment: %q", got)
		}
	}
}

func TestRecorder_DisabledIsNoop(t *testing.T) {
	r := New("")
	if r.Enabled() {
		t.Fatal("empty root should be disabled")
	}
	// Must not panic or create files.
	r.Record("domains/example.com/report.json", "domain-report", "example.com", "leakradar-cli domain example.com", map[string]int{"a": 1}, "some table\n")
	r.RecordFile("domain-pdf", "example.com", "leakradar-cli domain pdf example.com", "/tmp/whatever.pdf")
}

func TestRecorder_SaveJSONAndAudit(t *testing.T) {
	dir := t.TempDir()
	r := New(dir)

	type payload struct {
		Total int `json:"total"`
	}
	r.Record(DomainPath("example.com", "report.json"), "domain-report", "example.com", "leakradar-cli domain example.com", payload{Total: 3}, "CATEGORY  COMPROMISED\nemployees 3\n")

	resultPath := filepath.Join(dir, "domains", "example.com", "report.json")
	data, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("result file not written: %v", err)
	}
	var got payload
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("decoding result file: %v", err)
	}
	if got.Total != 3 {
		t.Fatalf("got.Total = %d, want 3", got.Total)
	}

	textPath := filepath.Join(dir, "domains", "example.com", "report.txt")
	textData, err := os.ReadFile(textPath)
	if err != nil {
		t.Fatalf("text rendering not written: %v", err)
	}
	if !strings.Contains(string(textData), "employees 3") {
		t.Fatalf("unexpected text file content: %q", textData)
	}

	auditData, err := os.ReadFile(filepath.Join(dir, "audit.jsonl"))
	if err != nil {
		t.Fatalf("audit log not written: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(auditData)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 audit line, got %d: %q", len(lines), auditData)
	}
	var entry auditEntry
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("decoding audit line: %v", err)
	}
	if entry.Kind != "domain-report" || entry.Target != "example.com" || entry.OutFile != resultPath || entry.OutFileText != textPath {
		t.Fatalf("unexpected audit entry: %+v", entry)
	}
}

// TestRecorder_NoTextWhenEmpty confirms passing "" for text (the common case
// for small structured payloads with no natural table view) skips the .txt
// file and audit's out_file_text entirely, rather than writing an empty file.
func TestRecorder_NoTextWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	r := New(dir)

	r.Record(TimestampPath("exports", "email"), "export-email", "user@example.com", "leakradar-cli export email user@example.com", map[string]string{"status": "queued"}, "")

	txtFiles, _ := filepath.Glob(filepath.Join(dir, "exports", "*.txt"))
	if len(txtFiles) != 0 {
		t.Fatalf("expected no .txt file for empty text, found: %v", txtFiles)
	}

	auditData, err := os.ReadFile(filepath.Join(dir, "audit.jsonl"))
	if err != nil {
		t.Fatalf("audit log not written: %v", err)
	}
	var entry auditEntry
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(auditData))), &entry); err != nil {
		t.Fatalf("decoding audit line: %v", err)
	}
	if entry.OutFileText != "" {
		t.Fatalf("expected empty OutFileText, got %q", entry.OutFileText)
	}
}

func TestRecorder_AuditAppendsAcrossMultipleCalls(t *testing.T) {
	dir := t.TempDir()
	r := New(dir)

	r.Record(TimestampPath("advanced", "search"), "advanced-search", "", "leakradar-cli advanced --url-domain example.com", map[string]int{"total": 1}, "")
	r.Record(TimestampPath("advanced", "search"), "advanced-search", "", "leakradar-cli advanced --url-domain example.com", map[string]int{"total": 2}, "")

	auditData, err := os.ReadFile(filepath.Join(dir, "audit.jsonl"))
	if err != nil {
		t.Fatalf("audit log not written: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(auditData)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 audit lines after 2 recordings, got %d", len(lines))
	}
}

func TestRecorder_UsesPrivatePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows permissions are ACL-based")
	}
	dir := t.TempDir()
	r := New(dir)
	rel := DomainPath("example.com", "report.json")
	result := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(result), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(result, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "audit.jsonl"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	r.Record(rel, "domain-report", "example.com", "leakradar-cli domain example.com", map[string]int{"total": 1}, "secret\n")

	for _, path := range []string{result, filepath.Join(dir, "domains", "example.com", "report.txt"), filepath.Join(dir, "audit.jsonl")} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != privateFileMode {
			t.Errorf("%s mode = %#o, want %#o", path, got, privateFileMode)
		}
	}
	for _, path := range []string{dir, filepath.Join(dir, "domains"), filepath.Join(dir, "domains", "example.com")} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != privateDirMode {
			t.Errorf("%s mode = %#o, want %#o", path, got, privateDirMode)
		}
	}
}

func TestRecorder_DoesNotFollowResultSymlink(t *testing.T) {
	dir := t.TempDir()
	targetDir := t.TempDir()
	target := filepath.Join(targetDir, "target")
	if err := os.WriteFile(target, []byte("unchanged"), 0o600); err != nil {
		t.Fatal(err)
	}
	resultDir := filepath.Join(dir, "domains", "example.com")
	if err := os.MkdirAll(resultDir, 0o700); err != nil {
		t.Fatal(err)
	}
	result := filepath.Join(resultDir, "report.json")
	if err := os.Symlink(target, result); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	r := New(dir)
	if _, err := r.SaveJSON(DomainPath("example.com", "report.json"), map[string]int{"total": 1}); err != nil {
		t.Fatalf("SaveJSON: %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "unchanged" {
		t.Fatalf("symlink target was overwritten: %q", data)
	}
	info, err := os.Lstat(result)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("result path remained a symlink")
	}
}

func TestRecorder_RejectsAuditSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(target, []byte("unchanged"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "audit.jsonl")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	r := New(dir)
	if err := r.LogAudit("test", "", "leakradar-cli test", "", ""); err == nil {
		t.Fatal("expected symlinked audit log to be rejected")
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "unchanged" {
		t.Fatalf("audit symlink target was modified: %q", data)
	}
}

func TestSaveRawToDoesNotFollowSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(target, []byte("unchanged"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "report.pdf")
	if err := os.Symlink(target, out); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := SaveRawTo(out, strings.NewReader("new data")); err != nil {
		t.Fatalf("SaveRawTo: %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "unchanged" {
		t.Fatalf("raw output symlink target was overwritten: %q", data)
	}
	data, err = os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new data" {
		t.Fatalf("raw output = %q", data)
	}
}

func TestRecorderSaveRawRejectsDirectorySymlink(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "domains"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "domains", "example.com")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	r := New(dir)
	if _, err := r.SaveRaw(DomainPath("example.com", "report.pdf"), strings.NewReader("data")); err == nil {
		t.Fatal("expected symlinked output directory to be rejected")
	}
	if _, err := os.Stat(filepath.Join(outside, "report.pdf")); !os.IsNotExist(err) {
		t.Fatalf("file was written outside output root: %v", err)
	}
}

func TestRecorderRejectsPathOutsideRoot(t *testing.T) {
	r := New(t.TempDir())
	if _, err := r.SaveJSON("../escape.json", map[string]int{"x": 1}); err == nil {
		t.Fatal("expected escaping result path to be rejected")
	}
}

func TestDefaultPathRejectsPathOutsideRoot(t *testing.T) {
	r := New(t.TempDir())
	if got := r.DefaultPath("../escape.json"); got != "" {
		t.Fatalf("DefaultPath returned escaping path %q", got)
	}
}
