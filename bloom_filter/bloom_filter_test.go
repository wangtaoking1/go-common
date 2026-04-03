// Copyright 2024 Tao Wang <wangtaoking1@qq.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package bloom_filter

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBloomFilter(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// use 5 shards
	shards := 5
	totalElements := 1000000
	falsePositiveRate := 0.01

	// initialize bloom filter
	bf := NewBloomFilter(client, totalElements, falsePositiveRate, shards)

	key := "key"
	elements := [][]byte{
		[]byte("te1"),
		[]byte("te2"),
		[]byte("te3"),
	}

	// 1. add elements and verify no error
	err := bf.Add(ctx, key, elements)
	require.NoError(t, err, "failed to add elements")

	// 2. check elements exist
	exists, err := bf.Exists(ctx, key, elements)
	require.NoError(t, err, "failed to check elements")

	// 3. assert each element is found
	for i := range elements {
		assert.True(t, exists[i], "element %d not found: %s", i, elements[i])
	}
}
