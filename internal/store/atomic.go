package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func replaceFile(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create snapshot directory: %w", err)
	}
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode snapshot: %w", err)
	}
	encoded = append(encoded, '\n')

	temp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".pending-*")
	if err != nil {
		return fmt.Errorf("create pending snapshot: %w", err)
	}
	tempPath := temp.Name()
	remove := func() {
		_ = temp.Close()
		_ = os.Remove(tempPath)
	}
	if err := temp.Chmod(0o600); err != nil {
		remove()
		return fmt.Errorf("set pending snapshot mode: %w", err)
	}
	if _, err := temp.Write(encoded); err != nil {
		remove()
		return fmt.Errorf("write pending snapshot: %w", err)
	}
	if err := temp.Sync(); err != nil {
		remove()
		return fmt.Errorf("sync pending snapshot: %w", err)
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("close pending snapshot: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("replace snapshot: %w", err)
	}
	if directory, err := os.Open(filepath.Dir(path)); err == nil {
		_ = directory.Sync()
		_ = directory.Close()
	}
	return nil
}
