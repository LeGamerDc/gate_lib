package main

import (
	"bytes"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/LeGamerDc/gate"

	_ "net/http/pprof"
)

var (
	port int
	loop int
)

func main() {
	fmt.Println("pid", os.Getpid())
	flag.IntVar(&port, "p", 8081, "port")
	flag.IntVar(&loop, "l", 1, "loop")
	flag.Parse()
	go func() {
		http.ListenAndServe("0.0.0.0:8520", nil)
	}()
	e := gate.StartEventLoop(&gate.Config{
		LoopCount: loop,
		Port:      port,
		CHB:       &hb{},
		SB: gate.NewSenderBuilder(&gate.SenderConfig{
			CompressThreshold: 100,
			MaxBufferSize:     64 * 1024 * 1024,
			MaxClusterSize:    32 * 1024,
			DelaySendMs:       5,
		}),
		Logger: &logger{},
	})
	fmt.Println(e)
}

type logger struct{}

func (l *logger) Debugf(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

func (l *logger) Infof(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

func (l *logger) Warnf(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

func (l *logger) Errorf(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

func (l *logger) Fatalf(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

type hb struct{}

func (hb *hb) Build(conn *gate.Conn) gate.ConnHandler {
	return &h{conn: conn}
}

type h struct {
	conn  *gate.Conn
	login bool
}

func (h *h) Handle(raw []byte) {
	ack := bytes.Clone(raw)
	_ = h.conn.Send(ack)
	//if !h.login {
	//	h.login = true
	//	ack := bytes.Clone(raw)
	//	h.conn.AsyncDo(func() {
	//		time.Sleep(rand.N[time.Duration](time.Second * 3))
	//		_ = h.conn.Send(ack)
	//	})
	//} else {
	//	_ = h.conn.Send(raw)
	//}
}

func (h *h) Close() {}
