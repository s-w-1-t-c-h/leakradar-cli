// Package record persists command results to an organised on-disk tree —
// domains/<domain>/*.json, emails/<email>/*.json, timestamped files for
// non-targeted queries, and a running audit.jsonl of every recorded
// command — so an engagement leaves a structured evidence trail instead of
// only whatever scrolled past in the terminal. Entirely opt-in: a Recorder
// with an empty Root is a no-op, so normal stdout/--json behaviour is
// unaffected when no --outdir is configured.
package record

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	privateDirMode  = 0o700
	privateFileMode = 0o600
)

// Recorder writes results under Root. A zero-value/empty-Root Recorder is
// disabled — every method becomes a no-op so callers don't need to branch
// on Enabled() themselves except where it changes control flow (e.g.
// choosing a default --out path).
type Recorder struct {
	Root string
}

// New builds a Recorder rooted at root. Passing "" yields a disabled recorder.
func New(root string) *Recorder {
	return &Recorder{Root: root}
}

// Enabled reports whether a root directory is configured.
func (r *Recorder) Enabled() bool {
	return r != nil && r.Root != ""
}

// DomainPath builds the relative path for a domain-scoped result file.
func DomainPath(domain, filename string) string {
	return filepath.Join("domains", Sanitize(domain), filename)
}

// EmailPath builds the relative path for an email/username-scoped result file.
func EmailPath(email, filename string) string {
	return filepath.Join("emails", Sanitize(email), filename)
}

// TargetPath builds the relative path for any other single-key target (e.g.
// a password-hash prefix).
func TargetPath(kind, target, filename string) string {
	return filepath.Join(kind, Sanitize(target), filename)
}

// TimestampPath builds a relative path for a non-targeted or multi-filter
// query (advanced search, dark-web search, batch checks, exports, profile
// snapshots): <kind>/<UTC timestamp>[_<suffix>].json.
func TimestampPath(kind, suffix string) string {
	name := time.Now().UTC().Format("20060102-150405")
	if suffix != "" {
		name += "_" + suffix
	}
	return filepath.Join(kind, name+".json")
}

// Sanitize replaces characters that are unsafe in a filename on Windows,
// macOS, or Linux with "_", so a domain/email/hash used as a path segment
// can never escape the intended directory or produce an invalid path.
func Sanitize(s string) string {
	clean := strings.Map(func(r rune) rune {
		switch {
		case r < 0x20:
			return '_'
		case strings.ContainsRune(`<>:"/\|?*`, r):
			return '_'
		default:
			return r
		}
	}, s)
	if clean == "." || clean == ".." {
		return strings.Repeat("_", len(clean))
	}
	return clean
}

// SaveJSON writes v as indented JSON to Root/relPath, creating parent
// directories as needed, and returns the full path written.
func (r *Recorder) SaveJSON(relPath string, v interface{}) (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encoding %s: %w", relPath, err)
	}
	root, clean, err := r.secureRoot(relPath)
	if err != nil {
		return "", err
	}
	defer root.Close()
	if err := writeAtomic(root, clean, bytes.NewReader(data)); err != nil {
		return "", fmt.Errorf("writing %s: %w", relPath, err)
	}
	return filepath.Join(r.Root, clean), nil
}

// auditEntry is one line of Root/audit.jsonl.
type auditEntry struct {
	Timestamp   string `json:"timestamp"`
	Command     string `json:"command"`
	Kind        string `json:"kind"`
	Target      string `json:"target,omitempty"`
	OutFile     string `json:"out_file,omitempty"`
	OutFileText string `json:"out_file_text,omitempty"`
}

