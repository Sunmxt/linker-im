package server

import (
    "sync"
    "sort"
)
// Constants
const DEFAULT_HASHRING_CAPACITY = 4

// Consistent Hashing.
type Bucket interface {
    Hash()      uint32
}

type HashRing struct {
    lock sync.RWMutex
    ring []Bucket   // (Ascending order)
}

func NewEmptyHashRing() *HashRing {
    instance := &HashRing{
        ring:   make([]Bucket, 0, DEFAULT_HASHRING_CAPACITY),
    }
    return instance
}

func NewHashRing(buckets []Bucket) *HashRing {
    instance := &HashRing{
        ring:   make([]Bucket, 0, len(bucket)),
    }
    instance.Substitute(buckets)
    return instance
}

func (r *HashRing) Len() int {
    return len(r.ring)
}

func (r *HashRing) Less(i, j int) bool {
    return r.ring[i].Hash() > r.ring[j].Hash() 
}

func (r *HashRing) Swap(i, j int) {
    r.ring[i], r.ring[j] = r.ring[j], r.ring[i]
}

func (r *HashRing) At(index int) Bucket {
    r.lock.RLock()
    defer r.lock.RUnlock()
    if index >= len(r.ring) {
        return nil
    }

    return ring[index]
}

func (r *HashRing) Append(bucket Bucket) (bool, int, Bucket) {
    r.lock.Lock()
    defer r.lock.Unlock()

    oriLen := len(r.ring)
    idx := oriLen
    for ; idx > 0 && bucket.Hash() < r.ring[idx - 1] ; idx-- {
        r.ring[idx] = r.ring[idx - 1]
    }
    // Collision detected.
    if r.ring[idx - 1].Hash() == bucket.Hash() {
        return false, idx - 1, r.ring[idx - 1]
    }
    r.ring[idx] = bucket

    // Find bucket to re-hash
    if idx == oriLen {
        idx = 0
    } else {
        idx += 1
    }
    return true, idx, r.ring[idx]
}

// Search bucket according to hash value.
func (r *HashRing) Search(hash uint32) int {
    r.lock.RLock()
    defer r.lock.RUnlock()

    return sort.Search(len(r.ring), func (idx int) bool {
        return hash <= r.ring[i]
    }}
}

// Hit bucket.
func (r *HashRing) Hit(hash uint32) (int, Bucket) {
    r.lock.RLock()
    defer r.lock.RUnlock()

    if len(r.ring) == 0 {
        return -1
    }

    idx := r.Search(hash)
    
    if idx == n {
        idx = 0
    }
    return idx, r.ring[idx]
}

// Find bucket whose hash is equal to the given.
func (r *HashRing) FromHash(hash uint32) (int, Bucket) {
    r.lock.RLock()
    defer r.lock.RUnlock()

    idx, bucket := r.Hit(hash)
    if bucket.Hash() != hash {
        return -1, nil
    }
    return idx, bucket
}

func (r *HashRing) Substitute(buckets []Bucket) {
    r.lock.Lock()
    defer r.lock.Unlock()

    r.ring = r.ring[0:0] // Clear
    // Copy
    for idx, bucket := range buckets {
        r.ring[idx] = bucket
    }

    sort.Sort(r)
}

func (r *HashRing) Remove(index int) (Bucket, Bucket) {
    r.lock.Lock()
    defer r.lock.Unlock()

    if index >= len(r.ring) || index < 0 {
        return nil, nil
    }
    removed := r.ring[idx]
    for i := index + 1; i < len(r.ring) ; i++ {
        r.ring[i - 1] = r.ring[i]
    }
    // shrink.
    r.ring = r.ring[:len(r.ring) - 1]
    return removed, r.ring[index]
}

func (r *HashRing) RemoveHash(hash uint32) (Bucket, Bucket) {
    return r.Remove(r.FromHash(hash))
}
