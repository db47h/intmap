// +build amd64 amd64p32 arm64 mips64 mips64le ppc64 ppc64le s390x wasm

package intmap

func hash(v int) int {
	// v ^= v << 13
	// v ^= v >> 7
	// v ^= v << 17
	// return v
	v *= -7046029254386353131 // 0x9E3779B97F4A7C15
	return v ^ (v >> 32)
}

func nextPowerOf2(v int) int {
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v |= v >> 32
	return v + 1
}
