
[bench-img]: https://raw.githubusercontent.com/db47h/intmap/master/bench-1.png
[godoc]: https://pkg.go.dev/github.com/db47h/intmap
[godoc-img]: https://pkg.go.dev/badge/github.com/db47h/intmap.svg
[goreport]: https://goreportcard.com/report/github.com/db47h/intmap
[goreport-img]: https://goreportcard.com/badge/github.com/db47h/intmap
[license]: https://img.shields.io/github/license/db47h/intmap.svg

# intmap

[![GoDoc][godoc-img]][godoc]
[![GoReportCard][goreport-img]][goreport]
![MIT License][license]

## Overview

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


### Internals
The implementation is based on <a href="http://java-performance.info/implementing-world-fastest-java-int-to-int-hash-map/">http://java-performance.info/implementing-world-fastest-java-int-to-int-hash-map/</a>.

The stored values can be of any type.
