package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"runtime/debug"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/klauspost/compress/zstd"
)

const core = 4

var (
	host   string
	port   int
	client int
)

func main() {
	wrap(func() {
		flag.StringVar(&host, "h", "127.0.0.1", "host")
		flag.IntVar(&port, "p", 8081, "port")
		flag.IntVar(&client, "c", 1, "client")
		flag.Parse()

		var wg sync.WaitGroup
		wg.Add(core)
		for i := 0; i < core; i++ {
			go func() {
				runAsyncClient()
				wg.Done()
			}()
		}
		wg.Wait()
	})
}

func wrap(f func()) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err, string(debug.Stack()))
			os.Exit(1)
		}
	}()
	f()
}

func runSyncClient() {
	for i := 0; i < client; i++ {
		go runClient()
	}

	//never stop
	ch := make(chan struct{})
	<-ch
}

func runClient() {
	c, e := net.Dial("tcp", host+":"+strconv.Itoa(port))
	if e != nil {
		fmt.Printf("dial err:%v\n", e)
		return
	}
	var (
		data    []byte
		ack     []byte
		header  [4]byte
		n, size int
		flags   byte
	)
	defer c.Close()
	r := bufio.NewReader(c)
	for {
		data = getByte()
		n = encodeHeader(header[:], len(data))
		if n == 4 {
			header[0] |= 1 << 7
		}
		start := time.Now()
		if _, e = c.Write(header[:n]); e != nil {
			fmt.Printf("write err:%v\n", e)
			return
		}
		if _, e = c.Write(data); e != nil {
			fmt.Printf("write err:%v\n", e)
			return
		}
		if _, e = io.ReadFull(r, header[:2]); e != nil {
			fmt.Printf("read header err:%v\n", e)
			return
		}
		flags = header[0]
		size = int(binary.BigEndian.Uint16(header[0:2]) & 4095)
		if flags&(1<<7) != 0 {
			if _, e = io.ReadFull(r, header[2:4]); e != nil {
				fmt.Printf("read header err:%v\n", e)
				return
			}
			size = int(binary.BigEndian.Uint16(header[2:4])) + size*4096
		}
		ack = ack[:0]
		ack = slices.Grow(ack, size)
		ack = ack[:size]
		if _, e = io.ReadFull(r, ack); e != nil {
			fmt.Printf("read ack err:%v\n", e)
			return
		}
		if flags&(1<<6) != 0 {
			//compressed
			var x []byte
			if x, e = dec.DecodeAll(ack, x); e != nil {
				fmt.Printf("decode ack err:%v\n", e)
				return
			}
			ack = x
		}
		if !check(data, ack, flags) {
			fmt.Println("check err")
			fmt.Println(len(data), len(ack), size, string(data), string(ack))
			return
		}
		send(time.Since(start), 1, len(data), size)
	}
}

func check(i, o []byte, flag byte) bool {
	if flag&(1<<6) == 0 {
		return bytes.Equal(i, o)
	}
	return true
}

func encodeHeader(b []byte, size int) (n int) {
	if size < 4096 {
		binary.BigEndian.PutUint16(b[:2], uint16(size))
		return 2
	}
	binary.BigEndian.PutUint32(b[:4], uint32(size))
	return 4
}

var dec *zstd.Decoder

func init() {
	dec, _ = zstd.NewReader(nil, zstd.WithDecoderConcurrency(1))
}
