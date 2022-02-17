// Copyright (c) 2022 XMM project Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// XMM Project Site: https://github.com/heiyeluren
// XMM URL: https://github.com/heiyeluren/XMM
//

package xmm

import (
	"fmt"
	"github.com/spf13/cast"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof" // 会自动注册 handler 到 http server，方便通过 http 接口获取程序运行采样报告
	"os"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"
	"unsafe"
)

func TestName(t *testing.T) {
	fmt.Println(heapRawMemoryBytes, _PageSize, pagesPerRawMemory, logHeapRawMemoryBytes, RawMemoryL1Bits, RawMemoryL2Bits)
	fmt.Println(heapAddrBits, logHeapRawMemoryBytes, RawMemoryL1Bits, 1<<28)
	fmt.Println(1<<RawMemoryL2Bits*_PageSize, heapRawMemoryBytes)
	fmt.Println(_FixAllocChunk)
}

func TestNewXAllocator(t *testing.T) {
	var data []unsafe.Pointer
	acllotor := newXAllocator(unsafe.Sizeof(xSpan{}))
	for i := 0; i < 100000; i++ {
		p, err := acllotor.alloc()
		if err != nil {
			t.Fatal(err)
		}
		data = append(data, p)
		span2 := (*xSpan)(p)
		span2.npages = uintptr(i)
		span2.startAddr = uintptr(i)
		span2.Init(0.75, nil)

		if sss := (*xSpan)(p); sss.npages != uintptr(i) {
			t.Fatalf("%+v\n", (*xSpan)(p))
		}
	}
	acllotor2 := newXAllocator(unsafe.Sizeof(xChunk{}))
	for i := 0; i < 10000; i++ {
		p, err := acllotor2.alloc()
		if err != nil {
			t.Fatal(err)
		}
		data = append(data, p)
		span2 := (*xChunk)(p)
		span2.npages = uintptr(i * 2)
		span2.startAddr = uintptr(i * 2)

		if sss := (*xChunk)(p); sss.npages != uintptr(i*2) {
			t.Fatalf("%+v\n", (*xChunk)(p))
		}
	}

	for i, span := range data {
		if i < 100000 {
			if sss := (*xSpan)(span); sss.npages != uintptr(i) {
				t.Fatalf("%+v\n", (*xSpan)(span))
			}
			continue
		}
		if sss := (*xChunk)(span); sss.npages != uintptr((i-100000)*2) {
			t.Fatalf("%+v\n", (*xChunk)(span))
		}
	}

}

func TestNewXAllocator3(t *testing.T) {
	h, err := newXHeap()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10000; i++ {
		ptr, err := h.chunkAllocator.alloc()
		if err != nil {
			t.Fatal(err)
		}
		a := (*xChunk)(ptr)
		a.npages = uintptr(i)
		a.startAddr = uintptr(i)
		if err := h.addChunks([]*xChunk{a}); err != nil {
			t.Fatal(err)
		}
	}
	type SL struct {
		addr uintptr
		len  int
		cap  int
	}
	sl := *(*SL)(unsafe.Pointer(&h.allChunk))
	fmt.Println("------------", sl.addr)
	for i := 0; i < 10000; i++ {
		if h.allChunk[i].npages != uintptr(i) {
			t.Fatalf("%d   %+v\n", i, h.allChunk[i])
		}
	}
}

func TestHeap(t *testing.T) {
	h, err := newXHeap()
	if err != nil {
		t.Fatal(err)
	}
	c, _ := h.allocRawSpan(1)
	for i := 0; i < pagesPerRawMemory; i++ {
		c2, err := h.allocRawSpan(1)
		if err != nil {
			t.Fatal(err)
		}
		if c2.startAddr-c.startAddr != _PageSize*uintptr(i+1) {
			t.Errorf("c2:%+v  c:%+v", c2, c)
		}
	}
}

