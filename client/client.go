package client

import (
	"github.com/yddeng/smux"
	"net"
)

type Dispatcher interface {
	OnNewStream(stream *smux.Stream)
	OnSessionClose(err error)
}

type Client struct {
	smuxSession *smux.Session
	dispatcher  Dispatcher
}

func NewClient(address string, dispatcher Dispatcher) (*Client, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	cli := &Client{smuxSession: smux.SmuxSession(conn), dispatcher: dispatcher}
	go cli.start()

	return cli, nil
}

func (cli *Client) start() {
	for {
		stream, err := cli.smuxSession.Accept()
		if err != nil {
			cli.smuxSession.Close()
			cli.dispatcher.OnSessionClose(err)
			return
		}
		cli.dispatcher.OnNewStream(stream)
	}
}

func (this *Client) Close() error {
	return this.smuxSession.Close()
}
