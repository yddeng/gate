package gate

import (
	"encoding/binary"
	"io"
	"net"
	"time"
)

/*
	client 第一次连接，不指定ID，由gate分配。
	重连后连接上次ID的service，如果不存在，重新分配。
	用途是： 用户网络闪断，希望连接到上次登录的服务器。并且上次登录的服务器还没有执行下线流程，仍保留用户信息
*/
type Client struct {
	net.Conn
	addr  string
	id    uint32
	token string
}

func (this *Client) Dial() error {
	conn, err := net.Dial("tcp", this.addr)
	if err != nil {
		return err
	}
	if id, err := clientLogin(conn, this.id, this.token); err != nil {
		conn.Close()
		return err
	} else {
		this.Conn = conn
		this.id = id
		return nil
	}
}

func clientLogin(conn net.Conn, id uint32, token string) (uint32, error) {
	data := make([]byte, 9)
	data[0] = typeClient
	binary.BigEndian.PutUint32(data[1:], id)
	binary.BigEndian.PutUint32(data[5:], HashNumber(token))

	conn.SetWriteDeadline(time.Now().Add(time.Second * 2))
	if _, err := conn.Write(data); err != nil {
		panic(err)
	}
	conn.SetWriteDeadline(time.Time{})

	ret := make([]byte, 8)
	conn.SetReadDeadline(time.Now().Add(time.Second * 2))
	if _, err := io.ReadFull(conn, ret); err != nil {
		return 0, err
	}
	conn.SetReadDeadline(time.Time{})

	code := binary.BigEndian.Uint32(ret)
	if code != OK {
		return 0, GetCode(code)
	}
	id = binary.BigEndian.Uint32(data[4:])
	return id, nil
}

func NewClient(addr, token string) (*Client, error) {
	return &Client{addr: addr, token: token}, nil
}

func DialClient(addr string, token string) (*Client, error) {
	client, _ := NewClient(addr, token)
	return client, client.Dial()
}
