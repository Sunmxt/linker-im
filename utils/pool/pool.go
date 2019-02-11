package pool

import (
	"container/heap"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Drip is item of pool.
type Drip struct {
	used  int
	index int
	pool  *Pool
	X     interface{}
}

func (d *Drip) Release(err error) {
	d.pool.lock.Lock()
	defer d.pool.lock.Unlock()

	if !d.pool.Interface.Healthy(d.X, err) {
		if d.index >= 0 {
			// Remove drip when drip is unhealthy or pool is closed.
			heap.Remove(d.pool, d.index)
			d.pool.Interface.Notify(&NotifyContext{
				Event:     POOL_REMOVE_DRIP,
				Related:   d,
				Used:      d.used,
				Index:     -1,
				DripCount: len(d.pool.drip),
				Error:     nil,
			})
		}
	}

	d.used--
	d.pool.Interface.Notify(&NotifyContext{
		Event:     DRIP_USED_COUNTER_DOWN,
		Related:   d,
		Used:      d.used,
		Index:     d.index,
		DripCount: len(d.pool.drip),
		Error:     nil,
	})
	d.pool.wakeMany(1)

	if d.index >= 0 {
		heap.Fix(d.pool, d.index)

	} else if d.used <= 0 {
		d.pool.Interface.Destroy(d.X)

		d.pool.Interface.Notify(&NotifyContext{
			Event:     POOL_DESTROY_DRIP,
			Related:   d,
			Used:      d.used,
			DripCount: len(d.pool.drip),
			Index:     d.index,
			Error:     nil,
		})

		d.pool, d.X = nil, nil
	}
}

const (
	POOL_NEW_DRIP = iota
	POOL_NEW
	POOL_DESTROY_DRIP
	POOL_REMOVE_DRIP
	POOL_NEW_DRIP_FAILURE
	DRIP_USED_COUNTER_UP
	DRIP_USED_COUNTER_DOWN
)

var EventDescription = map[uint]string{
	POOL_NEW:               "Pool created.",
	POOL_NEW_DRIP:          "New drip created.",
	POOL_DESTROY_DRIP:      "Drip destroyed.",
	POOL_REMOVE_DRIP:       "Drip removed.",
	POOL_NEW_DRIP_FAILURE:  "Drip creation failure.",
	DRIP_USED_COUNTER_UP:   "Drip got a new user.",
	DRIP_USED_COUNTER_DOWN: "Drip lost an user.",
}

func GetEventDescription(event uint) string {
	desp, ok := EventDescription[event]
	if !ok {
		return fmt.Sprintf("Unknwon event: %v", event)
	}
	return desp
}

var ErrFull = errors.New("Pool is full.")
var ErrClosed = errors.New("Pool is closed.")
var ErrWaitTimeout = errors.New("Waiting time elapsed.")

type DripInterface interface {
	// Allocate new drip
	New() (interface{}, error)

	// Destroy drip.
	Destroy(interface{})

	// Check whether drip is healthy when an error occurs.
	// Unhealthy drip will be destroyed.
	Healthy(interface{}, error) bool

	// Notify that `used` counter is changed or Pool state is changed.
	Notify(ctx *NotifyContext)
}

// NotifyContext is notification holder.
type NotifyContext struct {
	Event     uint
	Related   *Drip
	Used      int
	Index     int
	DripCount int
	Error     error
}

func (ctx *NotifyContext) Get() *Drip {
	if ctx.Related == nil || ctx.Related.index < 0 {
		return nil
	}

	ctx.Related.used++
	heap.Fix(ctx.Related.pool, ctx.Related.index)

	ctx.Related.pool.Interface.Notify(&NotifyContext{
		Event:     DRIP_USED_COUNTER_UP,
		Related:   ctx.Related,
		Used:      ctx.Related.used,
		Index:     ctx.Related.index,
		DripCount: len(ctx.Related.pool.drip),
		Error:     nil,
	})

	return ctx.Related
}

func (ctx *NotifyContext) String() string {
	var errmsg string
	if ctx.Error != nil {
		errmsg = ctx.Error.Error()
	} else {
		errmsg = "none"
	}

	return fmt.Sprintf("{Event: \"%v\", Drip: %v, Used: %v, Index: %v, DripCount: %v, Error: \"%v\"}", GetEventDescription(ctx.Event), ctx.Related, ctx.Used, ctx.Index, ctx.DripCount, errmsg)
}

type Pool struct {
	Interface DripInterface

	maxDrip int
	maxUsed int

	drip []*Drip

	lock sync.Mutex

	waitc  uint32
	wake   uint32
	waiter map[uint32]*sync.WaitGroup
	closed bool
}

func (p *Pool) Len() int {
	return len(p.drip)
}

func (p *Pool) Swap(i, j int) {
	p.drip[i].index, p.drip[j].index = j, i
	p.drip[i], p.drip[j] = p.drip[j], p.drip[i]
}

func (p *Pool) Less(i, j int) bool {
	return p.drip[i].used < p.drip[j].used
}

func (p *Pool) Push(x interface{}) {
	idx := len(p.drip)
	p.drip = append(p.drip, &Drip{
		used:  0,
		index: idx,
		X:     x,
		pool:  p,
	})

	p.Interface.Notify(&NotifyContext{
		Event:     POOL_NEW_DRIP,
		Related:   p.drip[idx],
		Used:      p.drip[idx].used,
		Index:     idx,
		DripCount: idx + 1,
		Error:     nil,
	})
}

func (p *Pool) Pop() interface{} {
	idx := len(p.drip) - 1
	p.drip[idx].index = -1
	removed := p.drip[idx]
	p.drip = p.drip[:idx]
	return removed
}

// Create new pool
func NewPool(ifce DripInterface, maxDrip, maxUsed int) *Pool {
	instance := &Pool{
		maxDrip:   maxDrip,
		maxUsed:   maxUsed,
		drip:      make([]*Drip, 0, maxDrip),
		Interface: ifce,
		waiter:    make(map[uint32]*sync.WaitGroup),
		closed:    false,
	}
	heap.Init(instance)

	instance.Interface.Notify(&NotifyContext{
		Event:     POOL_NEW,
		Related:   nil,
		Used:      0,
		Index:     -1,
		DripCount: len(instance.drip),
		Error:     nil,
	})

	return instance
}

// Select a drip accoarding to straregies.
func (p *Pool) balanceSelect() (int, error) {
	if len(p.drip) < 1 || (p.maxUsed > 0 && p.drip[0].used >= p.maxUsed) {
		return -1, ErrFull
	}
	return 0, nil
}

// Wait for free drip.
func (p *Pool) wait(timeout uint32) error {
	var err error

	// Register myself
	waitToken, wg := p.waitc, &sync.WaitGroup{}
	waitChan := make(chan struct{})

	p.waitc++
	p.waiter[waitToken] = wg
	wg.Add(1)

	p.lock.Unlock()

	go func() {
		wg.Wait()
		waitChan <- struct{}{}
	}()

	// Then wait
	if timeout > 0 {
		chanTimeout := time.After(time.Duration(timeout) * time.Millisecond)
		select {
		case <-waitChan:
		case <-chanTimeout:
			p.lock.Lock()
			p.wakeTarget(waitToken)
			p.lock.Unlock()
			err = ErrWaitTimeout
		}
	} else {
		<-waitChan
	}

	close(waitChan)

	return err
}

func (p *Pool) wakeMany(sleeper uint32) uint32 {
	for sleeper > 0 && p.waitc > p.wake {
		if p.wakeTarget(p.wake) {
			sleeper--
		}
		p.wake++
	}
	return sleeper
}

func (p *Pool) wakeTarget(waitc uint32) bool {
	wg, ok := p.waiter[waitc]
	if !ok || wg == nil {
		return false
	}

	wg.Done()
	delete(p.waiter, p.wake)

	return true
}

func (p *Pool) newDrip() (interface{}, error) {
	if p.maxDrip != 0 && len(p.drip) >= p.maxDrip {
		return nil, nil
	}

	raw, err := p.Interface.New()
	if err != nil {
		p.Interface.Notify(&NotifyContext{
			Event:     POOL_NEW_DRIP_FAILURE,
			Related:   nil,
			Used:      0,
			Index:     -1,
			DripCount: len(p.drip),
			Error:     err,
		})
	} else {
		// New drip allocated, push it.
		heap.Push(p, raw)
	}

	return raw, err
}

func (p *Pool) Get(toWait bool, timeout uint32) (*Drip, error) {
	var drip *Drip
	var idx int
	var err error

	for {
		p.lock.Lock()

		if p.closed {
			// Pool closed.
			err = ErrClosed
			break
		}

		p.newDrip()

		// select a drip
		idx, err = p.balanceSelect()
		if idx < 0 {
			if toWait {
				// No free drip. wait.

				err = p.wait(timeout)
				if err != nil {
					// Timeout
					return nil, err
				}
				continue
			}
		} else {
			drip = p.drip[idx]
			drip.used++
			heap.Fix(p, drip.index)
			p.Interface.Notify(&NotifyContext{
				Event:     DRIP_USED_COUNTER_UP,
				Related:   drip,
				Used:      drip.used,
				Index:     drip.index,
				DripCount: len(p.drip),
				Error:     nil,
			})
		}
		break
	}

	p.lock.Unlock()
	return drip, err
}

// Close pool and wait until all drips are closed.
func (p *Pool) Close() {
	p.lock.Lock()
	p.closed = true

closeDrip:
	for len(p.drip) > 0 { // If any drip exists.
		for p.drip[0].used == 0 {
			// Destroy free drips.
			raw := heap.Pop(p)
			drip, ok := raw.(*Drip)
			if ok {
				p.Interface.Destroy(drip.X)

				p.Interface.Notify(&NotifyContext{
					Event:     POOL_DESTROY_DRIP,
					Related:   drip,
					Used:      drip.used,
					DripCount: len(p.drip),
					Index:     drip.index,
					Error:     nil,
				})
			}

			if len(p.drip) < 1 {
				break closeDrip
			}
		}

		// wait until new drip released.
		p.wait(0)
		p.lock.Lock()
	}
	p.lock.Unlock()
}

// Reset pool to initial state.
func (p *Pool) Reset() {
	p.Close()

	p.lock.Lock()
	p.closed = false

	p.lock.Unlock()
}
