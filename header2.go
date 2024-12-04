package gate

import "encoding/binary"

/* 消息头设置 header
| -z:1- | -e:1- | ----size:30---- | ----api:16---- |
z 表示是否压缩，1bit
e 表示是否加密，1bit
api 表示消息类型，大端编码
size 表示消息长度，大端编码

|-header-|----message----|
*/

const (
	maskS uint32 = 1<<30 - 1
	maskZ byte   = 1 << 7
	maskE byte   = 1 << 6

	headerSize = 6
)

func parseHeader(b [6]byte) (api int32, size int, z, e bool) {
	z, e = b[0]&maskZ > 0, b[0]&maskE > 0
	size = int(binary.BigEndian.Uint32(b[0:4]) & maskS)
	api = int32(binary.BigEndian.Uint16(b[4:6]))
	return
}

func encodeHeader(b *[6]byte, api int32, size int) {
	binary.BigEndian.PutUint32(b[0:4], uint32(size))
	binary.BigEndian.PutUint16(b[4:6], uint16(api))
}
