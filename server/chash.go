package server

import (
	"fmt"
	"sort"
	"strings"
)

// Constants
const DEFAULT_HASHRING_CAPACITY = 4

// Consistent Hashing.
type Bucket interface {
	Hash() uint32
	ResetHash()
	Rehash()
	OrderLess(Bucket) bool
}

type BucketSlice []Bucket

func (bs BucketSlice) Less(i, j int) bool {
	return bs[i].OrderLess(bs[j])
}

func (bs BucketSlice) Len() int {
	return len(bs)
}

func (bs BucketSlice) Swap(i, j int) {
	bs[i], bs[j] = bs[j], bs[i]
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

func (r *HashRing) String() string {
	hashValueString := make([]string, 0, len(r.ring))
	for _, bucket := range r.ring {
		hashValueString = append(hashValueString, fmt.Sprintf("%v", bucket.Hash()))
	}
	return "{" + strings.Join(hashValueString, ", ") + "}"
}

func (r *HashRing) Append(bucket Bucket) (int, Bucket) {
	// avoid collision.
	hash := bucket.Hash()
	for {
		idx, _ := r.FromHash(hash)
		if idx != -1 {
			if bucket.OrderLess(r.ring[idx]) {
				bucket, r.ring[idx] = r.ring[idx], bucket
			}
			bucket.Rehash()
			hash = bucket.Hash()
			continue
		}
		break
	}

	oriLen := len(r.ring)
	idx := oriLen
	r.ring = append(r.ring, nil)
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
	if bucket == nil || bucket.Hash() != hash {
		return -1, nil
	}
	return idx, bucket
}

// Ensure hashes not collided and ring sorted.
func (r *HashRing) uniquify(buckets []Bucket) bool {
	sortTimes := 0
	for duplicated, ringLen := true, len(r.ring); duplicated; sortTimes++ {
		duplicated = false
		sort.Stable(r)
		for sb := 1; sb < ringLen; sb++ {
			se, hash := sb, r.ring[sb-1].Hash()
			for ; se < ringLen && r.ring[se].Hash() == hash; se++ {
			}
			if se > sb { // hash values duplicated. se - (sb - 1) > 1
				duplicated = true
				sort.Stable(BucketSlice(r.ring[sb-1 : se]))
				// Re-hash to avoid collision.
				for sb < se {
					r.ring[sb].Rehash()
				}
			}
		}
	}
	return sortTimes > 1
}

func (r *HashRing) Substitute(buckets []Bucket) {
	r.ring = r.ring[0:0] // Clear
	// Copy
	r.ring = append(r.ring, buckets...)
	for _, bucket := range r.ring {
		bucket.ResetHash()
	}
	r.uniquify(r.ring)
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
	if len(r.ring) == 0 {
		return removed, nil
	}
	if index >= len(r.ring) {
		index = 0
	}
	return removed, r.ring[index]
}

func (r *HashRing) RemoveHash(hash uint32) (Bucket, Bucket) {
	idx, _ := r.FromHash(hash)
	return r.Remove(idx)
}
