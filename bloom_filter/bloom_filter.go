// Copyright 2024 Tao Wang <wangtaoking1@qq.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package bloom_filter

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	redis "github.com/redis/go-redis/v9"
)

type BloomFilter struct {
	client      redis.Cmdable
	shards      int
	bitSize     int64
	hashFuncNum int
}

// NewBloomFilter creates a sharded bloom filter instance.
func NewBloomFilter(client redis.Cmdable, totalElements int, falsePositiveRate float64, shards int) *BloomFilter {
	if shards <= 0 {
		shards = 1 // default to at least one shard
	}

	totalBitSize := calculateBitSize(totalElements, falsePositiveRate)
	bitSize := int64(math.Ceil(float64(totalBitSize) / float64(shards)))
	numHashFuncs := calculateHashFuncCount(bitSize, totalElements/shards)

	return &BloomFilter{
		client:      client,
		shards:      shards,
		bitSize:     bitSize,
		hashFuncNum: numHashFuncs,
	}
}

// calculateBitSize calculates the required bit array size.
func calculateBitSize(numElements int, falsePositiveRate float64) int64 {
	return int64(-float64(numElements) * math.Log(falsePositiveRate) / (math.Ln2 * math.Ln2))
}

// calculateHashFuncCount calculates the optimal number of hash functions.
func calculateHashFuncCount(bitSize int64, numElements int) int {
	if numElements <= 0 {
		return 1
	}
	cnt := int(math.Ln2 * float64(bitSize) / float64(numElements))
	if cnt < 1 {
		cnt = 1
	}
	return cnt
}

// hash computes multiple hash positions using enhanced double hashing and
// returns the shard index and bit offset for each hash function.
// h_i(x) = (h1(x) + i * h2(x)) % (bitSize * shards), covering the full bit space.
// shard = pos / bitSize, offset = pos % bitSize.
func (bf *BloomFilter) hash(data []byte) ([]int, []int64) {
	sum := sha256.Sum256(data)
	h1 := binary.BigEndian.Uint64(sum[0:8])
	h2 := binary.BigEndian.Uint64(sum[8:16])
	if h2 == 0 {
		h2 = 1
	}

	totalBitSize := uint64(bf.bitSize) * uint64(bf.shards)
	shards := make([]int, bf.hashFuncNum)
	positions := make([]int64, bf.hashFuncNum)

	for i := 0; i < bf.hashFuncNum; i++ {
		pos := (h1 + uint64(i)*h2) % totalBitSize
		shards[i] = int(pos / uint64(bf.bitSize))
		positions[i] = int64(pos % uint64(bf.bitSize))
	}

	return shards, positions
}

// Add sets the corresponding bits for each element in the bloom filter.
func (bf *BloomFilter) Add(ctx context.Context, baseKey string, elements [][]byte) error {
	pipe := bf.client.Pipeline()
	for _, element := range elements {
		shards, positions := bf.hash(element)
		for i := 0; i < len(shards); i++ {
			shardKey := fmt.Sprintf("%s:shard%d", baseKey, shards[i])
			// set the corresponding bit in the bitmap
			pipe.SetBit(ctx, shardKey, positions[i], 1)
		}
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

// Exists checks whether each element is present in the bloom filter.
func (bf *BloomFilter) Exists(ctx context.Context, baseKey string, elements [][]byte) ([]bool, error) {
	results := make([]bool, len(elements))

	cmds := make([][]*redis.IntCmd, len(elements))
	pipeline := bf.client.Pipeline()
	for idx, element := range elements {
		shards, positions := bf.hash(element)
		cmds[idx] = make([]*redis.IntCmd, len(shards))

		// check each shard's bitmap bit by bit
		for i := 0; i < len(shards); i++ {
			shardKey := fmt.Sprintf("%s:shard%d", baseKey, shards[i])
			cmds[idx][i] = pipeline.GetBit(ctx, shardKey, positions[i])
		}
	}

	_, err := pipeline.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}
	for idx, elementCmds := range cmds {
		exists := true
		for _, cmd := range elementCmds {
			value, err := cmd.Result()
			if err != nil {
				if errors.Is(err, redis.Nil) {
					exists = false
					break
				}
				return nil, err
			}
			// if any bit is 0, the element is considered absent
			if value == 0 {
				exists = false
				break
			}
		}
		results[idx] = exists
	}

	return results, nil
}
