package main

import (
	"fmt"
	"github.com/yddeng/gate/client"
	"github.com/yddeng/smux"
	"sync"
)

type clientDispatcher struct {
	streams    map[uint16]*smux.Stream
	streamLock sync.Mutex
}

func (this *clientDispatcher) OnNewStream(stream *smux.Stream) {
	this.streamLock.Lock()
	if _, ok := this.streams[stream.StreamID()]; !ok {
		fmt.Println("new stream", stream.StreamID())
		this.streams[stream.StreamID()] = stream
		go this.handleStream(stream)
	} else {
		stream.Close()
		fmt.Println(stream.StreamID(), "is already in. ")
	}
	this.streamLock.Unlock()
}

func (this *clientDispatcher) OnSessionClose(err error) {
	fmt.Println(err)
}

func (this *clientDispatcher) closeStream(stream *smux.Stream) {
	this.streamLock.Lock()
	stream.Close()
	delete(this.streams, stream.StreamID())
	this.streamLock.Unlock()
}

func (this *clientDispatcher) handleStream(stream *smux.Stream) {
	buf := make([]byte, 1024)
	for {
		n, err := stream.Read(buf)
		if err != nil {
			fmt.Println(44, err)
			this.closeStream(stream)
			return
		}

		fmt.Println("read from", stream.StreamID(), buf[:n])
		if _, err = stream.Write(buf[:n]); err != nil {
			fmt.Println(err)
			this.closeStream(stream)
			return
		}
	}
}

func main() {
	dis := &clientDispatcher{
		streams:    map[uint16]*smux.Stream{},
		streamLock: sync.Mutex{},
	}

	if _, err := client.NewClient("127.0.0.1:4784", dis); err != nil {
		panic(err)
	}

	select {}

}
