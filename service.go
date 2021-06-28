package gate

import (
	"encoding/binary"
	"github.com/yddeng/smux"
	"github.com/yddeng/utils/log"
	"io"
	"net"
	"sync"
	"time"
)

/*
	service 第一次连接，不指定ID，由gate分配。
	重连后附带ID，在gate上重建对象。
	用途是： 网络闪断，希望连接到上次登录的服务器。并且上次登录的服务器还没有执行下线流程，仍保留用户信息
*/

type Service struct {
	id         uint32
	token      string
	muxSession *smux.MuxSession

	channels map[uint16]*channel
	chanLock sync.Mutex
}

func (this *Service) Accept() (*smux.MuxConn, error) {
	return this.muxSession.Accept()
}

func (this *Service) Close() {
	this.muxSession.Close()
}

func serviceLogin(conn net.Conn, id uint32, token string) (uint32, error) {
	data := make([]byte, 9)
	data[0] = typeServer
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

func newService(conn net.Conn, id uint32, token string) *Service {
	return &Service{
		id:         id,
		token:      token,
		muxSession: smux.NewMuxSession(conn),
		channels:   map[uint16]*channel{},
	}
}

func NewService(conn net.Conn, id uint32, token string) (*Service, error) {
	if id, err := serviceLogin(conn, id, token); err != nil {
		return nil, err
	} else {
		return newService(conn, id, token), nil
	}
}

func DialService(addr string, id uint32, token string) (*Service, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return NewService(conn, id, token)
}

func (this *Service) accept(closef func(err error)) {
	for {
		muxConn, err := this.muxSession.Accept()
		if err != nil {
			closef(err)
			this.muxSession.Close()
			return
		}
		// 暂不允许对端开启
		muxConn.Close()
	}
}

func (this *Service) handleConn(conn net.Conn) error {
	muxConn, err := this.muxSession.Open()
	if err != nil {
		return err
	}
	id := muxConn.ID()
	ch := newChannel(muxConn, conn)

	this.chanLock.Lock()
	this.channels[id] = ch
	this.chanLock.Unlock()

	ch.run(func(err error) {
		log.Errorf("channel %d closed %v", id, err)
		this.chanLock.Lock()
		delete(this.channels, id)
		this.chanLock.Unlock()
	})
	return nil
}
