package cache_test

import (
	"testing"
	"time"

	"github.com/JekaMas/pretty"
	"github.com/dchest/uniuri"

	"video-chat/api/cache"
)

var (
	longTime  = time.Second * 600000
	shortTime = time.Second * 1
)

var testValues = []cache.Session{{Id: "session-1"}, {Id: "session-2"}}

func TestCache_Get_OneKeyExists(t *testing.T) {
	c := cache.New(0, 0, time.Minute, time.Second * 1, nil)
	defer c.Close()

	key := uniuri.NewLen(8)
	toStore := &testValues[0]

	prev, err := c.Set(key, toStore, longTime)
	if prev != nil {
		t.Fatal("prev != nil")
	}
	if err != nil {
		t.Fatal(err)
	}

	gotValue, err := c.Get(key)
	if err != nil {
		t.Fatal(err)
	}

	diff := pretty.DiffMessage(gotValue, toStore)
	if len(diff) != 0 {
		t.Fatal(diff)
	}
}

func TestCache_Get_OneKey_NotExists(t *testing.T) {
	c := cache.New(0, 0, time.Minute, time.Second * 1, nil)
	defer c.Close()

	key := uniuri.NewLen(8)
	toStore := &testValues[0]

	prev, err := c.Set(key, toStore, longTime)
	if prev != nil {
		t.Fatal("prev != nil")
	}
	if err != nil {
		t.Fatal(err)
	}

	key = uniuri.NewLen(8)
	gotValue, err := c.Get(key)
	if gotValue != nil {
		t.Fatal("gotValue != nil")
	}
	if err != cache.NotFoundError {
		t.Fatal(err)
	}
}

func TestCache_Get_OneKey_Repeat(t *testing.T) {
	c := cache.New(0, 0, time.Minute, time.Second * 1, nil)
	defer c.Close()

	key := uniuri.NewLen(8)
	toStore := &testValues[0]

	prev, err := c.Set(key, toStore, longTime)
	if prev != nil {
		t.Fatal("prev != nil")
	}
	if err != nil {
		t.Fatal(err)
	}

	gotValue, err := c.Get(key)
	if err != nil {
		t.Fatal(err)
	}

	diff := pretty.DiffMessage(gotValue, toStore)
	if len(diff) != 0 {
		t.Fatal(diff)
	}

	//retry
	gotValue, err = c.Get(key)
	if err != nil {
		t.Fatal(err)
	}

	diff = pretty.DiffMessage(gotValue, toStore)
	if len(diff) != 0 {
		t.Fatal(diff)
	}
}

func TestCache_Get_OneKey_TTL_ClearByGC(t *testing.T) {
	c := cache.New(50000, 5, time.Minute, time.Second * 1, nil)
	defer c.Close()

	key := uniuri.NewLen(8)
	toStore := &testValues[0]

	prev, err := c.Set(key, toStore, shortTime)
	if prev != nil {
		t.Fatal("prev != nil")
	}
	if err != nil {
		t.Fatal(err)
	}

	gotValue, err := c.Get(key)
	if err != nil {
		t.Fatal(err)
	}

	diff := pretty.DiffMessage(gotValue, toStore)
	if len(diff) != 0 {
		t.Fatal(diff)
	}

	if n := c.Size(); n != 1 {
		t.Fatal(n)
	}

	//retry after TTL
	time.Sleep(time.Second * 6)

	gotValue, err = c.Get(key)
	if gotValue != nil {
		t.Fatal("gotValue != nil")
	}
	if err != cache.NotFoundError {
		t.Fatal(err)
	}

	if n := c.Size(); n != 0 {
		t.Fatal(n)
	}
}

