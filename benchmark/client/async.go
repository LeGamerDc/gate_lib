package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"slices"
	"time"

	jsoniter "github.com/json-iterator/go"
)

const pad = "aaaaaaaaaaabbbbbbbbbbbb"

type Message struct {
	Id int   `json:"id"`
	Ts int64 `json:"ts"`
	//Pad string `json:"pad,omitempty"`
}

func runAsyncClient() {
	cs := make([]net.Conn, 0, client)
	for i := 0; i < client; i++ {
		c, e := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
		if e != nil {
			fmt.Println("dial err:", e)
			return
		}
		go verify(c)
		cs = append(cs, c)
	}
	var (
		e      error
		n, i   int
		m      Message
		header [4]byte
	)
	//m.Pad = pad
	for {
		m.Id = i
		for _, conn := range cs {
			m.Ts = time.Now().UnixMicro()
			b, _ := jsoniter.Marshal(&m)

			n = encodeHeader(header[:], len(b))
			if n == 4 {
				header[0] |= 1 << 7
			}
			if _, e = conn.Write(header[:n]); e != nil {
				fmt.Println("write err:", e)
				return
			}
			if _, e = conn.Write(b); e != nil {
				fmt.Println("write err:", e)
				return
			}
		}
		//if i < 40 {
		//	time.Sleep(time.Millisecond * 2)
		//}
		i++
	}
}

func verify(c net.Conn) {
	r := bufio.NewReader(c)
	h := new(handler)
	e := h.handle(r, 0)
	fmt.Println(h.i, e)
	_ = c.Close()
}

type handler struct {
	i         int
	cost      time.Duration
	os, count int
}

func (h *handler) handle(r io.Reader, d int) error {
	var (
		e         error
		header    [4]byte
		flags     byte
		size      int
		ack, data []byte
		m         Message
	)
	for {
		_, e = io.ReadFull(r, header[:2])
		if e != nil {
			return e
		}
		flags = header[0]
		size = int(binary.BigEndian.Uint16(header[:2]) & 4095)
		if flags&(1<<7) != 0 {
			if _, e = io.ReadFull(r, header[2:4]); e != nil {
				fmt.Println("read header err:", e)
				return e
			}
			size = int(binary.BigEndian.Uint16(header[2:4])) + size*4096
		}
		ack = slices.Grow(ack, size)
		ack = ack[:size]
		if _, e = io.ReadFull(r, ack); e != nil {
			fmt.Println("read ack err:", e)
			return e
		}
		data = ack
		if flags&(1<<6) != 0 { // compressed
			var x []byte
			if x, e = dec.DecodeAll(ack, x); e != nil {
				fmt.Println("decode ack err:", e)
				return e
			}
			data = x
		}
		if flags&(1<<5) != 0 { // compound
			rr := bytes.NewReader(data)
			e = h.handle(rr, d+1)
			if h.count > 0 {
				send(h.cost, h.count, h.os, size)
				h.cost = 0
				h.count = 0
				h.os = 0
			}
			if d == 0 {
				if e == nil || e == io.EOF {
					continue
				}
				return e
			}
			return e
		} else {
			//	check data valid
			if e = jsoniter.Unmarshal(data, &m); e != nil {
				fmt.Printf("d %d %x %d %d\n", d, header[:2], size, h.i)
				fmt.Println("unmarshal err:", string(data))
				return e
			}
			if m.Id != h.i {
				fmt.Println(d, "id mismatch", m.Id, h.i)
				return e
			}
			if d == 0 {
				send(time.Since(time.UnixMicro(m.Ts)), 1, len(data), size)
			} else {
				h.cost += time.Since(time.UnixMicro(m.Ts))
				h.os += size
				h.count++
			}
			h.i++
		}
	}
}
