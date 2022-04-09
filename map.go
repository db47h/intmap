// Copyright 2019 Denis Bernard <db047h@gmail.com>
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

/*
Package intmap implements a fast integer keyed map. Map data is kept densely
packed in order to improve data locality.

Set and Get operations are consistently faster than the builtin
map[int]interface{}: Get takes from 25 to 50% less time, depending on CPU
architecture and map size. On some CPUs like an AMD FX, there is even an actual
drop off point in map size (~16384) where the builtin map gets slightly faster.

Benchmark sample (*Builtin are performed using a regular map[int]interface{}):

    BenchmarkIntMapSet-6            28105802                37.53 ns/op
    BenchmarkBuiltinMapSet-6        21551445                53.44 ns/op
    BenchmarkIntMapGet-6            35170137                33.10 ns/op
    BenchmarkBuiltinMapGet-6        26973751                44.79 ns/op
    BenchmarkIntMapDelete-6         33600421                32.16 ns/op
    BenchmarkBuiltinMapDelete-6     38036679                28.68 ns/op

The delete test is wrong since we end up deleting millions of non-existent keys,
which is not a typical use case. Regardless, deletes are slower than with the
builtin map.


Internals

The implementation is based on
http://java-performance.info/implementing-world-fastest-java-int-to-int-hash-map/.

The stored values can be of any type.

*/
package intmap

// KeyValue wraps a key-value pair.
//
type KeyValue[V any] struct {
	Key   int
	Value V
}

const (
	freeKey          = 0
	defaultFillRatio = 0.875
)

// Map is a fast int to interface{} map. Map data is kept densely packed in
// order to improve data locality.
//
// The primary use case for this implementation is that of small maps
// (regardless of the size of the key set) with almost no deletions.
//
// A Map can be used directly: the start capacity will be set to 8 entries and
// the fill ratio 87.5%. If the rough map size is known in advance, it is
// however preferable to initialize it with New or Init for better performance,
// especially when initializing a large number of maps.
//
// When the size of a Map grows over the fill ratio, its capacity is doubled.
// Maps are never shrunk when deleting keys.
//
type Map[V any] struct {
	es           []KeyValue[V]
	size         int
	threshold    int
	hasFreeKey   bool
	freeKeyValue V
}

func nextIdx(idx int) int {
	return idx + 1
}

// New returns a new Map initialized with the given starting capacity and fill
// ratio.
//
// See Map.Init for more details about the capacity and fillratio parameters.
//
func New[V any](capacity int, fillratio float32) *Map[V] {
	var m Map[V]
	m.Init(capacity, fillratio)
	return &m
}

// Init initializes the Map with the given initial capacity and fill ratio.
//
// If the Map already contains data, it will be reset.
//
// The capacity is rounded up to the next exponent of two. Values < 2 are
// rounded up to 2. It must be less than or equal to 0x40000000 for 32 bits ints
// and 0x4000000000000000 for 64 bits ints.
//
// The fill ratio should be between 0 (0%) and 1 (100%) exclusive. Values out of
// this range are silently rounded to the lowest or largest possible value. When
// the size of a Map grows over the fill ratio, its capacity is doubled.
//
// The fill ratio will be rounded as follows:
//
//  threshold := int(float32(capacity) * fillratio)
//  if threshold <= 0 {
//      threshold = 1
//  } else if threshold >= capacity {
//      threshold = capacity - 1
//  }
//  fillratio_rounded := float32(threshold) / float32(capacity)
//
// i.e. requesting a Map of initial capacity 2 with any fill ratio > 0.5 will
// result in a real fill ratio of 0.5 due to integer rounding.
//
func (m *Map[V]) Init(capacity int, fillratio float32) {
	capacity = nextPowerOf2(capacity)
	if capacity < 0 {
		panic("invalid capacity requested")
	}
	if capacity < 2 {
		capacity = 2
	}
	threshold := int(float32(capacity) * fillratio)
	if threshold <= 0 {
		threshold = 1
	} else if threshold >= capacity {
		threshold = capacity - 1
	}
	m.es = make([]KeyValue[V], capacity)
	m.size = 0
	m.threshold = threshold
	m.hasFreeKey = false
}

// Set sets or resets the value for the given key.
//
func (m *Map[V]) Set(key int, value V) {
	if key == freeKey {
		m.hasFreeKey = true
		m.freeKeyValue = value
		return
	}
	l := len(m.es)
	if m.size >= m.threshold {
		// over fillratio, rehash
		if l == 0 {
			l = 8
			m.es = make([]KeyValue[V], l)
			m.threshold = int(defaultFillRatio * float32(l)) // use a default fillratio of 87.5%
		} else {
			l *= 2
			m.rehash()
		}
	}

	mod := l - 1
	idx := hash(key) & mod
	for {
		switch m.es[idx].Key {
		case freeKey:
			m.size++
			fallthrough
		case key:
			m.es[idx] = KeyValue[V]{key, value}
			return
		}
		idx = nextIdx(idx) & mod
	}
}

