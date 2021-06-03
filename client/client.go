package client

import (
	"github.com/yddeng/smux"
	"net"
)

type Client struct {
	smuxSession *smux.Session
}

func NewClient(address string, newStream func(stream *smux.Stream, err error)) (*Client, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	cli := &Client{smuxSession: smux.SmuxSession(conn)}
	go cli.start(newStream)

	return cli, nil
}

func (cli *Client) start(newStream func(stream *smux.Stream, err error)) {
	for {
		stream, err := cli.smuxSession.Accept()
		if err != nil {
			newStream(nil, err)
			return
		}
		go newStream(stream, nil)
	}
}

func (this *Client) Close() error {
	return this.smuxSession.Close()
}
