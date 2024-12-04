package main

import (
	"bufio"
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
		i      int
		m      Message
		header [6]byte
	)
	//m.Pad = pad
	for {
		m.Id = i
		for _, conn := range cs {
			m.Ts = time.Now().UnixMicro()
			b, _ := jsoniter.Marshal(&m)

			encodeHeader(&header, int32(i), len(b))
			//fmt.Printf("%d %d %x\n", i, len(b), header)
			if _, e = conn.Write(header[:]); e != nil {
				fmt.Println("write err:", e)
				return
			}
			if _, e = conn.Write(b); e != nil {
				fmt.Println("write err:", e)
				return
			}
		}
		//time.Sleep(time.Second)
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
		header    [6]byte
		flags     byte
		size      int
		ack, data []byte
		m         Message
	)
	for {
		_, e = io.ReadFull(r, header[:6])
		if e != nil {
			return e
		}
		flags = header[0]
		size = int(binary.BigEndian.Uint32(header[0:4]) & (1<<30 - 1))
		ack = slices.Grow(ack, size)
		ack = ack[:size]
		if _, e = io.ReadFull(r, ack); e != nil {
			fmt.Println("read ack err:", e)
			return e
		}
		data = ack
		if flags&(1<<7) != 0 { // compressed
			var x []byte
			if x, e = dec.DecodeAll(ack, x); e != nil {
				fmt.Println("decode ack err:", e)
				return e
			}
			data = x
		}
		{
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
