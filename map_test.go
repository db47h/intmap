package intmap_test

import (
	"flag"
	"math/rand"
	"testing"

	"github.com/db47h/intmap"
)

var keyMax = flag.Int("maxkey", 1<<8, "Maximum key value")

type Value interface{}

var result Value
var bResult bool

func BenchmarkIntMapSet(b *testing.B) {
	var m intmap.Map[Value]
	rand.Seed(424242)
	for i := 0; i < b.N; i++ {
		v := rand.Intn(*keyMax)
		m.Set(v, Value(v))
	}
}

func BenchmarkBuiltinMapSet(b *testing.B) {
	m := make(map[int]Value)
	rand.Seed(424242)
	for i := 0; i < b.N; i++ {
		v := rand.Intn(*keyMax)
		m[v] = Value(v)
	}
}

func BenchmarkIntMapGet(b *testing.B) {
	var m intmap.Map[Value]
	for i := 0; i < *keyMax; i++ {
		m.Set(i, Value(i))
	}
	rand.Seed(424242)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v, ok := m.Get(rand.Intn(*keyMax))
		if ok {
			result = v
		}
	}
}

func BenchmarkBuiltinMapGet(b *testing.B) {
	m := make(map[int]Value)
	for i := 0; i < *keyMax; i++ {
		m[i] = Value(i)
	}
	rand.Seed(424242)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v, ok := m[rand.Intn(*keyMax)]
		if ok {
			result = v
		}
	}
}

func BenchmarkIntMapDelete(b *testing.B) {
	var m intmap.Map[Value]
	for i := 0; i < *keyMax; i++ {
		m.Set(i, Value(i))
	}
	rand.Seed(424242)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bResult = m.Delete(rand.Intn(*keyMax))
	}
}

func BenchmarkBuiltinMapDelete(b *testing.B) {
	var m = make(map[int]Value)
	for i := 0; i < *keyMax; i++ {
		m[i] = Value(i)
	}
	rand.Seed(424242)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		delete(m, rand.Intn(*keyMax))
	}
}

func TestMap(t *testing.T) {
	rand.Seed(424242)
	var mm intmap.Map[Value]
	var sm = make(map[int]Value)

	for i := 0; i < 1000000; i++ {
		if i%100 == 0 {
			for d := 0; d < 10; d++ {
				k := rand.Intn(1024)
				mm.Delete(k)
				delete(sm, k)
			}
		}
		k := rand.Intn(1024)
		v := rand.Int()
		mm.Set(k, Value(v))
		sm[k] = Value(v)
	}

	if len(sm) != mm.Len() {
		t.Fatalf("bad size: expected %d, got %d", len(sm), mm.Len())
	}
	for k, v := range sm {
		vv, ok := mm.Get(k)
		if !ok {
			t.Fatalf("Key %d not found", k)
		}
		if vv != v {
			t.Fatalf("bad value for key %d, expected %v, got %v", k, v, vv)
		}
	}
}

func TestMap_Iter(t *testing.T) {
	var m intmap.Map[Value]

	m.Set(42, 21)
	m.Set(22, 11)
	m.Set(68, 34)
	// m.Set(0, 1337)

	for i := m.Iterator(); i.HasNext(); {
		i.HasNext() // No-op
		i.HasNext()
		k, _ := i.Next()
		if k == 42 || k == 0 {
			m.Delete(k)
		}
		i.Next() // No-Op
		i.Next()
		i.Next()
	}
	for i := m.Iterator(); i.HasNext(); {
		k, v := i.Next()
		switch k {
		case 22, 68:
			if v.(int) != k/2 {
				t.Errorf("bad value for key %d: expected %v, got %v", k, k/2, v)
			}
		default:
			t.Errorf("unexpected key %d", k)
		}
	}
}
