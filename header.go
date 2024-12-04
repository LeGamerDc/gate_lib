package gate

/* message header wire protocol
/	if message data less than 4KB

/	|-m=0-|-z-|-c-|-e-|------12------|
/	m = more header,	z = compressed
/	c = compound,		e = encrypt

/ 	if message data ge(great or equal) than 4KB

/ 	|-m=1-|-z-|-c-|-e-|------12------|--------16--------|
*/

/* message wire protocol
|-h-|-------- message --------|
    |-----decrypt message-----|							if e = 1
	|------------ uncompressed message -------------|	if z = 1
	|-h-|--message--|-h-|--message--|-h-|--message--|	if c = 1

we promise if message are compound, then sub messages
are always neither compressed nor encrypt nor compound (z=0 c=0 e=0)
*/

//const (
//	maskM byte = 1 << 7
//	maskZ byte = 1 << 6
//	maskC byte = 1 << 5
//	maskE byte = 1 << 4
//
//	maskS          = uint16(4095)
//	offset         = 1 << 16
//	moreHeaderSize = 4096
//)
//
//func parseHeader(b [2]byte) (size int, m, z, c, e bool) {
//	m, z, c, e = b[0]&maskM > 0, b[0]&maskZ > 0, b[0]&maskC > 0, b[0]&maskE > 0
//	size = int(binary.BigEndian.Uint16(b[:]) & maskS)
//	return
//}
//
//func parseHeader2(b [2]byte, size int) int {
//	return int(binary.BigEndian.Uint16(b[:2])) + size*offset
//}
//
//func encodeHeader(b []byte, size int) (n int) {
//	if size < moreHeaderSize {
//		binary.BigEndian.PutUint16(b[:2], uint16(size))
//		return 2
//	}
//	binary.BigEndian.PutUint32(b[:4], uint32(size))
//	b[0] |= maskM
//	return 4
//}
