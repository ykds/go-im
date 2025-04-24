package logstashwriter

import "net"

type LogstashWriter struct {
	conn net.Conn
}

func NewLogstashWriter(addr string) (*LogstashWriter, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &LogstashWriter{conn: conn}, nil
}

func (l *LogstashWriter) Write(p []byte) (n int, err error) {
	return l.conn.Write(p)
}

func (l *LogstashWriter) Close() error {
	return l.conn.Close()
}
