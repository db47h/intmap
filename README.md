
[bench-img]: https://raw.githubusercontent.com/db47h/intmap/master/bench-1.png
[godoc]: https://godoc.org/github.com/db47h/intmap
[godoc-img]: https://godoc.org/github.com/db47h/intmap?status.svg
[goreport]: https://goreportcard.com/report/github.com/db47h/intmap
[goreport-img]: https://goreportcard.com/badge/github.com/db47h/intmap
[license]: https://img.shields.io/github/license/db47h/intmap.svg

# intmap

[![GoDoc][godoc-img]][godoc]
[![GoReportCard][goreport-img]][goreport]
![MIT License][license]

`import "github.com/db47h/intmap"`

* [Overview](#pkg-overview)
* [Index](#pkg-index)

## <a name="pkg-overview">Overview</a>
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

Performance graph for different map sizes (shorter bars are better):

![benchmarks][bench-img]

### Internals
The implementation is based on <a href="http://java-performance.info/implementing-world-fastest-java-int-to-int-hash-map/">http://java-performance.info/implementing-world-fastest-java-int-to-int-hash-map/</a>.

The stored values can be of any type. If interface{} is not suitable, just
fork/vendor the repository and change the Value type definition to the desired
value type (this will break the tests but not the implementation).




## <a name="pkg-index">Index</a>
* [type Iterator](#Iterator)
  * [func (i *Iterator) HasNext() bool](#Iterator.HasNext)
  * [func (i *Iterator) Next() (key int, value Value)](#Iterator.Next)
* [type KeyValue](#KeyValue)
* [type Map](#Map)
  * [func New(capacity int, fillratio float32) *Map](#New)
  * [func (m *Map) Delete(key int) bool](#Map.Delete)
  * [func (m *Map) Get(key int) (v Value, ok bool)](#Map.Get)
  * [func (m *Map) Init(capacity int, fillratio float32)](#Map.Init)
  * [func (m *Map) Iterator() *Iterator](#Map.Iterator)
  * [func (m *Map) Keys() []int](#Map.Keys)
  * [func (m *Map) Set(key int, value Value)](#Map.Set)
  * [func (m *Map) Size() int](#Map.Size)
* [type Value](#Value)


#### <a name="pkg-files">Package files</a>
[map.go](/src/target/map.go) [map_64.go](/src/target/map_64.go) 






## <a name="Iterator">type</a> [Iterator](/src/target/map.go?s=8561:8625#L333)
``` go
type Iterator struct {
    // contains filtered or unexported fields
}
```
Iterator represents an iterator over a map.










### <a name="Iterator.HasNext">func</a> (\*Iterator) [HasNext](/src/target/map.go?s=8690:8723#L341)
``` go
func (i *Iterator) HasNext() bool
```
HasNext returns true if there are any keys left to read.




### <a name="Iterator.Next">func</a> (\*Iterator) [Next](/src/target/map.go?s=9285:9333#L369)
``` go
func (i *Iterator) Next() (key int, value Value)
```
Next returns the next key/value pair. Calling Next several times in a row
without calling HasNext in between will yield the same result.




## <a name="KeyValue">type</a> [KeyValue](/src/target/map.go?s=2316:2364#L53)
``` go
type KeyValue struct {
    Key   int
    Value Value
}
```
KeyValue wraps a key-value pair.










## <a name="Map">type</a> [Map](/src/target/map.go?s=3099:3218#L77)
``` go
type Map struct {
    // contains filtered or unexported fields
}
```
Map is a fast int to interface{} map. Map data is kept densely packed in
order to improve data locality.

The primary use case for this implementation is that of small maps
(regardless of the size of the key set) with almost no deletions.

A Map can be used directly: the start capacity will be set to 8 entries and
the fill ratio 87.5%. If the rough map size is known in advance, it is
however preferable to initialize it with New or Init for better performance,
especially when initializing a large number of maps.

When the size of a Map grows over the fill ratio, its capacity is doubled.
Maps are never shrunk when deleting keys.







### <a name="New">func</a> [New](/src/target/map.go?s=3440:3486#L94)
``` go
func New(capacity int, fillratio float32) *Map
```
New returns a new Map initialized with the given starting capacity and fill
ratio.

See Map.Init for more details about the capacity and fillratio parameters.





### <a name="Map.Delete">func</a> (\*Map) [Delete](/src/target/map.go?s=6664:6698#L232)
``` go
func (m *Map) Delete(key int) bool
```
Delete deletes the given key and returns true if the key was present in the map.




### <a name="Map.Get">func</a> (\*Map) [Get](/src/target/map.go?s=6117:6162#L202)
``` go
func (m *Map) Get(key int) (v Value, ok bool)
```
Get returns the value associated with the given key and ok set to true if the key exists.
If the keys does not exist, it returns the zero value for the Value type and false.




### <a name="Map.Init">func</a> (\*Map) [Init](/src/target/map.go?s=4555:4606#L125)
``` go
func (m *Map) Init(capacity int, fillratio float32)
```
Init initializes the Map with the given initial capacity and fill ratio.

If the Map already contains data, it will be reset.

The capacity is rounded up to the next exponent of two. Values < 2 are
rounded up to 2. It must be less than or equal to 0x40000000 for 32 bits ints
and 0x4000000000000000 for 64 bits ints.

The fill ratio should be between 0 (0%) and 1 (100%) exclusive. Values out of
this range are silently rounded to the lowest or largest possible value. When
the size of a Map grows over the fill ratio, its capacity is doubled.

The fill ratio will be rounded as follows:


	threshold := int(float32(capacity) * fillratio)
	if threshold <= 0 {
	    threshold = 1
	} else if threshold >= capacity {
	    threshold = capacity - 1
	}
	fillratio_rounded := float32(threshold) / float32(capacity)

i.e. requesting a Map of initial capacity 2 with any fill ratio > 0.5 will
result in a real fill ratio of 0.5 due to integer rounding.




### <a name="Map.Iterator">func</a> (\*Map) [Iterator](/src/target/map.go?s=8385:8419#L326)
``` go
func (m *Map) Iterator() *Iterator
```
Iterator returns an iterator over the map's key/value pairs.


	for i := m.Iterator(); i.HasNext(); {
		k, v := i.Next()
		fmt.Printf("m[%v] = %v\n", k, v)
	}

While iterating over a map, deleting the value of the last key returned by
Next is supported as well of changing the value of any existing key.
Inserting new keys or deleting any other keys will break the iterator.




### <a name="Map.Keys">func</a> (\*Map) [Keys](/src/target/map.go?s=7762:7788#L298)
``` go
func (m *Map) Keys() []int
```
Keys returns an unordered slice of the map keys.




### <a name="Map.Set">func</a> (\*Map) [Set](/src/target/map.go?s=5078:5117#L149)
``` go
func (m *Map) Set(key int, value Value)
```
Set sets or resets the value for the given key.




### <a name="Map.Size">func</a> (\*Map) [Size](/src/target/map.go?s=7620:7644#L289)
``` go
func (m *Map) Size() int
```
Size returns the number if keys set in the map.




## <a name="Value">type</a> [Value](/src/target/map.go?s=2253:2275#L49)
``` go
type Value interface{}
```
Value type stored in the map. This can be change to suit specific needs without
breaking the implementation (the tests will break though): fork/vendor the repository
and change interface{} to the desired value type.

