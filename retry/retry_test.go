// Copyright 2024 Tao Wang <wangtaoking1@qq.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package retry

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetryWithTimeout(t *testing.T) {
	c := 0
	_ = WithTimeout(context.Background(), 10*time.Millisecond, 35*time.Millisecond, func() error {
		c++
		return ErrRetryable
	})
	assert.Equal(t, 3, c)
}
