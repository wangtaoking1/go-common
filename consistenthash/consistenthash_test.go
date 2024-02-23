// Copyright 2024 Tao Wang <wangtaoking1@qq.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package consistenthash

import (
	"fmt"
	"strconv"
	"testing"
)

func TestHashing(t *testing.T) {
	// Override the hash function to return easier to reason about values. Assumes
	// the keys can be converted to an integer.
	h := New(3, func(key []byte) uint32 {
		i, err := strconv.Atoi(string(key))
		if err != nil {
			panic(err)
		}
		return uint32(i)
	})

	// Given the above hash function, this will give replicas with "hashes":
	// 2, 4, 6, 12, 14, 16, 22, 24, 26
	h.Reset("2", "4", "6")

	testCases := map[string]string{
		"2":  "2",
		"11": "2",
		"23": "4",
		"27": "2",
	}

	for k, v := range testCases {
		if h.HashKey(k) != v {
			t.Errorf("Asking for %s, should have yielded %s", k, v)
		}
	}

	// Adds 8, 18, 28
	h.Reset("2", "4", "6", "8")

	// 27 should now map to 8.
	testCases["27"] = "8"

	for k, v := range testCases {
		if h.HashKey(k) != v {
			t.Errorf("Asking for %s, should have yielded %s", k, v)
		}
	}
}

func TestConsistency(t *testing.T) {
	hash1 := New(1, nil)
	hash2 := New(1, nil)

	hash1.Reset("Bill", "Bob", "Bonny")
	hash2.Reset("Bob", "Bonny", "Bill")

	if hash1.HashKey("Ben") != hash2.HashKey("Ben") {
		t.Errorf("Fetching 'Ben' from both hashes should be the same")
	}

	hash2.Reset("Bob", "Bonny", "Bill", "Becky", "Ben", "Bobby")

	if hash1.HashKey("Ben") != hash2.HashKey("Ben") ||
		hash1.HashKey("Bob") != hash2.HashKey("Bob") ||
		hash1.HashKey("Bonny") != hash2.HashKey("Bonny") {
		t.Errorf("Direct matches should always return the same entry")
	}
}

func BenchmarkGet8(b *testing.B)   { benchmarkGet(b, 8) }
func BenchmarkGet32(b *testing.B)  { benchmarkGet(b, 32) }
func BenchmarkGet128(b *testing.B) { benchmarkGet(b, 128) }
func BenchmarkGet512(b *testing.B) { benchmarkGet(b, 512) }

func benchmarkGet(b *testing.B, shards int) {
	h := New(50, nil)

	var buckets []string
	for i := 0; i < shards; i++ {
		buckets = append(buckets, fmt.Sprintf("shard-%d", i))
	}
	h.Reset(buckets...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.HashKey(buckets[i&(shards-1)])
	}
}
