package main

import (
	"fmt"
	"github.com/yddeng/gateway/client"
	"github.com/yddeng/smux"
	"sync"
)

var (
	streams    = map[uint16]*smux.Stream{}
	streamLock sync.Mutex
)

func newStream(stream *smux.Stream) {
	streamLock.Lock()
	if _, ok := streams[stream.StreamID()]; !ok {
		fmt.Println("new stream", stream.StreamID())
		streams[stream.StreamID()] = stream
		go handleStream(stream)
	} else {
		stream.Close()
		fmt.Println(stream.StreamID(), "is already . ")
	}
	streamLock.Unlock()
}

func closeStream(stream *smux.Stream) {
	streamLock.Lock()
	stream.Close()
	delete(streams, stream.StreamID())
	streamLock.Unlock()
}

func handleStream(stream *smux.Stream) {
	buf := make([]byte, 1024)
	for {
		n, err := stream.Read(buf)
		if err != nil {
			fmt.Println(err)
			closeStream(stream)
			return
		}

		fmt.Println("read from", stream.StreamID(), buf[:n])
		if _, err = stream.Write(buf[:n]); err != nil {
			fmt.Println(err)
			closeStream(stream)
			return
		}
	}
}

func main() {
	_, err := client.NewClient("127.0.0.1:4784", newStream)
	if err != nil {
		panic(err)
	}

	select {}

}