type User struct {
	Name int
	Age  int
	Addr string
	Desc string
}

func Test_xSpanPool(t *testing.T) {
	// 同步扩容 680  异步扩容  551
	h, err := newXHeap()
	if err != nil {
		t.Fatal(err)
	}
	sp, err := newXSpanPool(h, 0.6)
	if err != nil {
		t.Fatal(err)
	}
	sp.debug = true
	var us []unsafe.Pointer
	size := unsafe.Sizeof(User{})
	for i := 0; i < 100000; i++ {
		p, err := sp.Alloc(size)
		if err != nil {
			t.Fatal(err)
		}
		user := (*User)(p)
		user.Age = i
		user.Name = rand.Int()
		us = append(us, p)
	}

	for i, pointer := range us {
		if sss := (*User)(pointer); sss.Age != i {
			t.Fatalf("%+v\n", (*xSpan)(pointer))
		} else {
			fmt.Printf("User:%+v\n", sss)
		}
	}
}

func Test_xSpanPool2(t *testing.T) {
	// 同步扩容 680  异步扩容  551
	fmt.Println(100000 / (4096 / 48))
	stringHeader, err := newXFixedAllocator(unsafe.Sizeof(reflect.StringHeader{}), func(inuse uintptr) {
		fmt.Println(inuse)
	})
	if err != nil {
		t.Fatal(err)
	}
	a, err := newXSliceAllocator(1, 1, func(inuse uintptr) {
		fmt.Println(inuse)
	})
	if err != nil {
		t.Fatal(err)
	}
	h, err := newXHeap()
	if err != nil {
		t.Fatal(err)
	}
	sp, err := newXSpanPool(h, 0.6)
	if err != nil {
		t.Fatal(err)
	}
	sp.debug = true
	var lock sync.Mutex
	us := make([]unsafe.Pointer, 100000)
	var waiter sync.WaitGroup
	waiter.Add(100000)
	for i := 0; i < 100000; i++ {
		func(i int) {
			lock.Lock()
			p, err := sp.Alloc(unsafe.Sizeof(User{}))
			if err != nil {
				t.Fatal(err)
			}
			user := (*User)(p)
			user.Age = i
			user.Name = rand.Int()
			sss := fmt.Sprintf("xian（西安）_%d", rand.Int())
			stringP := (*reflect.StringHeader)(unsafe.Pointer(&sss))
			if i == 7418 {
				fmt.Println("sssss")
			}
			stringP2, _, err := a.batchAlloc(uintptr(stringP.Len), 0)
			if err != nil {
				t.Fatal(err)
			}
			dst := reflect.SliceHeader{
				Data: uintptr(stringP2),
				Len:  stringP.Len,
				Cap:  stringP.Len,
			}
			dstbytes := *(*[]byte)(unsafe.Pointer(&dst))

			source := reflect.SliceHeader{
				Data: (stringP.Data),
				Len:  stringP.Len,
				Cap:  stringP.Len,
			}
			sourcebyte := *(*[]byte)(unsafe.Pointer(&source))
			copy(dstbytes, sourcebyte)
			sh, err := stringHeader.alloc()
			if err != nil {
				t.Fatal(err)
			}
			resu := (*reflect.StringHeader)(sh)
			resu.Data = uintptr(stringP2)
			resu.Len = stringP.Len
			vv := *(*string)(sh)
			user.Addr = vv
			fmt.Println(i, resu, vv)
			us[i] = p

			lock.Unlock()
			waiter.Done()
		}(i)
	}

	waiter.Wait()
	for i, pointer := range us {
		if sss := (*User)(pointer); sss.Age != int(i) {
			t.Fatalf("%d %+v\n", i, (*User)(pointer))
		} else {
			fmt.Println((*User)(pointer).Addr)
		}
	}
}

