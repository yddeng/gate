package gate

import (
	"fmt"
	"github.com/yddeng/smux"
	"io"
	"net"
	"sync"
	"time"
)

type GateServer struct {
	internalListener net.Listener
	externalListener net.Listener

	clients    map[string]*client
	clientLock sync.Mutex
}

func (this *GateServer) random() *client {
	for _, cli := range this.clients {
		return cli
	}
	return nil
}

type client struct {
	smuxSession *smux.Session
	channel     map[uint16]*channel
	channelLock sync.Mutex
}

func Launch(internalAddr, externalAddr string) {
	gate := new(GateServer)
	gate.clients = map[string]*client{}

	var err error
	gate.internalListener, err = net.Listen("tcp", internalAddr)
	if err != nil {
		panic(err)
	}

	gate.externalListener, err = net.Listen("tcp", externalAddr)
	if err != nil {
		panic(err)
	}

	go listen(gate.internalListener, func(conn net.Conn) {
		// auth
		fmt.Println("new client", conn.RemoteAddr())
		cli := &client{
			smuxSession: smux.SmuxSession(conn),
			channel:     map[uint16]*channel{},
		}

		gate.clientLock.Lock()
		gate.clients[conn.RemoteAddr().String()] = cli
		gate.clientLock.Unlock()

		go cli.start(func(err error) {
			fmt.Println(62, err)
			gate.clientLock.Lock()
			delete(gate.clients, conn.RemoteAddr().String())
			gate.clientLock.Unlock()
		})
	})

	go listen(gate.externalListener, func(conn net.Conn) {
		fmt.Println("new user", conn.RemoteAddr())
		gate.clientLock.Lock()
		cli := gate.random()
		if cli == nil {
			conn.Close()
			gate.clientLock.Unlock()
			return
		}
		gate.clientLock.Unlock()

		cli.newConn(conn)
	})

}

func listen(listener net.Listener, newConn func(conn net.Conn)) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				time.Sleep(time.Millisecond * 5)
				continue
			} else {
				return
			}
		}
		go newConn(conn)
	}
}

func (cli *client) start(closeFunc func(err error)) {
	for {
		stream, err := cli.smuxSession.Accept()
		if err != nil {
			cli.smuxSession.Close()
			closeFunc(err)
			return
		}
		// 暂不允许对端开启
		stream.Close()
	}
}

func (cli *client) newConn(conn net.Conn) {
	stream, err := cli.smuxSession.Open()
	if err != nil {
		fmt.Println(err)
		conn.Close()
		return
	}

	fmt.Println("newConn", stream.StreamID(), conn.RemoteAddr())

	ch := newChannel(stream, conn)

	cli.channelLock.Lock()
	cli.channel[stream.StreamID()] = ch
	cli.channelLock.Unlock()

	ch.run(func() {
		fmt.Println("conn close", stream.StreamID(), conn.RemoteAddr())
		cli.channelLock.Lock()
		delete(cli.channel, stream.StreamID())
		cli.channelLock.Unlock()
	})
}

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
	this.close()
}

func (this *channel) handleWrite() {
	io.Copy(this.tcpConn, this.stream)
	this.close()
}
