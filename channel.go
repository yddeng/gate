package gate

import (
	"github.com/yddeng/smux"
	"io"
	"net"
	"sync"
)

type channel struct {
	muxConn *smux.MuxConn
	tcpConn net.Conn

	closeOnce sync.Once
	chClose   chan struct{}
	closeFunc func(err error)
}

func newChannel(muxConn *smux.MuxConn, conn net.Conn) *channel {
	return &channel{muxConn: muxConn, tcpConn: conn, chClose: make(chan struct{})}
}

func (this *channel) close(err error) {
	this.closeOnce.Do(func() {
		close(this.chClose)
		this.closeFunc(err)
		_ = this.muxConn.Close()
		_ = this.tcpConn.Close()
	})
}

func (this *channel) run(closeFunc func(err error)) {
	this.closeFunc = closeFunc
	go this.handleRead()
	go this.handleWrite()
}

func (this *channel) handleRead() {
	_, err := io.Copy(this.muxConn, this.tcpConn)
	this.close(err)
}

func (this *channel) handleWrite() {
	_, err := io.Copy(this.tcpConn, this.muxConn)
	this.close(err)
}
