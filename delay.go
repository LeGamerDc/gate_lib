package gate

import (
	"container/list"
	"sync/atomic"
	"time"
)

type Callable interface {
	Call()
}

type Item[C Callable] struct {
	ts int64
	c  C
}

type task struct {
	ts    int64
	index int64
}

type delay[C Callable] struct {
	data    []Item[C]
	f, r, n int64

	future int64
	ch     chan task
}

func newDelay[C Callable](n int) *delay[C] {
	return &delay[C]{
		data: make([]Item[C], n),
		f:    -1, r: 0, n: int64(n),
		ch: make(chan task, 128),
	}
}

func (d *delay[C]) Push(c C, ts int64) {
	f := atomic.AddInt64(&d.f, 1)
	d.data[f%d.n] = Item[C]{
		ts: ts, c: c,
	}
	if fu := atomic.LoadInt64(&d.future); ts > fu && atomic.CompareAndSwapInt64(&d.future, fu, ts) {
		d.ch <- task{ts: ts, index: f}
	}
}

func (d *delay[C]) Start() {
	var (
		haveTimer = true
		t         = time.NewTimer(0)
		q         = list.New()
		zero      Item[C]
	)
	for {
		select {
		case job := <-d.ch:
			q.PushBack(&job)
			if !haveTimer {
				haveTimer = true
				t.Reset(time.UnixMilli(job.ts).Sub(time.Now()))
			}
		case now := <-t.C:
			if idx, ok := index(q, now.UnixMilli()); ok && d.r <= idx {
				for i := d.r; i <= idx; i++ {
					d.data[i%d.n].c.Call()
					d.data[i%d.n] = zero
				}
				d.r = idx + 1
			}
			if wait, ok := next(q, time.Now()); ok {
				t.Reset(wait)
			} else {
				haveTimer = false
			}
		}
	}
}

func index(q *list.List, now int64) (idx int64, ok bool) {
	for q.Len() > 0 {
		e := q.Front()
		job := e.Value.(*task)
		if now < job.ts {
			return
		}
		idx, ok = job.index, true
		q.Remove(e)
	}
	return
}

func next(q *list.List, now time.Time) (d time.Duration, ok bool) {
	e := q.Front()
	if e != nil {
		job := e.Value.(*task)
		return time.UnixMilli(job.ts).Sub(now), true
	}
	return
}
