package pool

import (
    "sync"
    "container/heap"
    "errors"
)

// Drip is pool item
type Drip struct {
    used int
    index int
    pool *Pool
    x interface{}
}

func (d *Drip) Release(err error) {
    d.pool.lock.Lock()

    defer d.pool.lock.Unlock()

    if !p.Interface.Healthy(x, err) {
        if d.index > 0 {
            // Remove unhealth drip.
            heap.Remove(pool, d.index)
            defer d.pool.Notify(&NotifyContext{
                Event: POOL_REMOVE_DRIP,
                Related: d,
                Used: d.used
                Index: -1, 
                DripCount: len(d.pool.drip),
                Error: nil,
            })
        }
    }

    d.used --
    defer d.pool.Notify(&NotifyContext{
        Event: DRIP_USED_COUNTER_DOWN,
        Related: d,
        Used: d.used,
        Index, d.index,
        DripCount: len(d.pool.drip),
        Error: nil,
    })
    d.pool.wakeSleeper(1)

    if d.index >= 0 {
        heap.Fix(d, d.index)

    } else if d.used <= 0 {
        p.Interface.Destroy(d.x)

        defer d.pool.Notify(&NotifyContext{
            Event: POOL_DESTROY_DRIP,
            Related: d,
            Used: d.used,
            DripCount: len(d.pool.drip),
            Index: d.index,
            Error: nil,
        })

        d.pool, d.x = nil
    }
}

const (
    POOL_NEW_DRIP           = iota
    POOL_ADD_DRIP
    POOL_DESTROY_DRIP
    POOL_REMOVE_DRIP     
    POOL_NEW_DRIP_FAILURE
    DRIP_USED_COUNTER_UP
    DRIP_USED_COUNTER_DOWN
)

var ErrFull = errors.New("Pool is full.")

type DripInterface struct {
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

type NotifyContext struct {
    Event uint
    Related *Drip
    Used uint32
    Index int
    DripCount int
    Error error
}

func (ctx *NotifyContext) Get() *Drip {
    if ctx.Related == nil || ctx.Related.index < 0 {
        return nil
    }

    ctx.Related.used ++
    heap.Fix(p, drip.Related.index)

    p.Interface.Notify(&NotifyContext{
        Event: DRIP_USED_COUNTER_UP,
        Related: ctx.Related,
        Used: ctx.Related.used
        Index: ctx.Related.index,
        DripCount: len(p.drip),
        Error: nil,
    })
}

type Pool struct {
    Interface DripInterface

    maxDrip     uint32
    maxUsed     uint32

    drip []*Drip

    lock sync.Mutex

    wait      uint32
    wake        uint32
    waiter map[uint32]*sync.WaitGroup
    
}

func (p *Pool) Len() int {
    return len(p.drip)
}

func (p *Pool) Swap(i, j int) {
    p.drip[i].index, p.drip[j].index = j, i
    p.drip[i], p.drip[j] = p.drip[j], p.drip[i]
}

func (p *Pool) Less(i, j int) {
    return p.drip[i].used < p.drip[j].used
}

func (p *Pool) Push(x interface{}) {
    idx := len(p.drip)
    p.drip = append(p.drip, &Drip{
        used: 0,
        index: idx,
        x: x,
        pool: p,
    })

    p.Interface.Notify(&NotifyContext{
        Event: POOL_NEW_DRIP,
        Related: p.drip[idx],
        Used: p.drip[idx].used,
        Index: idx + 1,
        DripCount: idx + 1,
        Error: nil,
    })
}

func (p *Pool) Pop() interface{} {
    idx := len(p.drip) - 1
    p.drip[idx].index = -1
    removed := p.drip[idx].x
    p.drip = p.drip[:idx]
    return removed
}

// Create new pool
func (p *Pool) NewPool(ifce DripInterface, maxDrip, maxUsed uint32) *Pool {
    instance := &Pool{
        maxDrip: maxDrip,
        maxUsed: maxUsed,
        drip: make([]*Drip, 0, maxDrip),
        Interface: ifce,
        waiter: make(map[uint32]*sync.WaitGroup),
    }
    heap.Init(instance)
    return instance
}

func (p *Pool) wakeSleeper(sleeper uint32) {
    for ; sleeper > 0 && p.wait > p.wake;  sleeper--, p.wake++ {
        wg, ok := p.waiter[p.wake]
        if ok && wg != nil {
            wg.Done()
        }
        delete(p.waiter, p.wake)
    }
}

// Select the drip accoarding to straregies.
func (p *Pool) balanceSelect() (int, error) {
    if len(p.drip) < 1 || p.drip[0].used >= p.maxUsed {
        return -1, ErrFull
    }
    return 0, nil
}

// Wait for free drip.
func (p *Pool) wait() {
    // Register myself
    waitToken, wg := p.wait, &sync.WaitGroup{}
    p.wait ++
    p.waiter[waitToken] = wg
    wg.Add(1)
    p.lock.Unlock()

    // Then wait
    wg.Wait()
}

func (p *Pool) Get(wait bool) (interface{}, error) {
    var x interface{}
    var drip *Drip
    var idx int
    var err error

    for {
        p.lock.Lock()
        if len(p.drip) < maxDrip {
            // More drip allowed, allocate new drip.
            raw, err := p.Interface.New()
            if err != nil {
                p.Interface.Notify(&NotifyContext{
                    Event: POOL_NEW_DRIP_FAILURE,
                    Related: nil,
                    Used: 0,
                    Index: -1,
                    DripCount: len(p.drip),
                    Error: err,
                })
            } else {
                // New drip allocated, push it.
                heap.Push(p, raw)
            }
        }

        // Try to get drip
        idx, err = p.balanceSelect()
        if idx < 0 {
            if wait {
                p.wait()
                continue
            }
        } else {
            drip = p.drip[idx]
            drip.used++
            heap.Fix(p, drip.index)
            p.Interface.Notify(&NotifyContext{
                Event: DRIP_USED_COUNTER_UP,
                Related: drip,
                Used: drip.used
                Index: drip.index,
                DripCount: len(p.drip),
                Error: nil,
            })
        }
        break
    }

    p.lock.Unlock()
    return x, err
}
