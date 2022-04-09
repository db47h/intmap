// Copyright 2019 Denis Bernard <db047h@gmail.com>
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build 386 || arm || mips || mipsle

package intmap

func hash(v int) int {
	// v ^= v << 13
	// v ^= v >> 17
	// v ^= v << 5
	// return v
	v *= -1640531527 // 0x9E3779B9
	return v ^ (v >> 32)
}

func nextPowerOf2(v int) int {
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	return v + 1
}
