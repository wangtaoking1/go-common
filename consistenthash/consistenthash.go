// Copyright 2024 Tao Wang <wangtaoking1@qq.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package consistenthash

import (
	"fmt"
	"hash/crc32"
	"slices"
	"sort"
	"sync"
)

type HashFunc func(data []byte) uint32

// Hash defines the interface of consistent ring hash.
type Hash interface {
	Reset(nodes ...string)
	HashKey(key string) string
}

// New returns a consistent ring hash with the virtual node replicas and hash function.
func New(replicas int, fn HashFunc) Hash {
	m := &hash{
		hashFunc: fn,
		replicas: replicas,
		nodes:    make(map[uint32]string),
	}
	if m.hashFunc == nil {
		m.hashFunc = crc32.ChecksumIEEE
	}

	return m
}

type hash struct {
	hashFunc HashFunc
	replicas int
	ring     []uint32
	nodes    map[uint32]string

	mtx sync.RWMutex
}

func (h *hash) addNodes(nodes ...string) {
	for _, node := range nodes {
		for i := 0; i < h.replicas; i++ {
			hashVal := h.hashFunc([]byte(fmt.Sprintf("%d%s", i, node)))
			h.ring = append(h.ring, hashVal)
			h.nodes[hashVal] = node
		}
	}
	slices.Sort(h.ring)
}

func (h *hash) Reset(nodes ...string) {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	h.ring = nil
	h.nodes = make(map[uint32]string)
	h.addNodes(nodes...)
}

func (h *hash) HashKey(key string) string {
	h.mtx.RLock()
	defer h.mtx.RUnlock()

	if len(h.ring) == 0 {
		return ""
	}

	hashVal := h.hashFunc([]byte(key))
	idx := sort.Search(len(h.ring), func(i int) bool { return h.ring[i] >= hashVal })
	// Slots is a circle, goto the first one when the index is out of bounds.
	if idx == len(h.ring) {
		idx = 0
	}

	return h.nodes[h.ring[idx]]
}

var _ Hash = (*hash)(nil)
