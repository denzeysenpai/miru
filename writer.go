package miru

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const logFileName = "debug.log"

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

// output dir, plus YYYY-MM or YYYY subdir if FolderBy is set
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

// no ANSI, for writing to file
func plainLine(tag, dateTime, location, rest string) string {
	return fmt.Sprintf("%s:\t%s\t%s\t->\t%s", tag, dateTime, location, rest)
}