func (m *Map[V]) rehash() {
	es := m.es
	l := len(es) << 1
	if l < 0 {
		panic("map size overflows addressable space")
	}
	m.es = make([]KeyValue[V], l)
	m.size = 0
	m.threshold <<= 1
	for i := range es {
		if es[i].Key != freeKey {
			m.Set(es[i].Key, es[i].Value)
		}
	}
}

// Get returns the value associated with the given key and ok set to true if the key exists.
// If the keys does not exist, it returns the zero value for the Value type and false.
//
func (m *Map[V]) Get(key int) (v V, ok bool) {
	if key == freeKey {
		if m.hasFreeKey {
			return m.freeKeyValue, true
		}
		return v, false
	}
	mod := len(m.es) - 1
	if mod < 0 {
		return v, false
	}
	startIdx := hash(key) & mod
	idx := startIdx
	for {
		t := &m.es[idx]
		switch t.Key {
		case freeKey:
			return v, false
		case key:
			return t.Value, true
		}
		idx = nextIdx(idx) & mod
		if idx == startIdx {
			return v, false
		}
	}
}

// Delete deletes the given key and returns true if the key was present in the map.
//
func (m *Map[V]) Delete(key int) bool {
	if key == freeKey {
		var zv V
		rv := m.hasFreeKey
		m.freeKeyValue = zv
		m.hasFreeKey = false
		return rv
	}
	mod := len(m.es) - 1
	if mod < 0 {
		return false
	}
	startIdx := hash(key) & mod
	idx := startIdx
	for {
		switch m.es[idx].Key {
		case freeKey:
			return false
		case key:
			m.shiftKeys(idx)
			m.size--
			return true
		}
		idx = nextIdx(idx) & mod
		if idx == startIdx {
			return false
		}
	}
}

func (m *Map[V]) shiftKeys(idx int) {
	var k int
	mod := len(m.es) - 1
	for {
		last := idx
		idx = nextIdx(idx) & mod
		for {
			k = m.es[idx].Key
			if k == freeKey {
				m.es[last] = KeyValue[V]{Key: freeKey}
				return
			}
			slot := hash(k) & mod
			if last <= idx {
				if last >= slot || slot > idx {
					break
				}
			} else if last >= slot && slot > idx {
				break
			}
			idx = nextIdx(idx) & mod
		}
		m.es[last] = KeyValue[V]{k, m.es[idx].Value}
	}
}

// Len returns the number if keys set in the map.
//
func (m *Map[V]) Len() int {
	if m.hasFreeKey {
		return m.size + 1
	}
	return m.size
}

// Keys returns an unordered slice of the map keys.
//
func (m *Map[V]) Keys() []int {
	ks := make([]int, m.Len())
	i := 0
	if m.hasFreeKey {
		ks[i] = freeKey
		i++
	}
	es := m.es
	for e := range es {
		if k := es[e].Key; k != freeKey {
			ks[i] = k
			i++
		}
	}
	return ks
}

// Iterator returns an iterator over the map's key/value pairs.
//
//	for i := m.Iterator(); i.HasNext(); {
//		k, v := i.Next()
//		fmt.Printf("m[%v] = %v\n", k, v)
//	}
//
// While iterating over a map, deleting the value of the last key returned by
// Next is supported as well of changing the value of any existing key.
// Inserting new keys or deleting any other keys will break the iterator.
//
func (m *Map[V]) Iterator() *Iterator[V] {
	// find a sensible default for
	return &Iterator[V]{m: m, lastKey: freeKey ^ -1, i: -1}
}

// Iterator represents an iterator over a map.
//
type Iterator[V any] struct {
	m       *Map[V]
	lastKey int
	i       int
}

// HasNext returns true if there are any keys left to read.
//
func (i *Iterator[V]) HasNext() bool {
	es := i.m.es
	l := len(es)
	if i.i < 0 {
		// first call
		if i.m.hasFreeKey && i.lastKey != freeKey {
			return true
		}
		i.lastKey = freeKey
	} else {
		// check for deletion of last key read by next
		if k := i.m.es[i.i].Key; k != freeKey && k != i.lastKey {
			return true
		}
	}
	for e := i.i + 1; e < l; e++ {
		if k := es[e].Key; k != freeKey {
			i.i = e
			return true
		}
	}
	i.i = l
	return false
}

// Next returns the next key/value pair. Calling Next several times in a row
// without calling HasNext in between will yield the same result.
//
func (i *Iterator[V]) Next() (key int, value V) {
	if i.i < 0 {
		if !i.m.hasFreeKey {
			panic("Next() called without calling HasNext() first")
		}
		i.lastKey = freeKey
		return freeKey, i.m.freeKeyValue
	}
	i.lastKey = i.m.es[i.i].Key
	return i.lastKey, i.m.es[i.i].Value
}