func BenchmarkRBTree_ClearxSpanPool(b *testing.B) {
	h, err := newXHeap()
	if err != nil {
		b.Fatal(err)
	}
	sp, err := newXSpanPool(h, 0.6)
	if err != nil {
		b.Fatal(err)
	}
	us := make([]unsafe.Pointer, 100000)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			p, err := sp.Alloc(unsafe.Sizeof(User{}))
			if err != nil {
				b.Fatal(err)
			}
			i := rand.Int() % 100000
			user := (*User)(p)
			user.Age = i
			user.Name = rand.Int()
			us[i] = p
			fmt.Println(i, user, p)
		}
		for i, pointer := range us {
			if pointer == nil {
				continue
			}
			if sss := (*User)(pointer); sss.Age != int(i) {
				b.Fatalf("%d %+v\n", i, (*User)(pointer))
			} else {
				fmt.Println((*User)(pointer))
			}
		}
	})
}

//数组、string(序列化后内容)、slice需要知道其长度。别的都不需要知道，固定的。
//释放内容的时候需要传入大小和offset
func TestString(t *testing.T) {
	h, err := newXHeap()
	if err != nil {
		t.Fatal(err)
	}
	sp, err := newXSpanPool(h, 0.6)
	if err != nil {
		t.Fatal(err)
	}
	sa := newXStringAllocator(sp)
	for i := 0; i < 1000000; i++ {
		p, err := sa.From(fmt.Sprintf("hjshdjhjshdjhdj%d", rand.Int()))
		if err != nil {
			t.Fatal(err)
		}
		/*if pre > 0 && uintptr(p)-pre != 128 {
			t.Log("重新分配了一个span地址")
		} else {
			t.Log("同一个span中")
		}
		pre = uintptr(unsafe.Pointer(p))
		v := *(p)*/
		fmt.Println(p)
	}
}

func TestUser(t *testing.T) {
	h, err := newXHeap()
	if err != nil {
		t.Fatal(err)
	}
	sp, err := newXSpanPool(h, 0.6)
	if err != nil {
		t.Fatal(err)
	}
	sa := newXStringAllocator(sp)
	us := make([]*User, 100000)
	for i := 0; i < 100000; i++ {
		p, err := sp.Alloc(unsafe.Sizeof(User{}))
		if err != nil {
			t.Fatal(err)
		}
		user := (*User)(p)
		user.Age = i
		user.Name = rand.Int()
		strPtr, err := sa.From(fmt.Sprintf("chaoyang_北京_%d", rand.Int()))
		if err != nil {
			t.Fatal(err)
		}
		user.Addr = strPtr
		us[i] = user
	}

	for i, user := range us {
		fmt.Println(i, user)
	}
}

