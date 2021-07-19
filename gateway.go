package gate

import (
	"encoding/binary"
	"github.com/yddeng/utils/log"
	"io"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Gateway struct {
	internalListener net.Listener
	externalListener net.Listener

	token string

	counter     uint32
	services    []*Service
	serviceLock sync.RWMutex
}

func (this *Gateway) hashService(id uint32) *Service {
	this.serviceLock.RLock()
	defer this.serviceLock.RUnlock()
	if id != 0 {
		for _, s := range this.services {
			if s.id == id {
				return s
			}
		}
	}

	if len(this.services) > 0 {
		idx := rand.Uint32() % uint32(len(this.services))
		return this.services[idx]
	}
	return nil
}

func (this *Gateway) setService(id uint32, ser *Service) bool {
	this.serviceLock.Lock()
	defer this.serviceLock.Unlock()
	for _, s := range this.services {
		if s.id == id {
			return false
		}
	}
	this.services = append(this.services, ser)
	return true
}

func (this *Gateway) delService(id uint32) {
	this.serviceLock.Lock()
	defer this.serviceLock.Unlock()
	var idx = -1
	for i, s := range this.services {
		if s.id == id {
			idx = i
			break
		}
	}

	if idx != -1 {
		this.services = append(this.services[:idx], this.services[idx+1:]...)
	}
}

func (this *Gateway) Stop() {
	this.internalListener.Close()
	this.externalListener.Close()
}

func Launch(internalAddr, externalAddr, token string) *Gateway {
	gate := new(Gateway)
	gate.token = token

	var err error
	gate.internalListener, err = net.Listen("tcp", internalAddr)
	if err != nil {
		panic(err)
	}

	gate.externalListener, err = net.Listen("tcp", externalAddr)
	if err != nil {
		panic(err)
	}

	go listen(gate.internalListener, gate.handleService)
	go listen(gate.externalListener, gate.handleClient)

	return gate
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

func (this *Gateway) handleService(conn net.Conn) {
	data := make([]byte, 9)

	conn.SetReadDeadline(time.Now().Add(time.Second * 2))
	defer conn.SetReadDeadline(time.Time{})
	if _, err := io.ReadFull(conn, data); err != nil {
		conn.Close()
		return
	}

	reply := func(code, id uint32) error {
		data := make([]byte, 8)
		binary.BigEndian.PutUint32(data, code)
		binary.BigEndian.PutUint32(data[4:], id)

		conn.SetWriteDeadline(time.Now().Add(time.Second * 2))
		defer conn.SetWriteDeadline(time.Time{})
		_, err := conn.Write(data)
		return err
	}

	id := binary.BigEndian.Uint32(data[1:])
	tk := binary.BigEndian.Uint32(data[5:])

	if tk != HashNumber(this.token) {
		_ = reply(Err_TokenFailed, 0)
		_ = conn.Close()
		return
	}

	if id == 0 {
		id = atomic.AddUint32(&this.counter, 1)
	}
	serv := newService(conn, id, "")
	if ok := this.setService(id, serv); !ok {
		_ = reply(Err_NoService, 0)
		_ = conn.Close()
		return
	}

	if err := reply(OK, id); err != nil {
		_ = conn.Close()
		return
	}
	log.Info("new service", id)

	go serv.accept(func(err error) {
		log.Errorf("service %d closed %s", id, err)
		this.delService(id)
	})
}

func (this *Gateway) handleClient(conn net.Conn) {
	data := make([]byte, 9)

	conn.SetReadDeadline(time.Now().Add(time.Second * 2))
	defer conn.SetReadDeadline(time.Time{})
	if _, err := io.ReadFull(conn, data); err != nil {
		conn.Close()
		return
	}

	reply := func(code, id uint32) error {
		data := make([]byte, 8)
		binary.BigEndian.PutUint32(data, code)
		binary.BigEndian.PutUint32(data[4:], id)

		conn.SetWriteDeadline(time.Now().Add(time.Second * 2))
		defer conn.SetWriteDeadline(time.Time{})
		_, err := conn.Write(data)
		return err
	}

	id := binary.BigEndian.Uint32(data[1:])
	tk := binary.BigEndian.Uint32(data[5:])

	if tk != HashNumber(this.token) {
		_ = reply(Err_TokenFailed, 0)
		_ = conn.Close()
		return
	}

	serv := this.hashService(id)
	if serv == nil {
		_ = reply(Err_NoService, 0)
		_ = conn.Close()
		return
	}

	log.Info("new client", id)
	if err := serv.handleConn(conn); err != nil {
		_ = reply(Err_NoService, 0)
		_ = conn.Close()
		return
	}
	if err := reply(OK, serv.id); err != nil {
		_ = conn.Close()
	}
}