func TestCache_Get_OneKey_TTL_ClearByGC_OneKeyLive(t *testing.T) {
	c := cache.New(50000, 5, time.Minute, time.Second * 1, nil)
	defer c.Close()

	key := uniuri.NewLen(8)
	toStore := &testValues[0]

	prev, err := c.Set(key, toStore, shortTime)
	if prev != nil {
		t.Fatal("prev != nil")
	}
	if err != nil {
		t.Fatal(err)
	}

	gotValue, err := c.Get(key)
	if err != nil {
		t.Fatal(err)
	}

	diff := pretty.DiffMessage(gotValue, toStore)
	if len(diff) != 0 {
		t.Fatal(diff)
	}

	if n := c.Size(); n != 1 {
		t.Fatal(n)
	}

	//second key
	toStore = &testValues[1]
	key2 := uniuri.NewLen(8)

	prev, err = c.Set(key2, toStore, longTime)
	if prev != nil {
		t.Fatal("prev != nil")
	}
	if err != nil {
		t.Fatal(err)
	}

	gotValue, err = c.Get(key2)
	if err != nil {
		t.Fatal(err)
	}

	diff = pretty.DiffMessage(gotValue, toStore)
	if len(diff) != 0 {
		t.Fatal(diff)
	}

	if n := c.Size(); n != 2 {
		t.Fatal(n)
	}

	//retry after TTL
	time.Sleep(time.Second * 6)

	// expired key
	gotValue, err = c.Get(key)
	if gotValue != nil {
		t.Fatal("gotValue != nil")
	}
	if err != cache.NotFoundError {
		t.Fatal(err)
	}

	if n := c.Size(); n != 1 {
		t.Fatal(n)
	}

	// live key
	gotValue, err = c.Get(key2)
	if gotValue == nil {
		t.Fatal("gotValue == nil")
	}
	if err != nil {
		t.Fatal(err)
	}

	if n := c.Size(); n != 1 {
		t.Fatal(n)
	}
}

func TestCache_Get_Close(t *testing.T) {
	c := cache.New(0, 0, time.Minute, time.Second * 1, nil)
	c.Close()

	_, err := c.Get(uniuri.NewLen(8))
	if err != cache.CloseError {
		t.Fatal(err)
	}
}

func TestCache_Delete(t *testing.T) {
	c := cache.New(0, 0, time.Minute, time.Second * 1, nil)
	defer c.Close()

	key := uniuri.NewLen(8)
	toStore := &testValues[0]

	prev, err := c.Set(key, toStore, longTime)
	if prev != nil {
		t.Fatal("prev != nil")
	}
	if err != nil {
		t.Fatal(err)
	}

	err = c.Delete(key)
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.Get(key)
	if err != cache.NotFoundError {
		t.Fatal(err)
	}

	if n := c.Size(); n != 0 {
		t.Fatal(n)
	}
}

func TestCache_Delete_Close(t *testing.T) {
	c := cache.New(0, 0, time.Minute, time.Second * 1, nil)
	c.Close()

	err := c.Delete(uniuri.NewLen(8))
	if err != cache.CloseError {
		t.Fatal(err)
	}
}

func TestCache_Delete_Repeat(t *testing.T) {
	c := cache.New(0, 0, time.Minute, time.Second * 1, nil)
	defer c.Close()

	key := uniuri.NewLen(8)
	toStore := &testValues[0]

	prev, err := c.Set(key, toStore, longTime)
	if prev != nil {
		t.Fatal("prev != nil")
	}
	if err != nil {
		t.Fatal(err)
	}

	err = c.Delete(key)
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.Get(key)
	if err != cache.NotFoundError {
		t.Fatal(err)
	}

	if n := c.Size(); n != 0 {
		t.Fatal(n)
	}

	// repeat
	err = c.Delete(key)
	if err != cache.NotFoundError {
		t.Fatal(err)
	}

	_, err = c.Get(key)
	if err != cache.NotFoundError {
		t.Fatal(err)
	}

	if n := c.Size(); n != 0 {
		t.Fatal(n)
	}
}

func TestCache_Size_Zero(t *testing.T) {
	c := cache.New(0, 0, time.Minute, time.Second * 1, nil)
	defer c.Close()

	if n := c.Size(); n != 0 {
		t.Fatal(n)
	}
}

func TestCache_Size_One(t *testing.T) {
	c := cache.New(0, 0, time.Minute, time.Second * 1, nil)
	defer c.Close()

	key := uniuri.NewLen(8)
	toStore := &testValues[0]

	prev, err := c.Set(key, toStore, longTime)
	if prev != nil {
		t.Fatal("prev != nil")
	}
	if err != nil {
		t.Fatal(err)
	}

	if n := c.Size(); n != 1 {
		t.Fatal(n)
	}
}