func TestXTreap(t *testing.T) {
	valAllocator := newXAllocator(unsafe.Sizeof(treapNode{}))
	freeChunks := newXTreap(valAllocator)
	for i := 1; i < 1000000; i++ {
		err := freeChunks.insert(&xChunk{
			startAddr: uintptr(i),
			npages:    uintptr(i),
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	for i := 0; i < 3; i++ {
		for i := 1; i < 1000000; i++ {
			node, err := freeChunks.find(uintptr(i))
			if err != nil {
				t.Fatal(err)
			}
			freeChunks.removeNode(node)
			node.chunk.startAddr += 1
			node.chunk.npages -= 1
			freeChunks.insert(node.chunk)
		}
	}
}

func Test_NewUser(t *testing.T) {
	t1 := time.Now()
	us := make([]*User, 10000000)
	key, val := "key", "val"
	for i := 0; i < 10000000; i++ {
		user := User{}
		user.Addr = key
		user.Desc = val
		us[i] = &user
	}
	fmt.Println(time.Now().Sub(t1))
}

func Test_FromInAddr(t *testing.T) {
	f := &Factory{}
	mm, err := f.CreateConcurrentHashMapMemory(0.6, 1)
	if err != nil {
		t.Fatal(err)
	}
	key, val := "key", "val"
	uSize := unsafe.Sizeof(User{})
	t1 := time.Now()
	for i := 0; i < 10000000; i++ {
		entryPtr, err := mm.Alloc(uSize + uintptr(len(key)) + uintptr(len(val)))
		if err != nil {
			t.Fatal(err)
		}
		_ = (*User)(entryPtr)
		/*
			keyValPtrs, err := mm.FromInAddr(uintptr(entryPtr)+uSize, key, val)
			if err != nil {
				t.Fatal(err)
			}
			user.Addr = *(keyValPtrs[0])
			user.Desc = *(keyValPtrs[1])
		*/
		/*
			user.Addr = key
			user.Desc = val
		*/
	}
	fmt.Println(time.Now().Sub(t1))
}

func TestMm_Alloc(t *testing.T) {
	f := &Factory{}
	mm, err := f.CreateMemory(0.6)
	if err != nil {
		panic(err)
	}
	uSize := unsafe.Sizeof(User{})
	t1 := time.Now()
	us := make([]*User, 10000000)
	var wait sync.WaitGroup
	wait.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wait.Done()
			for i := 0; i < 1000000; i++ {
				entryPtr, err := mm.Alloc(uSize)
				if err != nil {
					panic(err)
				}
				user := (*User)(entryPtr)
				iii := strconv.Itoa(i)
				a, err := mm.From(iii)
				if err != nil {
					t.Error(err)
				}
				f1(user, a, a)
				us[i] = user
			}
		}()
	}
	wait.Wait()
	fmt.Println(time.Now().Sub(t1), len(us))
}

func f1(user *User, key, val string) {
	user.Addr = key
	user.Desc = val
}

func TestMm_GoAlloc(t *testing.T) {
	t1 := time.Now()
	us := make([]*User, 10000000)
	var wait sync.WaitGroup
	wait.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wait.Done()
			for i := 0; i < 1000000; i++ {
				user := &User{}
				iii := strconv.Itoa(i)
				key, val := iii, iii
				f1(user, key, val)
				us[i] = user
			}
		}()
	}

	fmt.Println(time.Now().Sub(t1))
}

func Test_String(t *testing.T) {
	us := make([]string, 10000000)
	i := []byte(cast.ToString("n"))

	for n := 0; n < 8000000; n++ {
		j := string(i)
		us[n] = j
	}
}

func TestMm_Free(t *testing.T) {
	h, err := newXHeap()
	if err != nil {
		t.Fatal(err)
	}
	sp, err := newXSpanPool(h, 0.6)
	if err != nil {
		t.Fatal(err)
	}
	sp.debug = true
	var us []unsafe.Pointer
	size := unsafe.Sizeof(User{})
	for i := 0; i < 1000; i++ {
		if i%85 == 0 && i > 0 {
			//panic清空前85个
			for j := 0; j < 85; j++ {
				if err := sp.Free(uintptr(us[j])); err != nil {
					t.Fatal(err)
				}
			}
		}
		p, err := sp.Alloc(size)
		if err != nil {
			t.Fatal(err)
		}
		user := (*User)(p)
		user.Age = i
		user.Name = rand.Int()
		us = append(us, p)
	}
}

func TestMm_Free2(t *testing.T) {
	h, err := newXHeap()
	if err != nil {
		t.Fatal(err)
	}
	sp, err := newXSpanPool(h, 0.6)
	if err != nil {
		t.Fatal(err)
	}
	sp.debug = true
	var us []unsafe.Pointer
	size := unsafe.Sizeof(User{})
	for i := 0; i < 20000; i++ {
		if i == 12000 {
			//删除前600个偶数对象
			for j := 0; j < 6000; j += 2 {
				if err := sp.Free(uintptr(us[j])); err != nil {
					t.Fatal(err)
				}
				us[j] = nil
			}
		}
		p, err := sp.Alloc(size)
		if err != nil {
			t.Fatal(err)
		}
		user := (*User)(p)
		user.Age = i
		user.Name = rand.Int()
		us = append(us, p)
	}

	for i, pointer := range us {
		if pointer == nil {
			continue
		}
		if sss := (*User)(pointer); sss.Age != i {
			t.Fatalf("%+v\n", (*User)(pointer))
		}
	}
}

