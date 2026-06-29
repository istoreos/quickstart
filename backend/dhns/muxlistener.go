package dhns

import (
	"errors"
	"io"
	"net"
	stdsync "sync"
)

type MuxListener struct {
	mu    stdsync.Mutex
	addr  net.Addr
	conns chan net.Conn
	dieCh chan struct{}
}

func NewMuxListener(addr net.Addr) *MuxListener {
	return &MuxListener{
		addr:  addr,
		conns: make(chan net.Conn),
		dieCh: make(chan struct{}),
	}
}

func (ml *MuxListener) Addr() net.Addr {
	return ml.addr
}

func (ml *MuxListener) Close() error {
	ml.mu.Lock()
	select {
	case <-ml.dieCh:
		ml.mu.Unlock()
		return errors.New("already die")
	default:
		close(ml.dieCh)
		ml.mu.Unlock()
	}
	return nil
}

func (ml *MuxListener) Accept() (net.Conn, error) {
	select {
	case c := <-ml.conns:
		return c, nil
	case <-ml.dieCh:
		return nil, io.EOF
	}
}

func (ml *MuxListener) PutConn(conn net.Conn) error {
	select {
	case <-ml.dieCh:
		return io.EOF
	case ml.conns <- conn:
		return nil
	}
}