func TestCache_Size_Close(t *testing.T) {
	c := cache.New(0, 0, time.Minute, time.Second * 1, nil)

	key := uniuri.NewLen(8)
	toStore := &testValues[0]

	prev, err := c.Set(key, toStore, longTime)
	if prev != nil {
		t.Fatal("prev != nil")
	}
	if err != nil {
		t.Fatal(err)
	}

	if n := c.Size(); n != 1 {
		t.Fatal(n)
	}

	c.Close()

	if n := c.Size(); n != 0 {
		t.Fatal(n)
	}
}

func TestCache_Size_Many(t *testing.T) {
	c := cache.New(0, 0, time.Minute, time.Second * 1, nil)
	defer c.Close()

	for i := 0; i < 2; i++ {
		key := uniuri.NewLen(8)
		toStore := &testValues[i]

		prev, err := c.Set(key, toStore, longTime)
		if prev != nil {
			t.Fatal("prev != nil")
		}
		if err != nil {
			t.Fatal(err)
		}

		if n := c.Size(); n != i+1 {
			t.Fatal(n)
		}
	}
}

func TestCache_Close(t *testing.T) {
	c := cache.New(0, 0, time.Minute, time.Second * 1, nil)

	key := uniuri.NewLen(8)
	toStore := &testValues[0]

	prev, err := c.Set(key, toStore, longTime)
	if prev != nil {
		t.Fatal("prev != nil")
	}
	if err != nil {
		t.Fatal(err)
	}

	c.Close()

	prev, err = c.Set(key, toStore, longTime)
	if prev != nil {
		t.Fatal("prev != nil")
	}
	if err != cache.CloseError {
		t.Fatal(err)
	}

	if n := c.Size(); n != 0 {
		t.Fatal(n)
	}
}

func TestCache_Close_Repeat(t *testing.T) {
	c := cache.New(0, 0, time.Minute, time.Second * 1, nil)

	key := uniuri.NewLen(8)
	toStore := &testValues[0]

	prev, err := c.Set(key, toStore, longTime)
	if prev != nil {
		t.Fatal("prev != nil")
	}
	if err != nil {
		t.Fatal(err)
	}

	c.Close()
	c.Close()

	prev, err = c.Set(key, toStore, longTime)
	if prev != nil {
		t.Fatal("prev != nil")
	}
	if err != cache.CloseError {
		t.Fatal(err)
	}

	if n := c.Size(); n != 0 {
		t.Fatal(n)
	}
}

func TestCache_Set_One_Repeat(t *testing.T) {
	c := cache.New(0, 0, time.Minute, time.Second * 1, nil)
	defer c.Close()

	key := uniuri.NewLen(8)
	toStore := &testValues[0]

	prev, err := c.Set(key, toStore, longTime)
	if prev != nil {
		t.Fatal("prev != nil")
	}
	if err != nil {
		t.Fatal(err)
	}

	gotValue, err := c.Get(key)
	if err != nil {
		t.Fatal(err)
	}

	diff := pretty.DiffMessage(gotValue, toStore)
	if len(diff) != 0 {
		t.Fatal(diff)
	}

	if n := c.Size(); n != 1 {
		t.Fatal(n)
	}

	//retry
	toStore = &testValues[1]
	gotValue, err = c.Set(key, toStore, longTime)
	if gotValue == nil {
		t.Fatal("gotValue == nil")
	}
	if err != nil {
		t.Fatal(err)
	}
	diff = pretty.DiffMessage(gotValue, &testValues[0])
	if len(diff) != 0 {
		t.Fatal(diff)
	}

	gotValue, err = c.Get(key)
	if err != nil {
		t.Fatal(err)
	}

	diff = pretty.DiffMessage(gotValue, toStore)
	if len(diff) != 0 {
		t.Fatal(diff)
	}

	if n := c.Size(); n != 1 {
		t.Fatal(n)
	}
}