// LogAudit appends one line to Root/audit.jsonl describing a recorded
// command. outFileText is optional (pass "" when there's no companion
// human-readable rendering, e.g. a raw PDF/CSV download).
func (r *Recorder) LogAudit(kind, target, commandLine, outFile, outFileText string) error {
	entry := auditEntry{
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Command:     commandLine,
		Kind:        kind,
		Target:      target,
		OutFile:     outFile,
		OutFileText: outFileText,
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("encoding audit entry: %w", err)
	}
	root, clean, err := r.secureRoot("audit.jsonl")
	if err != nil {
		return err
	}
	defer root.Close()
	if info, err := root.Lstat(clean); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("audit log must not be a symbolic link")
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("checking audit log: %w", err)
	}
	f, err := root.OpenFile(clean, os.O_APPEND|os.O_CREATE|os.O_WRONLY, privateFileMode)
	if err != nil {
		return fmt.Errorf("opening audit log: %w", err)
	}
	defer f.Close()
	if err := f.Chmod(privateFileMode); err != nil {
		return fmt.Errorf("restricting audit log permissions: %w", err)
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("writing audit log: %w", err)
	}
	return nil
}

// SaveText writes text (an already-rendered human-readable view, typically
// the same table shown on stdout) to Root/<relPath with its extension
// swapped for .txt>, creating parent directories as needed.
func (r *Recorder) SaveText(relPath, text string) (string, error) {
	txtRel := strings.TrimSuffix(relPath, filepath.Ext(relPath)) + ".txt"
	root, clean, err := r.secureRoot(txtRel)
	if err != nil {
		return "", err
	}
	defer root.Close()
	if err := writeAtomic(root, clean, strings.NewReader(text)); err != nil {
		return "", fmt.Errorf("writing %s: %w", txtRel, err)
	}
	return filepath.Join(r.Root, clean), nil
}

// SaveRaw streams a download to a path beneath Root using the same anchored,
// private, atomic write path as structured results.
func (r *Recorder) SaveRaw(relPath string, body io.Reader) (string, error) {
	root, clean, err := r.secureRoot(relPath)
	if err != nil {
		return "", err
	}
	defer root.Close()
	if err := writeAtomic(root, clean, body); err != nil {
		return "", fmt.Errorf("writing %s: %w", relPath, err)
	}
	return filepath.Join(r.Root, clean), nil
}

// Record saves v as JSON at relPath, optionally saves text as a companion
// .txt rendering alongside it, and appends one audit line covering both.
// Pass text == "" when there's no meaningful table/text view to save (e.g.
// a small queued-job response). Best-effort: failures are reported on
// stderr rather than returned, so a recording problem never fails the
// primary command.
func (r *Recorder) Record(relPath, kind, target, commandLine string, v interface{}, text string) {
	if !r.Enabled() {
		return
	}
	jsonPath, err := r.SaveJSON(relPath, v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not save result under %s: %v\n", r.Root, err)
		return
	}
	var textPath string
	if text != "" {
		if p, err := r.SaveText(relPath, text); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not save text rendering under %s: %v\n", r.Root, err)
		} else {
			textPath = p
		}
	}
	if err := r.LogAudit(kind, target, commandLine, jsonPath, textPath); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not write audit log: %v\n", err)
	}
}

// RecordFile appends an audit line pointing at outFile, for commands whose
// output is already a file written elsewhere (PDF report, streamed CSV/TXT
// export) rather than a JSON value this package should serialise itself.
func (r *Recorder) RecordFile(kind, target, commandLine, outFile string) {
	if !r.Enabled() {
		return
	}
	if err := r.LogAudit(kind, target, commandLine, outFile, ""); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not write audit log: %v\n", err)
	}
}

// DefaultPath resolves relPath against Root, for callers choosing a default
// --out destination when the recorder is enabled and the user didn't pass
// an explicit path. Callers must check Enabled() first.
func (r *Recorder) DefaultPath(relPath string) string {
	clean := filepath.Clean(relPath)
	if !filepath.IsLocal(clean) || clean == "." {
		return ""
	}
	return filepath.Join(r.Root, clean)
}

