package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type OutboxEntry struct {
	Path    string `json:"path"`
	Caption string `json:"caption,omitempty"`
	Type    string `json:"type,omitempty"` // "voice" sends as voice note; default sends as document
}

func ReadOutbox(path string) ([]OutboxEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading outbox: %w", err)
	}

	var entries []OutboxEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parsing outbox: %w", err)
	}

	return entries, nil
}

func RemoveOutbox(path string) {
	os.Remove(path)
}

func ValidateOutboxEntry(entry OutboxEntry) error {
	absPath, err := filepath.Abs(entry.Path)
	if err != nil {
		return fmt.Errorf("invalid path %q", entry.Path)
	}

	resolved, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s not found", filepath.Base(absPath))
		}
		return fmt.Errorf("resolving path: %w", err)
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return fmt.Errorf("%s not found", filepath.Base(resolved))
	}

	if info.IsDir() {
		return fmt.Errorf("cannot send a directory")
	}

	const maxSize = 50 * 1024 * 1024
	if info.Size() > maxSize {
		return fmt.Errorf("%s exceeds 50MB limit (%d MB)", filepath.Base(resolved), info.Size()/(1024*1024))
	}

	return nil
}
