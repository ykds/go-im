package log

import (
	"log"
	"math/rand/v2"
	"net"
	"testing"
	"time"
)

type mockConn struct {
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	return 0, nil
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	log.Printf("%v\n", string(b))
	return 0, nil
}

func (m *mockConn) Close() error {
	return nil
}

func (m *mockConn) LocalAddr() net.Addr {
	return nil
}

func (m *mockConn) RemoteAddr() net.Addr {
	return nil
}

func (m *mockConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func TestWriter(t *testing.T) {
	w := NewNetWriter(&mockConn{}, WithBatchBytes(1024))
	for i := 0; i < 1; i++ {
		w.Write([]byte("test1"))
	}

	b := make([]byte, 1024)
	rand.ChaCha8
	w.Write(b)

	w.Write([]byte("test2"))
	time.Sleep(2 * time.Second)
}
