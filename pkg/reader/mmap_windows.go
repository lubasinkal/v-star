//go:build windows

package reader

import (
	"os"
	"reflect"
	"syscall"
	"unsafe"
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

	h, err := syscall.CreateFileMapping(
		syscall.Handle(f.Fd()), nil,
		syscall.PAGE_READONLY,
		uint32(size>>32), uint32(size), nil,
	)
	if err != nil {
		return nil, err
	}
	defer syscall.CloseHandle(h)

	addr, err := syscall.MapViewOfFile(
		h, syscall.FILE_MAP_READ, 0, 0, 0,
	)
	if err != nil {
		return nil, err
	}

	var data []byte
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&data))
	sh.Data = addr
	sh.Len = int(size)
	sh.Cap = int(size)
	return data, nil
}

func munmap(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&data))
	return syscall.UnmapViewOfFile(sh.Data)
}
