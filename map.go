// Copyright 2019 Denis Bernard <db047h@gmail.com>
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

/*
Package intmap implements a fast int to interface{} map. Map data is kept
densely packed in order to improve data locality.

The primary use case for this implementation is that of small maps
(regardless of the size of the key set) with almost no deletions.

Set and Get operations are consistently faster than the builtin
map[int]interface{}: Get performs 1.5 times faster for data sets <= 256
entries, down to 1.1 at 2^22 entries, while Set starts at 1.3x at 256 entries,
up to 1.5 at 2^22.

The delete test is wrong since we end up deleting millions of non-existent keys.
In its current state, it shows that deletes are slower, from 0.9x at 256 entries
down to 0.7x at 2^18 entries.

With certain map sizes (> 500K entries), and depending on the host CPU cache
size, intmap.Delete seems to get suddenly faster than the builtin delete. This
is simply due to the fact that this happens when the map size reaches a sweet
spot where the builtin map starts to be adversely impacted by cache misses while
intmap.Map is not yet affected due to its smaller memory footprint. This
behavior should be not relied upon: intmap.Delete is slower for all intents and
purposes.

Benchmark sample (*Builtin are performed using a regular map[int]interface{}):

	BenchmarkIntMapSet-6            20000000                71.6 ns/op
	BenchmarkBuiltinMapSet-6        20000000                99.0 ns/op
	BenchmarkIntMapGet-6            30000000                42.2 ns/op
	BenchmarkBuiltinMapGet-6        20000000                62.5 ns/op
	BenchmarkIntMapDelete-6         30000000                37.7 ns/op
	BenchmarkBuiltinMapDelete-6     30000000                34.4 ns/op


Internals

The implementation is based on http://java-performance.info/implementing-world-fastest-java-int-to-int-hash-map/.

The stored values can be of any type. If interface{} is not suitable, just
fork/vendor the repository and change the Value type definition to the desired
value type (this will break the tests but not the implementation).
*/
package intmap

// Value type stored in the map. This can be change to suit specific needs without
// breaking the implementation (the tests will break though): fork/vendor the repository
// and change interface{} to the desired value type.
//
type Value interface{}

// KeyValue wraps a key-value pair.
//
type KeyValue struct {
	Key   int
	Value Value
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
type Map struct {
	es           []KeyValue
	size         int
	threshold    int
	freeKeyValue Value
	hasFreeKey   bool
}

func nextIdx(idx int) int {
	return idx + 1
}

// New returns a new Map initialized with the given starting capacity and fill
// ratio.
//
// See Map.Init for more details about the capacity and fillratio parameters.
//
func New(capacity int, fillratio float32) *Map {
	var m Map
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
func (m *Map) Init(capacity int, fillratio float32) {
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
	var zv Value
	m.es = make([]KeyValue, capacity)
	m.size = 0
	m.threshold = threshold
	m.freeKeyValue = zv
	m.hasFreeKey = false
}

// Set sets or resets the value for the given key.
//
func (m *Map) Set(key int, value Value) {
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
			m.es = make([]KeyValue, l)
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
			m.es[idx] = KeyValue{key, value}
			return
		}
		idx = nextIdx(idx) & mod
	}
}

func (m *Map) rehash() {
	es := m.es
	l := len(es) << 1
	if l < 0 {
		panic("map size overflows addressable space")
	}
	m.es = make([]KeyValue, l)
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
func (m *Map) Get(key int) (v Value, ok bool) {
	var zv Value
	if key == freeKey {
		if m.hasFreeKey {
			return m.freeKeyValue, true
		}
		return zv, false
	}
	mod := len(m.es) - 1
	if mod < 0 {
		return zv, false
	}
	startIdx := hash(key) & mod
	idx := startIdx
	for {
		switch m.es[idx].Key {
		case freeKey:
			return zv, false
		case key:
			return m.es[idx].Value, true
		}
		idx = nextIdx(idx) & mod
		if idx == startIdx {
			return zv, false
		}
	}
}

// Delete deletes the given key and returns true if the key was present in the map.
//
func (m *Map) Delete(key int) bool {
	if key == freeKey {
		rv := m.hasFreeKey
		m.freeKeyValue = nil
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

func (m *Map) shiftKeys(idx int) {
	var k int
	mod := len(m.es) - 1
	for {
		last := idx
		idx = nextIdx(idx) & mod
		for {
			k = m.es[idx].Key
			if k == freeKey {
				m.es[last] = KeyValue{Key: freeKey}
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
		m.es[last] = KeyValue{k, m.es[idx].Value}
	}
}

// Size returns the number if keys set in the map.
//
func (m *Map) Size() int {
	if m.hasFreeKey {
		return m.size + 1
	}
	return m.size
}

// Keys returns an unordered slice of the map keys.
//
func (m *Map) Keys() []int {
	ks := make([]int, m.Size())
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
func (m *Map) Iterator() *Iterator {
	// find a sensible default for
	return &Iterator{m: m, lastKey: freeKey ^ -1, i: -1}
}

// Iterator represents an iterator over a map.
//
type Iterator struct {
	m       *Map
	lastKey int
	i       int
}

// HasNext returns true if there are any keys left to read.
//
func (i *Iterator) HasNext() bool {
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
func (i *Iterator) Next() (key int, value Value) {
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
