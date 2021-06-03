package gateway

import (
	"github.com/yddeng/smux"
	"io"
	"net"
	"sync"
)

type channel struct {
	stream  *smux.Stream
	tcpConn net.Conn

	closeOnce sync.Once
	closeFunc func()
}

func newChannel(stream *smux.Stream, conn net.Conn) *channel {
	return &channel{stream: stream, tcpConn: conn}
}

func (this *channel) close() {
	this.closeOnce.Do(func() {
		this.closeFunc()
		this.stream.Close()
		this.tcpConn.Close()
	})
}

func (this *channel) run(closeFunc func()) {
	this.closeFunc = closeFunc
	go this.handleRead()
	go this.handleWrite()
}

func (this *channel) handleRead() {
	io.Copy(this.stream, this.tcpConn)
}

func (this *channel) handleWrite() {
	io.Copy(this.tcpConn, this.stream)
}
