//go:build !windows

package reader

import (
	"os"
	"syscall"
)

func mmapFile(f *os.File) ([]byte, error) {
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := info.Size()
	if size == 0 {
		return nil, nil
	}
	data, err := syscall.Mmap(
		int(f.Fd()), 0, int(size),
		syscall.PROT_READ, syscall.MAP_PRIVATE,
	)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func munmap(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return syscall.Munmap(data)
}
