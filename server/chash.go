package server

import (
	"sort"
)

// Constants
const DEFAULT_HASHRING_CAPACITY = 4

// Consistent Hashing.
type Bucket interface {
	Hash() uint32
	ResetHash()
	Rehash()
}

type Hashable interface {
	Hash() uint32
}

type HashRing struct {
	ring []Bucket // (Ascending order)
}

func NewEmptyHashRing() *HashRing {
	instance := &HashRing{
		ring: make([]Bucket, 0, DEFAULT_HASHRING_CAPACITY),
	}
	return instance
}

func NewHashRing(buckets []Bucket) *HashRing {
	instance := &HashRing{
		ring: make([]Bucket, 0, len(buckets)),
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
	if index >= len(r.ring) {
		return nil
	}

	return r.ring[index]
}

func (r *HashRing) Append(bucket Bucket) (int, Bucket) {
	// avoid collision.
	hash := bucket.Hash()
	for {
		idx, _ := r.FromHash(hash)
		if idx != -1 {
			bucket.Rehash()
			hash = bucket.Hash()
		}
		break
	}

	oriLen := len(r.ring)
	idx := oriLen
	for ; idx > 0 && bucket.Hash() < r.ring[idx-1].Hash(); idx-- {
		r.ring[idx] = r.ring[idx-1]
	}
	r.ring[idx] = bucket

	// Find the following bucket
	if idx == oriLen {
		idx = 0
	} else {
		idx += 1
	}
	return idx, r.ring[idx]
}

// Search bucket according to hash value.
func (r *HashRing) Search(hash uint32) int {
	return sort.Search(len(r.ring), func(idx int) bool {
		return hash <= r.ring[idx].Hash()
	})
}

// Hit bucket.
func (r *HashRing) HashHit(hash uint32) (int, Bucket) {
	if len(r.ring) == 0 {
		return -1, nil
	}

	idx := r.Search(hash)

	if idx == len(r.ring) {
		idx = 0
	}
	return idx, r.ring[idx]
}

func (r *HashRing) Hit(instance Hashable) (int, Bucket) {
	return r.HashHit(instance.Hash())
}

// Find bucket whose hash is equal to the given.
func (r *HashRing) FromHash(hash uint32) (int, Bucket) {
	idx, bucket := r.HashHit(hash)
	if bucket.Hash() != hash {
		return -1, nil
	}
	return idx, bucket
}

func (r *HashRing) uniquify(buckets []Bucket) bool {
	hashes := make(map[uint32]Bucket, len(buckets))
	rehash := false

	// Reverse iteration to keep the result same as appending bucket with Append() sequentially.
	for i := len(buckets); i > 0; {
		hash := buckets[i-1].Hash()
		if _, exists := hashes[hash]; exists {
			buckets[i-1].Rehash()
			rehash = true
			continue
		}
		hashes[hash] = buckets[i-1]
		i--
	}
	return rehash
}

func (r *HashRing) Substitute(buckets []Bucket) {
	r.ring = r.ring[0:0] // Clear
	// Copy
	for idx, bucket := range buckets {
		bucket.ResetHash()
		r.ring[idx] = bucket
	}
	r.uniquify(r.ring)
	sort.Stable(r)
}

func (r *HashRing) Remove(index int) (Bucket, Bucket) {
	if index >= len(r.ring) || index < 0 {
		return nil, nil
	}
	removed := r.ring[index]
	for i := index + 1; i < len(r.ring); i++ {
		r.ring[i-1] = r.ring[i]
	}
	// shrink.
	r.ring = r.ring[:len(r.ring)-1]
	return removed, r.ring[index]
}

func (r *HashRing) RemoveHash(hash uint32) (Bucket, Bucket) {
	idx, _ := r.FromHash(hash)
	return r.Remove(idx)
}
