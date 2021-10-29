package downloader

import (
	"io"
	"sync/atomic"
)

// Reader represents a custom reader
type Reader struct {
	io.Reader

	downloaded *uint64
}

func (r Reader) Read(b []byte) (int, error) {
	n, err := r.Reader.Read(b)
	atomic.AddUint64(r.downloaded, uint64(n))
	return n, err
}
