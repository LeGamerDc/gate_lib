package main

import (
	"fmt"
	"time"
)

var ss *Stat

type Stat struct {
	ch     chan one
	count  int
	cost   time.Duration
	o, z   int
	nn, nc int
}

type one struct {
	cost    time.Duration
	n, o, z int
}

func (s *Stat) Print() {
	if s.count > 0 {
		avg := time.Duration(float64(s.cost) / float64(s.count))
		ratio := 0.0
		if s.o > 0 {
			ratio = float64(s.z) / float64(s.o)
		}
		fmt.Printf("count: %d, avg: %v, byte: %s, z: %s, ratio: %.2f, compound %.2f \n", s.count, avg,
			format(s.o), format(s.z), ratio, float64(s.nn)/float64(s.nc))
	} else {
		fmt.Println("???")
	}
	s.count = 0
	s.cost = 0
	s.o = 0
	s.z = 0
	s.nn = 0
	s.nc = 0
}

func (s *Stat) run() {
	t := time.NewTicker(time.Second)
	for {
		select {
		case <-t.C:
			s.Print()
		case c := <-s.ch:
			s.nn += c.n
			s.nc++

			s.count += c.n
			s.cost += c.cost
			s.o += c.o
			s.z += c.z
		}
	}
}

func send(c time.Duration, n, o, z int) {
	ss.ch <- one{c, n, o, z}
}

func init() {
	ss = new(Stat)
	ss.ch = make(chan one, 5000)
	go ss.run()
}

func format(x int) string {
	switch {
	case x > 1024*1024*1024:
		return fmt.Sprintf("%.2fG", float64(x)/1024/1024/1024)
	case x > 1024*1024:
		return fmt.Sprintf("%.2fM", float64(x)/1024/1024)
	case x > 1024:
		return fmt.Sprintf("%.2fK", float64(x)/1024)
	default:
		return fmt.Sprintf("%.2fB", float64(x))
	}
}