// SaveRawTo copies body to path, creating parent directories as needed.
// Shared by callers that need a MkdirAll'd write outside the JSON path
// (kept here so record and cmd/util agree on directory-creation behaviour).
func SaveRawTo(path string, body io.Reader) error {
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, privateDirMode); err != nil {
		return fmt.Errorf("creating directory for %s: %w", path, err)
	}
	root, err := openStableRoot(parent, false)
	if err != nil {
		return fmt.Errorf("opening directory for %s: %w", path, err)
	}
	defer root.Close()
	name := filepath.Base(filepath.Clean(path))
	if name == "." || name == string(filepath.Separator) {
		return fmt.Errorf("invalid output file path %s", path)
	}
	if err := writeAtomic(root, name, body); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

func (r *Recorder) secureRoot(relPath string) (*os.Root, string, error) {
	clean := filepath.Clean(relPath)
	if !filepath.IsLocal(clean) || clean == "." {
		return nil, "", fmt.Errorf("result path %q escapes output root", relPath)
	}
	if err := os.MkdirAll(r.Root, privateDirMode); err != nil {
		return nil, "", fmt.Errorf("creating output root: %w", err)
	}
	root, err := openStableRoot(r.Root, true)
	if err != nil {
		return nil, "", fmt.Errorf("opening output root: %w", err)
	}
	if err := preparePrivateDirs(root, filepath.Dir(clean)); err != nil {
		root.Close()
		return nil, "", fmt.Errorf("creating directory for %s: %w", relPath, err)
	}
	return root, clean, nil
}

func openStableRoot(path string, makePrivate bool) (*os.Root, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
		return nil, fmt.Errorf("%s is not a real directory", path)
	}
	dir, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	opened, err := dir.Stat()
	if err != nil {
		dir.Close()
		return nil, err
	}
	if !os.SameFile(before, opened) {
		dir.Close()
		return nil, fmt.Errorf("directory changed while opening it")
	}
	if makePrivate {
		if err := dir.Chmod(privateDirMode); err != nil {
			dir.Close()
			return nil, fmt.Errorf("restricting %s permissions: %w", path, err)
		}
	}
	if err := dir.Close(); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, err
	}
	after, err := root.Stat(".")
	if err != nil {
		root.Close()
		return nil, err
	}
	if !os.SameFile(before, after) {
		root.Close()
		return nil, fmt.Errorf("directory changed while opening it")
	}
	return root, nil
}

func preparePrivateDirs(root *os.Root, dir string) error {
	if dir == "." {
		return nil
	}
	if err := root.MkdirAll(dir, privateDirMode); err != nil {
		return err
	}
	current := ""
	for _, part := range strings.Split(dir, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, err := root.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("%s is not a real directory", current)
		}
		dir, err := root.Open(current)
		if err != nil {
			return err
		}
		opened, err := dir.Stat()
		if err != nil {
			dir.Close()
			return err
		}
		if !os.SameFile(info, opened) {
			dir.Close()
			return fmt.Errorf("%s changed while opening it", current)
		}
		if err := dir.Chmod(privateDirMode); err != nil {
			dir.Close()
			return err
		}
		if err := dir.Close(); err != nil {
			return err
		}
	}
	return nil
}

func writeAtomic(root *os.Root, name string, body io.Reader) error {
	dir := filepath.Dir(name)
	base := filepath.Base(name)
	var f *os.File
	var temp string
	for attempts := 0; attempts < 10; attempts++ {
		random := make([]byte, 12)
		if _, err := rand.Read(random); err != nil {
			return fmt.Errorf("generating temporary filename: %w", err)
		}
		temp = filepath.Join(dir, "."+base+"."+hex.EncodeToString(random)+".tmp")
		var err error
		f, err = root.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, privateFileMode)
		if err == nil {
			break
		}
		if !os.IsExist(err) {
			return err
		}
	}
	if f == nil {
		return fmt.Errorf("could not create a unique temporary file")
	}
	keep := false
	defer func() {
		if !keep {
			_ = root.Remove(temp)
		}
	}()
	if _, err := io.Copy(f, body); err != nil {
		f.Close()
		return err
	}
	if err := f.Chmod(privateFileMode); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := root.Rename(temp, name); err != nil {
		return err
	}
	keep = true
	return nil
}
