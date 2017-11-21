package qcache

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		c := New(20 * time.Second)

		if got, want := c.maxPurgeInterval, 1*time.Second; got != want {
			t.Errorf("c.maxPurgeInterval = %s, got %s", got, want)
		}

		if got, want := c.itemTTL, 20*time.Second; got != want {
			t.Errorf("c.itemTTL = %s, got %s", got, want)
		}
	})

	t.Run("WithMaxPurgeInterval", func(t *testing.T) {
		c := New(0, WithMaxPurgeInterval(20*time.Second))

		if got, want := c.maxPurgeInterval, 20*time.Second; got != want {
			t.Errorf("c.maxPurgeInterval = %s, got %s", got, want)
		}
	})
}

func TestExpireAll(t *testing.T) {
	c := New(1 * time.Minute)

	for key := 0; key < 10; key++ {
		c.Set(key, "value")
	}

	c.ExpireAll()

	for key := 0; key < 10; key++ {
		gotValue, ok := c.Get(key)

		if gotValue != nil || ok {
			t.Errorf("c.Get(%d) = %q, %t; want <nil>, false", key, gotValue, ok)
		}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if got, want := len(c.items), 0; got != want {
		t.Errorf("len(c.items) = %d, want %d", got, want)
	}
	if got, want := len(c.queue), 0; got != want {
		t.Errorf("len(c.queue) = %d, want %d", got, want)
	}
}

func TestGetExisting(t *testing.T) {
	c := New(1 * time.Minute)
	key, value := "key", "value"
	c.Set(key, value)

	gotValue, ok := c.Get(key)

	if !ok {
		t.Error("ok = false, want true")
	}

	if got, want := gotValue, value; got != want {
		t.Errorf("c.Get(%q) = %q; want %q", key, got, want)
	}
}

func TestGetExpired(t *testing.T) {
	c := New(10 * time.Millisecond)

	c.Set("key", "value")

	time.Sleep(20 * time.Millisecond)

	gotValue, ok := c.Get("key")

	if gotValue != nil || ok {
		t.Errorf(`c.Get("key") = %q, %t; want <nil>, false`, gotValue, ok)
	}
}

func TestGetExpiredWithHighMaxPurgeInterval(t *testing.T) {
	c := New(10*time.Millisecond, WithMaxPurgeInterval(5*time.Minute))

	c.Set("key", "value")

	time.Sleep(20 * time.Millisecond)

	gotValue, ok := c.Get("key")

	if gotValue != nil || ok {
		t.Errorf(`c.Get("key") = %q, %t; want <nil>, false`, gotValue, ok)
	}
}

func TestGetNonExistent(t *testing.T) {
	c := New(1 * time.Minute)

	gotValue, ok := c.Get("no-key")

	if gotValue != nil || ok {
		t.Errorf(`c.Get("no-key") = %q, %t; want <nil>, false`, gotValue, ok)
	}
}

func TestSet(t *testing.T) {
	c := New(1 * time.Minute)

	c.Set("key", "value")

	gotValue, ok := c.Get("key")

	if gotValue != "value" || !ok {
		t.Errorf(`c.Get("key") = %q, %t; want "value", true`, gotValue, ok)
	}
}

func TestSetExisting(t *testing.T) {
	c := New(1 * time.Second)

	c.Set("key", "value")
	c.Set("key", "value")

	gotValue, ok := c.Get("key")

	if gotValue != "value" || !ok {
		t.Errorf(`c.Get("key") = %q, %t; want "value", true`, gotValue, ok)
	}
}

func TestSetUpdatesTTL(t *testing.T) {
	c := New(50 * time.Millisecond)
	c.Set("key", "value")
	time.Sleep(30 * time.Millisecond)
	c.Set("key", "value")
	time.Sleep(30 * time.Millisecond)
	gotValue, ok := c.Get("key")

	if gotValue != "value" || !ok {
		t.Errorf(`c.Get("key") = %v, %t; want "value", true`, gotValue, ok)
	}
}

func TestSetUpdatesValue(t *testing.T) {
	c := New(50 * time.Millisecond)
	c.Set("key", "value")
	time.Sleep(30 * time.Millisecond)
	c.Set("key", "value2")
	time.Sleep(30 * time.Millisecond)
	gotValue, ok := c.Get("key")

	if gotValue != "value2" || !ok {
		t.Errorf(`c.Get("key") = %v, %t; want "value", true`, gotValue, ok)
	}
}

func TestSize(t *testing.T) {
	c := New(10*time.Millisecond, WithMaxPurgeInterval(0))

	if got, want := c.Size(), 0; got != want {
		t.Errorf("c.Size() = %d, want %d", got, want)
	}

	c.Set("key", "value")

	if got, want := c.Size(), 1; got != want {
		t.Errorf("c.Size() = %d, want %d", got, want)
	}

	time.Sleep(20 * time.Millisecond)

	if got, want := c.Size(), 0; got != want {
		t.Errorf("c.Size() = %d, want %d", got, want)
	}
}

func TestTimerRestart(t *testing.T) {
	c := New(10*time.Millisecond, WithMaxPurgeInterval(0))

	c.Set("key-1", "value-1")

	time.Sleep(20 * time.Millisecond)

	c.Set("key-2", "value-2")

	time.Sleep(20 * time.Millisecond)

	c.mu.RLock()
	defer c.mu.RUnlock()

	if got, want := len(c.items), 0; got != want {
		t.Errorf("len(c.items) = %d, want %d", got, want)
	}
	if got, want := len(c.queue), 0; got != want {
		t.Errorf("len(c.queue) = %d, want %d", got, want)
	}
}

func TestMultipleExpiration(t *testing.T) {
	c := New(20*time.Millisecond, WithMaxPurgeInterval(5*time.Millisecond))

	for n := 0; n < 100; n++ {
		c.Set(n, "value")
	}

	time.Sleep(10 * time.Millisecond)

	for n := 100; n < 200; n++ {
		c.Set(n, "value")
	}

	c.mu.RLock()
	if got, want := len(c.items), 200; got != want {
		t.Errorf("[0] len(c.items) = %d, want %d", got, want)
	}
	if got, want := len(c.queue), 200; got != want {
		t.Errorf("[0] len(c.queue) = %d, want %d", got, want)
	}
	c.mu.RUnlock()

	time.Sleep(15 * time.Millisecond)

	c.mu.RLock()
	if got, want := len(c.items), 100; got != want {
		t.Errorf("[1] len(c.items) = %d, want %d", got, want)
	}
	if got, want := len(c.queue), 100; got != want {
		t.Errorf("[1] len(c.queue) = %d, want %d", got, want)
	}
	c.mu.RUnlock()

	time.Sleep(10 * time.Millisecond)

	c.mu.RLock()
	if got, want := len(c.items), 0; got != want {
		t.Errorf("[2] len(c.items) = %d, want %d", got, want)
	}
	if got, want := len(c.queue), 0; got != want {
		t.Errorf("[2] len(c.queue) = %d, want %d", got, want)
	}
	c.mu.RUnlock()
}

func TestExpireEmptyList(t *testing.T) {
	c := New(5 * time.Minute)

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("got panic, want no panic")
			panic(r)
		}
	}()

	c.expire()
}

