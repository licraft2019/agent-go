package agent

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/magicsea/behavior3go/core"
)

type One interface {
	ID() string
}

type Market struct {
	idx    int64
	hub    chan One
	acc    chan One
	amount int
	used   map[string]One
	sync.Mutex
}

func newMarket(amount int) *Market {
	amount = amount * 2
	return &Market{
		amount: amount,
		hub:    make(chan One, amount),
		acc:    make(chan One, amount),
		used:   make(map[string]One),
	}
}

func (h *Market) PushOne(one One) {
	h.hub <- one
}

func (h *Market) RequireOne(like func(One) bool) One {
	var i int
	for {
		one := <-h.hub
		h.hub <- one
		if like == nil {
			return one
		}
		if like(one) {
			return one
		}
		i++
		if i >= h.amount {
			return nil
		}
	}
}

func (h *Market) JoinOne(one One) {
	h.hub <- one
}

func (h *Market) UseAcc(one One) {
	h.Lock()
	h.used[one.ID()] = one
	h.Unlock()
}

func (h *Market) InviteLikeOne(like func(One) bool) One {
	for i := 0; i < h.amount; i++ {
		select {
		case one := <-h.hub:
			if like == nil {
				return one
			}
			if like(one) {
				return one
			} else {
				h.hub <- one
			}
		case <-time.After(time.Millisecond * 10):
			return nil
		}
	}
	return nil
}

func(h *Market) InviteAcc() One {
	select {
	case one := <- h.acc:
		h.UseAcc(one)
		return one
	case <-time.After(time.Millisecond * 10):
		return nil
	}
}

func(h *Market) InviteOne() One {
	return h.InviteLikeOne(nil)
}

func (h *Market) reset() {
	h.idx = 0
	for _, o := range h.used {
		h.acc <- o
	}
	h.hub = make(chan One, h.amount)
	h.used = map[string]One{}
}

func (h *Market) Index() int {
	return int(atomic.AddInt64(&h.idx, 1))
}

type Ticker interface {
	core.Ticker
	Marget() *Market
	context() context.Context
	stat() *actor.PID
	actorCtx() *actor.RootContext
	RecvTime(unixNano string)
	SendTime(unixNano string)
}

type Tick struct {
	core.Tick
	market           *Market
	ctx              context.Context
	actorRootContext *actor.RootContext
	statPID          *actor.PID
	recvTime         int64
	sendTime         int64
}

func NewTick() *Tick {
	tick := &Tick{}
	tick.Initialize()
	return tick
}

// 用于并行的情况，分裂解决并发问题，一个行为树协程上下文使用一个tick
func (t *Tick) Tear(ticker core.Ticker) {
	tick := ticker.(*Tick)
	tick.market = t.market
	tick.ctx = t.ctx
	tick.statPID = t.statPID
	tick.actorRootContext = t.actorRootContext
	t.Tick.Tear(&tick.Tick)
}

func (t *Tick) TearTick() core.Ticker {
	tick := NewTick()
	t.Tear(tick)
	return tick
}

func (t *Tick) Marget() *Market {
	return t.market
}

func (t *Tick) context() context.Context {
	return t.ctx
}

func (t *Tick) stat() *actor.PID {
	return t.statPID
}

func (t *Tick) actorCtx() *actor.RootContext {
	return t.actorRootContext
}

func (t *Tick) RecvTime(unixNano string) {
	t.recvTime = strToInt64(unixNano)
}

func (t *Tick) SendTime(unixNano string) {
	t.sendTime = strToInt64(unixNano)
}

func strToInt64(s string) int64 {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return v
}
