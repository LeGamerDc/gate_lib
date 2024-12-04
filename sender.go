package gate

import (
	"bytes"
	"math"
	"slices"
	"sync"

	"github.com/bytedance/gopkg/lang/mcache"
	"github.com/klauspost/compress/zstd"
	"github.com/panjf2000/gnet/v2"
)

var (
	DefaultSenderBuilder = NewSenderBuilder(&SenderConfig{
		CompressThreshold: 1024,            // 1KB
		MaxBufferSize:     2 * 1024 * 1024, // 2MB
		//MaxClusterSize:    32 * 1024,       // 32KB
		//DelaySendMs: 0, // disabled
	})

	enc *zstd.Encoder
)

type sendMsg struct {
	maskPermit, maskAlready byte
	api                     int32
	data                    []byte
}

type SenderConfig struct {
	CompressThreshold int // 开启压缩阈值，<=0 表示从不压缩，单位 Byte
	MaxBufferSize     int // 最大待发送缓冲区大小，<=0 表示不限制，单位 Byte
	//MaxClusterSize    int   // 最大合并数据大小，<=0 表示不合并，单位 Byte
	//DelaySendMs int64 // 延迟发送，用于更多地合并发送数据，减少系统调用，提高压缩率，<= 0 表示不开启，单位 ms
}

type senderBuilder struct {
	c *SenderConfig
}

func (s *senderBuilder) Build(conn *Conn) SenderI {
	sd := &sender{
		c:    s.c,
		conn: conn,
	}
	//d := newDelay[*sender](100000)
	//sd.d = d
	//go d.Start()
	return sd
}

func NewSenderBuilder(c *SenderConfig) SenderBuilder {
	return &senderBuilder{c: c}
}

type sender struct {
	c    *SenderConfig
	conn *Conn

	mu sync.Mutex
	//d         *delay[*sender]
	queue     []sendMsg
	triggered bool
}

func (s *sender) callback(_ gnet.Conn, e error) (_ error) {
	if e == nil {
		buf := func() (q []sendMsg) {
			s.mu.Lock()
			defer s.mu.Unlock()

			s.triggered = false
			if len(s.queue) == 0 {
				return
			}
			q = s.queue
			s.queue = getQ()
			return
		}()

		if len(buf) == 0 {
			return
		}
		defer putQ(buf)
		s.pushTcp(buf)
	}
	return
}

func (s *sender) pushTcp(buf []sendMsg) {
	//fmt.Printf("send %d\n", buf[0].api)
	s.pushSeparate(buf...)
	//config := s.c
	//if config.MaxClusterSize <= 0 || config.CompressThreshold <= 0 {
	//	// 不开启合并
	//	s.pushSeparate(buf...)
	//	return
	//}
	// 开启合并
	//var (
	//	i, j, l = 0, 0, len(buf)
	//)
	//for i < l {
	//	if config.MaxBufferSize > 0 && conn.OutboundBuffered() >= config.MaxBufferSize {
	//		log.Warnf("client blocking, drop message %s", s.conn.Remote())
	//		return
	//	}
	//	if buf[i].maskPermit&maskC == 0 {
	//		for j = i + 1; j < l; j++ {
	//			if buf[j].maskPermit&maskC != 0 {
	//				break
	//			}
	//		}
	//		s.pushSeparate(buf[i:j]...)
	//	} else {
	//		size := len(buf[i].data)
	//		for j = i + 1; j < l; j++ {
	//			size += len(buf[j].data)
	//			if buf[j].maskPermit&maskC == 0 || size > config.MaxClusterSize {
	//				break
	//			}
	//		}
	//		if j-i == 1 || size < config.CompressThreshold {
	//			s.pushSeparate(buf[i:j]...)
	//		} else {
	//			s.pushCompound(buf[i:j])
	//		}
	//	}
	//	i = j
	//}
}