func TestMaxPurgeInterval(t *testing.T) {
	c := New(10*time.Millisecond, WithMaxPurgeInterval(20*time.Millisecond))

	c.Set("key", "value")

	time.Sleep(15 * time.Millisecond)

	c.mu.RLock()
	if got, want := len(c.items), 1; got != want {
		t.Errorf("item purged too early: len(c.items) = %d, want %d", got, want)
	}
	if got, want := len(c.queue), 1; got != want {
		t.Errorf("item purged too early: len(c.queue) = %d, want %d", got, want)
	}
	c.mu.RUnlock()

	time.Sleep(15 * time.Millisecond)

	c.mu.RLock()
	if got, want := len(c.items), 0; got != want {
		t.Errorf("item not purged: len(c.items) = %d, want %d", got, want)
	}
	if got, want := len(c.queue), 0; got != want {
		t.Errorf("item not purged: len(c.queue) = %d, want %d", got, want)
	}
	c.mu.RUnlock()
}

func BenchmarkGetExisting(b *testing.B) {
	c := New(5 * time.Minute)

	const numKeys = 100000

	for key := 0; key < numKeys; key++ {
		c.Set(key, "value")
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.Get(i % numKeys)
	}

	b.StopTimer()

	c.ExpireAll()
}

func BenchmarkGetNonExistent(b *testing.B) {
	c := New(5 * time.Minute)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.Get(i)
	}

	b.StopTimer()

	c.ExpireAll()
}

func BenchmarkSet(b *testing.B) {
	c := New(5 * time.Minute)

	const numKeys = 100000

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if i%numKeys == 0 {
			b.StopTimer()
			c.ExpireAll()
			b.StartTimer()
		}
		c.Set(i%numKeys, "value")
	}

	b.StopTimer()

	c.ExpireAll()
}

func BenchmarkSetExisting(b *testing.B) {
	c := New(5 * time.Minute)

	const numKeys = 100000

	for key := 0; key < numKeys; key++ {
		c.Set(key, "value")
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.Set(i%numKeys, "value")
	}

	b.StopTimer()

	c.ExpireAll()
}