func TestMm2(t *testing.T) {
	f := Factory{}
	mm, err := f.CreateMemory(0.6)
	if err != nil {
		t.Fatal(err)
	}

	uSize := unsafe.Sizeof(User{})
	//t1 := time.Now()
	us := make(chan uintptr, 1000000)
	var wait sync.WaitGroup
	wait.Add(10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			defer wait.Done()
			for j := 0; j < 100000; j++ {
				age := i*100000 + j
				entryPtr, err := mm.Alloc(uSize)
				if err != nil {
					t.Fatal(err)
				}
				user := (*User)(entryPtr)
				user.Age = age
				us <- uintptr(entryPtr)
			}
		}(i)
	}
	wait.Wait()

	wait = sync.WaitGroup{}
	wait.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wait.Done()
		l:
			for {
				select {
				case uPtr := <-us:
					user := (*User)(unsafe.Pointer(uPtr))
					if user.Age%2 == 0 {
						continue
					}
					if err := mm.Free(uPtr); err != nil {
						t.Fatal(err)
					}
				case <-time.After(time.Second * 30):
					break l
				}
			}
		}()
	}
	wait.Wait()

	wait = sync.WaitGroup{}
	wait.Add(10)
	us3 := make(chan uintptr, 1000000)
	for i := 0; i < 10; i++ {
		go func(i int) {
			defer wait.Done()
			for j := 0; j < 100000; j++ {
				age := i*100000 + j
				entryPtr, err := mm.Alloc(uSize)
				if err != nil {
					t.Fatal(err)
				}
				user := (*User)(entryPtr)
				user.Age = age
				us3 <- uintptr(entryPtr)
			}
		}(i)
	}
	wait.Wait()
	fmt.Println("分配完成")

l2:
	for {
		select {
		case uPtr := <-us3:
			user := (*User)(unsafe.Pointer(uPtr))
			if user.Age == 0 {
				t.Error(user)
			}
		case <-time.After(time.Second * 10):
			break l2
		}
	}
}

func TestRawAlloc2(t *testing.T) {
	f := Factory{}
	mm, err := f.CreateMemory(0.6)
	if err != nil {
		t.Fatal(err)
	}
	p, err := mm.RawAlloc(536871918/_PageSize + 1)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(p.StartAddr, p.Npages)
}

func Init() {
	// 略
	runtime.GOMAXPROCS(6)              // 限制 CPU 使用数，避免过载
	runtime.SetMutexProfileFraction(1) // 开启对锁调用的跟踪
	runtime.SetBlockProfileRate(1)     // 开启对阻塞操作的跟踪

	go func() {
		// 启动一个 http server，注意 pprof 相关的 handler 已经自动注册过了
		if err := http.ListenAndServe(":6060", nil); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	<-time.After(time.Second * 10)
}

func TestAlloc_Benchmark(t *testing.T) {
	//	Init()
	f := &Factory{}
	mm, err := f.CreateMemory(0.75)
	if err != nil {
		t.Fatal(err)
	}
	var wait sync.WaitGroup
	wait.Add(10)
	now := time.Now()
	for j := 0; j < 10; j++ {
		go func(z int) {
			defer wait.Done()
			for i := 0; i < 1000000; i++ {
				n := rand.Intn(60) + 1
				if _, err := mm.Alloc(uintptr(n)); err != nil {
					t.Error(err)
				}
			}
		}(j)
	}
	wait.Wait()
	fmt.Println(time.Now().Sub(now), 10*1000000/1000000, "百万") //300w ops
}
