package gateway

import (
	"github.com/yddeng/smux"
	"net"
	"sync"
	"time"
)

type Gateway struct {
	internalListener net.Listener
	externalListener net.Listener

	clients    map[string]*client
	clientLock sync.Mutex
}

type client struct {
	smuxSession *smux.Session
	channel     map[uint16]*channel
}

func Gate(internalAddr, externalAddr string) {
	gate := new(Gateway)
	gate.clients = map[string]*client{}

	var err error
	gate.internalListener, err = net.Listen("tcp", internalAddr)
	if err != nil {
		panic(err)
	}

	gate.externalListener, err = net.Listen("tcp", internalAddr)
	if err != nil {
		panic(err)
	}

	go listen(gate.internalListener, func(conn net.Conn) {
		// auth

		cli := &client{
			smuxSession: smux.SmuxSession(conn),
			channel:     map[uint16]*channel{},
		}

		gate.clientLock.Lock()
		gate.clients[conn.RemoteAddr().String()] = cli
		gate.clientLock.Unlock()

		go cli.start()
	})

	go listen(gate.externalListener, func(conn net.Conn) {

		gate.clientLock.Lock()

		length := len(gate.clients)
		if length == 0 {
			conn.Close()
			gate.clientLock.Unlock()
			return
		}
		cli := gate.clients["sf"]
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

func (cli *client) start() {
	for {
		stream, err := cli.smuxSession.Accept()
		if err != nil {
			return
		}
		stream.Close()
		// 暂不允许对端开启
	}
}

func (cli *client) newConn(conn net.Conn) {
	stream, err := cli.smuxSession.Open()
	if err != nil {
		conn.Close()
		return
	}

	ch := newChannel(stream, conn)

	cli.channel[stream.StreamID()] = ch

	go ch.run(func() {
		delete(cli.channel, stream.StreamID())
	})
}
