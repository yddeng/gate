package main

import (
	"fmt"
	"github.com/yddeng/gate"
	"github.com/yddeng/smux"
	"sync"
)

type clientDispatcher struct {
	streams    map[uint16]*smux.MuxConn
	streamLock sync.Mutex
}

func (this *clientDispatcher) OnNewStream(stream *smux.MuxConn) {
	this.streamLock.Lock()
	if _, ok := this.streams[stream.ID()]; !ok {
		fmt.Println("new stream", stream.ID())
		this.streams[stream.ID()] = stream
		go this.handleStream(stream)
	} else {
		stream.Close()
		fmt.Println(stream.ID(), "is already in. ")
	}
	this.streamLock.Unlock()
}

func (this *clientDispatcher) closeStream(stream *smux.MuxConn) {
	this.streamLock.Lock()
	stream.Close()
	delete(this.streams, stream.ID())
	this.streamLock.Unlock()
}

func (this *clientDispatcher) handleStream(stream *smux.MuxConn) {
	buf := make([]byte, 1024)
	for {
		n, err := stream.Read(buf)
		if err != nil {
			fmt.Println(44, err)
			this.closeStream(stream)
			return
		}

		fmt.Println("read from", stream.ID(), buf[:n])
		if _, err = stream.Write(buf[:n]); err != nil {
			fmt.Println(err)
			this.closeStream(stream)
			return
		}
	}
}

func main() {
	dis := &clientDispatcher{
		streams:    map[uint16]*smux.MuxConn{},
		streamLock: sync.Mutex{},
	}

	serv, err := gate.DialService("127.0.0.1:4784", 0, "gatetoken")
	if err != nil {
		panic(err)
	}

	for {
		conn, err := serv.Accept()
		if err != nil {
			fmt.Println(err)
			break
		}
		dis.OnNewStream(conn)
	}

	select {}

}
