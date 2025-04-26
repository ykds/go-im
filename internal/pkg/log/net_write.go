package log

import (
	"go-im/internal/pkg/utils"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

var _ io.Writer = (*NetWriter)(nil)
var _ io.Closer = (*NetWriter)(nil)

const (
	defaultBatchSize    = 100
	defaultBatchBytes   = 1024 * 1024
	defaultBatchTimeout = 1
)

type Option func(*NetWriter)

func WithBatchSize(size int) Option {
	if size <= 0 {
		size = defaultBatchSize
	}
	return func(w *NetWriter) {
		w.batchSize = size
	}
}

func WithBatchTimeout(timeout int) Option {
	if timeout <= 0 {
		timeout = defaultBatchTimeout
	}
	return func(w *NetWriter) {
		w.batchTimeout = timeout
	}
}

func WithBatchBytes(bytes int) Option {
	if bytes <= 0 {
		bytes = defaultBatchBytes
	}
	return func(w *NetWriter) {
		w.batchBytes = bytes
	}
}

type NetWriter struct {
	buffer [][]byte
	bytes  int
	count  int

	batchSize    int
	batchBytes   int
	batchTimeout int

	m sync.Mutex

	conn net.Conn
}

func NewNetWriter(conn net.Conn, opts ...Option) *NetWriter {
	w := &NetWriter{
		buffer:       make([][]byte, 0, defaultBatchSize),
		batchSize:    defaultBatchSize,
		batchTimeout: defaultBatchTimeout,
		m:            sync.Mutex{},
		conn:         conn,
	}
	for _, opt := range opts {
		opt(w)
	}
	utils.SafeGo(func() {
		if w.batchTimeout <= 0 {
			return
		}
		t := time.NewTicker(time.Duration(w.batchTimeout) * time.Second)
		for range t.C {
			w.m.Lock()
			if len(w.buffer) > 0 {
				newBuffer := make([]byte, 0, w.bytes)
				for _, b := range w.buffer {
					newBuffer = append(newBuffer, b...)
				}
				w.bytes = 0
				w.count = 0
				w.buffer = make([][]byte, 0, w.batchSize)
				w.m.Unlock()
				_, err := w.conn.Write(newBuffer)
				if err != nil {
					log.Printf("write log to network error: %v", err)
				}
			} else {
				w.m.Unlock()
			}
		}
	})
	return w
}

func (w *NetWriter) Write(p []byte) (n int, err error) {
	if w.batchBytes <= 0 || w.batchSize <= 0 {
		return w.conn.Write(p)
	}
	w.m.Lock()
	w.buffer = append(w.buffer, p)
	plen := len(p)
	w.bytes += plen
	w.count++
	if w.count >= w.batchSize || w.bytes >= w.batchBytes {
		newBuffer := make([]byte, 0, w.bytes)
		for _, b := range w.buffer {
			newBuffer = append(newBuffer, b...)
		}
		w.bytes = 0
		w.count = 0
		w.buffer = make([][]byte, 0, w.batchSize)
		w.m.Unlock()
		_, err := w.conn.Write(newBuffer)
		if err != nil {
			log.Printf("write log to network error: %v", err)
		}
	} else {
		w.m.Unlock()
	}
	return plen, nil
}

func (w *NetWriter) Close() error {
	w.m.Lock()
	defer w.m.Unlock()
	w.buffer = nil
	return w.conn.Close()
}