func TestCache_Set_Overflow(t *testing.T) {
	cacheSize := 50_000
	c := cache.New(cacheSize, 5, time.Minute, time.Second * 1, nil)
	defer c.Close()

	k := uniuri.NewLen(8)
	for i := 0; i < cacheSize; i++ {
		k = uniuri.NewLen(8)
		toStore := &testValues[i % len(testValues)]

		prev, err := c.Set(k, toStore, time.Second * 1)
		if prev != nil {
			t.Fatal("prev != nil")
		}
		if err != nil {
			t.Fatal(err)
		}
		if c.Size() != i + 1 {
			t.Fatal("incorrect c size")
		}
	}
	if c.Size() != cacheSize {
		t.Fatal("incorrect c size")
	}

	key := uniuri.NewLen(8)
	toStore := &testValues[0]

	prev, err := c.Set(key, toStore, longTime)
	if prev != nil {
		t.Fatal("prev != nil")
	}
	if err != cache.MaxSizeExceed {
		t.Fatal("incorrect error")
	}

	gotValue, err := c.Get(key)
	if gotValue != nil {
		t.Fatal("gotValue != nil")
	}
	if err != cache.NotFoundError {
		t.Fatal(err)
	}

	prev, err = c.Set(k, toStore, longTime)
	if prev == nil {
		t.Fatal("prev == nil")
	}
	if err != nil {
		t.Fatal(err)
	}
	diff := pretty.DiffMessage(prev, &testValues[(cacheSize - 1) % len(testValues)])
	if len(diff) != 0 {
		t.Fatal(diff)
	}
	if c.Size() != cacheSize {
		t.Fatal("incorrect c size")
	}
	gotValue, err = c.Get(k)
	if gotValue == nil {
		t.Fatal("gotValue == nil")
	}
	if err != nil {
		t.Fatal(err)
	}
	diff = pretty.DiffMessage(gotValue, toStore)
	if len(diff) != 0 {
		t.Fatal(diff)
	}

	time.Sleep(time.Second * 6)

	if c.Size() != 1 {
		t.Fatal("incorrect c size")
	}

	prev, err = c.Set(key, toStore, longTime)
	if prev != nil {
		t.Fatal("prev != nil")
	}
	if err != nil {
		t.Fatal(err)
	}

	if c.Size() != 2 {
		t.Fatal("incorrect c size")
	}
	gotValue, err = c.Get(key)
	if gotValue == nil {
		t.Fatal("gotValue == nil")
	}
	if err != nil {
		t.Fatal(err)
	}
	diff = pretty.DiffMessage(gotValue, toStore)
	if len(diff) != 0 {
		t.Fatal(diff)
	}
}

func TestCache_Add_One_Repeat(t *testing.T) {
	c := cache.New(0, 0, time.Minute, time.Second * 1, nil)
	defer c.Close()

	key := uniuri.NewLen(8)
	toStore := &testValues[0]

	err := c.Add(key, toStore, longTime)
	if err != nil {
		t.Fatal(err)
	}

	gotValue, err := c.Get(key)
	if err != nil {
		t.Fatal(err)
	}
	diff := pretty.DiffMessage(gotValue, toStore)
	if len(diff) != 0 {
		t.Fatal(diff)
	}
	if c.Size() != 1 {
		t.Fatal("incorrect c size")
	}

	//retry
	toStore = &testValues[1]
	err = c.Add(key, toStore, longTime)
	if err != nil && err.Error() != "item '" + key + "' already exists" {
		t.Fatal(err)
	}
	if c.Size() != 1 {
		t.Fatal("incorrect c size")
	}
}

func TestCache_Replace_One_Repeat(t *testing.T) {
	c := cache.New(0, 0, time.Minute, time.Second * 1, nil)
	defer c.Close()

	key := uniuri.NewLen(8)
	toStore := &testValues[0]

	prev, err := c.Replace(key, toStore, longTime)
	if prev != nil {
		t.Fatal("prev != nil")
	}
	if err != nil && err.Error() != "item '" + key + "' doesn't exist" {
		t.Fatal(err)
	}

	prev, err = c.Set(key, toStore, longTime)
	if prev != nil {
		t.Fatal("prev != nil")
	}
	if err != nil {
		t.Fatal(err)
	}
	if c.Size() != 1 {
		t.Fatal("incorrect c size")
	}

	prev, err = c.Replace(key, &testValues[1], longTime)
	if prev == nil {
		t.Fatal("prev == nil")
	}
	if err != nil {
		t.Fatal(err)
	}
	if c.Size() != 1 {
		t.Fatal("incorrect c size")
	}
	diff := pretty.DiffMessage(prev, toStore)
	if len(diff) != 0 {
		t.Fatal(diff)
	}

	gotValue, err := c.Get(key)
	if gotValue == nil {
		t.Fatal("gotValue == nil")
	}
	diff = pretty.DiffMessage(gotValue, &testValues[1])
	if len(diff) != 0 {
		t.Fatal(diff)
	}
}

