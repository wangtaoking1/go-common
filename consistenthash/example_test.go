// Copyright 2024 Tao Wang <wangtaoking1@qq.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package consistenthash

import "fmt"

func ExampleHash() {
	h := New(3, nil)
	h.Reset("node1", "node2", "node3")

	fmt.Println(h.HashKey("key1"))
	fmt.Println(h.HashKey("key3"))

	h.Reset("node1", "node2")
	fmt.Println(h.HashKey("key1"))
	fmt.Println(h.HashKey("key3"))

	// Output:
	// node1
	// node3
	// node1
	// node1
}
