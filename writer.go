package miru

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const logFileName = "debug.log"

// writer handles file logging with optional folder partitioning.
type writer struct {
	mu         sync.Mutex
	outputPath string
	folderBy   FolderBy
}

func newWriter(cfg DebugConfig) *writer {
	return &writer{
		outputPath: cfg.OutputPath,
		folderBy:   cfg.FolderBy,
	}
}

// resolveDir returns the directory for the current time (possibly with date subfolder).
func (w *writer) resolveDir() (string, error) {
	base := w.outputPath
	switch w.folderBy {
	case FolderMonth:
		base = filepath.Join(base, time.Now().Format("2006-01"))
	case FolderYear:
		base = filepath.Join(base, time.Now().Format("2006"))
	}
	if err := os.MkdirAll(base, 0755); err != nil {
		return "", err
	}
	return base, nil
}

// append writes a line to the log file (and creates dir/file if needed).
func (w *writer) append(line string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	dir, err := w.resolveDir()
	if err != nil {
		return err
	}
	fpath := filepath.Join(dir, logFileName)
	f, err := os.OpenFile(fpath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(f, line)
	_ = f.Close()
	return err
}

// plainLine returns the same content as the console but without ANSI codes (for file).
func plainLine(tag, dateTime, location, rest string) string {
	return fmt.Sprintf("%s:\t%s\t%s\t->\t%s", tag, dateTime, location, rest)
}