func TestCache_KeepAlive_One_Repeat(t *testing.T) {
	c := cache.New(0, 0, time.Minute, time.Second * 1, nil)
	defer c.Close()

	key := uniuri.NewLen(8)
	toStore := &testValues[0]

	gotValue, err := c.KeepAlive(key, shortTime)
	if gotValue != nil {
		t.Fatal("gotValue != nil")
	}
	if err != cache.NotFoundError {
		t.Fatal(err)
	}

	err = c.Add(key, toStore, shortTime)
	if err != nil {
		t.Fatal(err)
	}
	if c.Size() != 1 {
		t.Fatal("incorrect c size")
	}

	gotValue, err = c.KeepAlive(key, longTime)
	if gotValue == nil {
		t.Fatal("gotValue == nil")
	}
	if err != nil {
		t.Fatal(err)
	}
	if c.Size() != 1 {
		t.Fatal("incorrect c size")
	}
	diff := pretty.DiffMessage(gotValue, toStore)
	if len(diff) != 0 {
		t.Fatal(diff)
	}
	time.Sleep(time.Second * 5)

	if c.Size() != 1 {
		t.Fatal("incorrect c size")
	}
	gotValue, err = c.Get(key)
	if gotValue == nil {
		t.Fatal("gotValue == nil")
	}
	diff = pretty.DiffMessage(gotValue, toStore)
	if len(diff) != 0 {
		t.Fatal(diff)
	}
}

func TestCache_OnEvict(t *testing.T) {
	key := uniuri.NewLen(8)
	toStore := &testValues[0]

	c := cache.New(50_000, 5, time.Second * 1, time.Second * 1, func(value *cache.Session) {
		diff := pretty.DiffMessage(value, toStore)
		if len(diff) != 0 {
			t.Fatal(diff)
		}
	})
	defer c.Close()

	err := c.Add(key, toStore, cache.DefaultExpiration)
	if err != nil {
		t.Fatal(err)
	}
	if c.Size() != 1 {
		t.Fatal("incorrect c size")
	}
	gotValue, err := c.Get(key)
	if gotValue == nil {
		t.Fatal("gotValue == nil")
	}
	diff := pretty.DiffMessage(gotValue, toStore)
	if len(diff) != 0 {
		t.Fatal(diff)
	}

	time.Sleep(time.Second * 6)
	if c.Size() != 0 {
		t.Fatal("incorrect c size")
	}
}

func TestCache_Set_Persistent(t *testing.T) {
	c := cache.New(50_000, 5, time.Second * 5, time.Second * 1, nil)
	defer c.Close()

	key := uniuri.NewLen(8)
	toStore := &testValues[0]

	prev, err := c.Set(key, toStore, cache.NoExpiration)
	if prev != nil {
		t.Fatal("prev != nil")
	}
	if err != nil {
		t.Fatal(err)
	}

	gotValue, err := c.Get(key)
	if err != nil {
		t.Fatal(err)
	}
	diff := pretty.DiffMessage(gotValue, toStore)
	if len(diff) != 0 {
		t.Fatal(diff)
	}
	if c.Size() != 1 {
		t.Fatal("incorrect c size")
	}

	time.Sleep(time.Second * 10)

	//retry
	gotValue, err = c.Get(key)
	if gotValue == nil {
		t.Fatal("gotValue == nil")
	}
	if err != nil {
		t.Fatal(err)
	}
	diff = pretty.DiffMessage(gotValue, toStore)
	if len(diff) != 0 {
		t.Fatal(diff)
	}
	if c.Size() != 1 {
		t.Fatal("incorrect c size")
	}
}
