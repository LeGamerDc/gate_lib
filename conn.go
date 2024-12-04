package gate

import (
	"fmt"
	"net"
	"runtime/debug"
	"strconv"
	"sync/atomic"

	"github.com/panjf2000/gnet/v2"
)

const (
	maxMessageSize = 32 * 1024 * 1024
)

type Conn struct {
	conn gnet.Conn

	remoteIp   net.IP
	remotePort int
	sender     SenderI
	cipher     Cipher
	handler    ConnHandler

	blocking atomic.Bool
}

func (c *Conn) onTraffic() {
	var (
		api   int32
		msg   []byte
		clean func()
		ok    = true
	)
	for ok {
		if c.blocking.Load() {
			break
		}
		if api, msg, clean, ok = c.read(); ok {
			func() {
				defer clean()
				c.handler.Handle(api, msg)
			}()
		}
	}
}

func (c *Conn) read() (api int32, data []byte, clean func(), ok bool) {
	var (
		buf  []byte
		e    error
		size int
		en   bool
	)
	if buf, e = c.conn.Peek(6); e != nil {
		return
	}
	//fmt.Printf("%x\n", [6]byte(buf))
	api, size, _, en = parseHeader([6]byte(buf))
	fmt.Printf("%d %d %v %x\n", api, size, en, [6]byte(buf))
	if size > maxMessageSize {
		logErr(c.conn.Close())
		return
	}
	if buf, e = c.conn.Peek(headerSize + size); e != nil {
		return
	}
	data = buf[headerSize : headerSize+size]
	clean = func() {
		c.conn.Discard(headerSize + size)
	}
	if en && c.cipher != nil {
		c.cipher.Decrypt(data)
	}
	return api, data, clean, true
}

// AsyncDo 阻塞connection读新消息，直到f完成。对于一些有限制串行的消息有用
func (c *Conn) AsyncDo(f func()) {
	c.blocking.Store(true)
	go func() {
		defer func() {
			c.blocking.Store(false)
			logErr(c.conn.Wake(nil))
			if r := recover(); r != nil {
				fmt.Printf("[gate] AsyncDo panic: %v\n%s\n", r, debug.Stack())
			}
		}()
		f()
	}()
}

func (c *Conn) Send(api int32, data []byte) error {
	return c.sender.Send(api, data)
}

// SendNoEncrypt 不启用加密
func (c *Conn) SendNoEncrypt(api int32, data []byte) error {
	return c.sender.SendNoEncrypt(api, data)
}

// SendCompressed 不启用压缩，有一些消息会发给多个client，可以提前压缩
// 避免每个connection压缩一次
func (c *Conn) SendCompressed(api int32, data []byte) error {
	return c.sender.SendCompressed(api, data)
}

func (c *Conn) UpdateCipher(cipher Cipher) {
	c.cipher = cipher
}

func (c *Conn) RemoteIp() string {
	return c.remoteIp.String()
}

func (c *Conn) RemotePort() int {
	return c.remotePort
}

func (c *Conn) Remote() string {
	return c.remoteIp.String() + ":" + strconv.Itoa(c.remotePort)
}

func (c *Conn) Close() {
	logErr(c.conn.Close())
}

func logErr(err error) {
	if err != nil {
		log.Errorf("[gate] %v", err)
	}
}
