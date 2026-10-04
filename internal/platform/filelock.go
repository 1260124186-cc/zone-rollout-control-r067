package platform

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

var ErrAlreadyLocked = errors.New("state path is already locked")

type FileLock struct {
	file *os.File
}

func LockFile(path string) (*FileLock, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrAlreadyLocked
		}
		return nil, fmt.Errorf("acquire lock: %w", err)
	}
	return &FileLock{file: file}, nil
}

func (l *FileLock) Unlock() error {
	if l == nil || l.file == nil {
		return nil
	}
	unlockErr := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	closeErr := l.file.Close()
	l.file = nil
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}
