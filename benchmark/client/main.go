package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"sync"

	"github.com/klauspost/compress/zstd"
)

const core = 1

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

func check(i, o []byte, flag byte) bool {
	if flag&(1<<6) == 0 {
		return bytes.Equal(i, o)
	}
	return true
}

func encodeHeader(b *[6]byte, api int32, size int) {
	binary.BigEndian.PutUint32(b[0:4], uint32(size))
	binary.BigEndian.PutUint16(b[4:6], uint16(api))
}

var dec *zstd.Decoder

func init() {
	dec, _ = zstd.NewReader(nil, zstd.WithDecoderConcurrency(1))
}
