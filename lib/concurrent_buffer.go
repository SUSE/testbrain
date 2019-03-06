package lib

import (
	"bytes"
	"io"
	"sync"
)

// ConcurrentBuffer is a Buffer that can be used concurrently on multiple goroutines.
type ConcurrentBuffer struct {
	mutex sync.Mutex
	buf   io.ReadWriter
}

// NewConcurrentBuffer constructs a new ConcurrentBuffer.
func NewConcurrentBuffer() *ConcurrentBuffer {
	return &ConcurrentBuffer{
		buf: new(bytes.Buffer),
	}
}

func (crw *ConcurrentBuffer) Read(p []byte) (int, error) {
	crw.mutex.Lock()
	defer crw.mutex.Unlock()
	return crw.buf.Read(p)
}

func (crw *ConcurrentBuffer) Write(b []byte) (int, error) {
	crw.mutex.Lock()
	defer crw.mutex.Unlock()
	return crw.buf.Write(b)
}