//func (s *sender) pushCompound(buf []sendMsg) {
//	var (
//		b      = getBuffer()
//		header [4]byte
//		flag   = maskC | maskZ
//	)
//	defer putBuffer(b)
//	for _, msg := range buf {
//		n := encodeHeader(header[:], len(msg.data))
//		b.Write(header[:n])
//		b.Write(msg.data)
//	}
//	// 1. compress
//	data := mcache.Malloc(4, 4+b.Len())
//	data = enc.EncodeAll(b.Bytes(), data)
//	defer mcache.Free(data)
//	// 2. encrypt
//	if cipher := s.conn.cipher; cipher != nil {
//		cipher.Encrypt(data[4:])
//		flag |= maskE
//	}
//	// 3. write header
//	if size := len(data[4:]); size < moreHeaderSize {
//		binary.BigEndian.PutUint16(data[2:4], uint16(size))
//		data = data[2:]
//	} else {
//		binary.BigEndian.PutUint32(data[0:4], uint32(size))
//		flag |= maskM
//	}
//	data[0] |= flag
//	// 4. send
//	_, e := s.conn.conn.Write(data)
//	logErr(e)
//}

func (s *sender) pushSeparate(buf ...sendMsg) {
	var (
		conn               = s.conn.conn
		rest               = math.MaxInt
		flag, maskP, maskA byte
		data               []byte
		vb                 = make([][]byte, 0, min(2*len(buf), 64))
		vc                 [][]byte
	)
	for sub := range slices.Chunk(buf, 32) {
		if s.c.MaxBufferSize > 0 {
			if rest = s.c.MaxBufferSize - conn.OutboundBuffered(); rest <= 0 {
				break
			}
		}
		vb = vb[:0]
		vc = vc[:0]

		for _, msg := range sub {
			if rest <= 0 {
				log.Warnf("client blocking, drop message %s", s.conn.Remote())
				break
			}
			var header [headerSize]byte
			data, maskP, maskA = msg.data, msg.maskPermit, msg.maskAlready
			// 1. compress
			if maskP&maskZ > 0 && s.c.CompressThreshold > 0 && len(data) > s.c.CompressThreshold {
				compressed := mcache.Malloc(0, len(data))
				vc = append(vc, compressed)
				data = enc.EncodeAll(data, compressed)
				flag |= maskZ
			}
			// 2. encrypt
			if cipher := s.conn.cipher; cipher != nil && maskP&maskE > 0 {
				cipher.Encrypt(data)
				flag |= maskE
			}
			encodeHeader(&header, msg.api, len(data))
			header[0] |= flag | maskA
			vb = append(vb, header[:headerSize])
			vb = append(vb, data)
			rest -= headerSize + len(data)
		}
		_, e := conn.Writev(vb)
		logErr(e)
		for _, c := range vc {
			mcache.Free(c)
		}
	}
}

func (s *sender) send(api int32, data []byte, maskP, maskA byte) error {
	if len(data) > maxMessageSize {
		return ErrMaxMessageSize
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.queue = append(s.queue, sendMsg{maskP, maskA, api, data})
	if !s.triggered {
		s.triggered = true
		logErr(s.conn.conn.Wake(s.callback))
		//if s.c.DelaySendMs > 0 {
		//	s.d.Push(s, time.Now().UnixMilli()+s.c.DelaySendMs)
		//} else {
		//	logErr(s.conn.conn.Wake(s.callback))
		//}
	}
	return nil
}

func (s *sender) Call() {
	logErr(s.conn.conn.Wake(s.callback))
}

func (s *sender) Send(api int32, data []byte) error {
	return s.send(api, data, maskZ|maskE, 0)
}

func (s *sender) SendNoEncrypt(api int32, data []byte) error {
	return s.send(api, data, 0, 0)
}

func (s *sender) SendCompressed(api int32, data []byte) error {
	return s.send(api, data, maskE, maskZ)
}

// queuePool 复用 sender buffer queue
var queuePool = sync.Pool{New: func() interface{} {
	return make([]sendMsg, 0, 8)
}}

func getQ() []sendMsg {
	return queuePool.Get().([]sendMsg)
}

func putQ(q []sendMsg) {
	if cap(q) > 1024 {
		return
	}
	clear(q)
	q = q[:0]
	queuePool.Put(q)
}

// byteBufferPool 复用 bytesBuffer
var byteBufferPool = sync.Pool{New: func() interface{} {
	return bytes.NewBuffer(make([]byte, 0, 2*1024))
}}

func getBuffer() *bytes.Buffer {
	b := byteBufferPool.Get().(*bytes.Buffer)
	b.Reset()
	return b
}

func putBuffer(b *bytes.Buffer) {
	byteBufferPool.Put(b)
}

func init() {
	enc, _ = zstd.NewWriter(nil, // wont fail
		zstd.WithEncoderLevel(zstd.SpeedBetterCompression),
		zstd.WithEncoderConcurrency(1))
}
