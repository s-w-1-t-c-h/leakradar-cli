package commands

import (
	"fmt"
	"io"
	"os"

	"leakradar-cli/internal/output"
	"leakradar-cli/internal/record"
)

// writeToFile streams r to path (creating parent directories as needed).
func writeToFile(path string, r io.Reader) error {
	if err := record.SaveRawTo(path, r); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "wrote %s\n", output.TerminalSafe(path))
	return nil
}

func writeRecordedFile(path, relPath string, recorder *record.Recorder, r io.Reader) error {
	if relPath == "" {
		return writeToFile(path, r)
	}
	written, err := recorder.SaveRaw(relPath, r)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "wrote %s\n", output.TerminalSafe(written))
	return nil
}
