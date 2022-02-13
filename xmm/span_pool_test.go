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
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"
)

func TestPoolAlloc(t *testing.T) {
	h, err := newXHeap()
	if err != nil {
		t.Fatal(err)
	}
	sp, err := newXSpanPool(h, 0.75)
	if err != nil {
		t.Fatal(err)
	}
	sp.debug = true
	var wait sync.WaitGroup
	wait.Add(20)
	for i := 0; i < 20; i++ {
		go func() {
			defer wait.Done()
			for i := 0; i < 1000; i++ {
				sp.Alloc(4)
			}
		}()
	}
	wait.Wait()
}

func TestSpanLock(t *testing.T) {
	h, err := newXHeap()
	if err != nil {
		t.Fatal(err)
	}
	sp, err := newXSpanPool(h, 0.75)
	if err != nil {
		t.Fatal(err)
	}
	sp.debug = true
	cs, err := sp.allocClassSpan(1)
	if err != nil {
		t.Fatal(err)
	}
	var wait sync.WaitGroup
	cnt := _PageSize / 4
	allcCnt := uintptr(20 * (cnt / 20))
	wait.Add(20)
	for i := 0; i < 20; i++ {
		go func() {
			defer wait.Done()
			for j := 0; j < cnt/20; j++ {
				cs.getFree(4)
			}
		}()
	}
	wait.Wait()
	fmt.Printf("%+v \nfreeIndex:%d %d %d \n", cs, cs.freeIndex, allcCnt*4, allcCnt)
	//todo 问题,好像自己实现的锁有问题，应该是CAS 和 悲观锁冲突了。只能选一个。cas和锁必须互斥
	// 锁里面有CAS，是否考虑用锁呢？
	<-time.After(time.Second)
	if val := atomic.LoadUintptr(&cs.freeIndex); val != allcCnt*4 {
		panic(val)
	}
}

func Test_SanPool_Alloc(t *testing.T) {
	f := &Factory{}
	mm, err := f.CreateConcurrentHashMapMemory(0.6, 1)
	if err != nil {
		panic(err)
	}
	key, val := "key", "val"
	uSize := unsafe.Sizeof(User{})
	t1 := time.Now()
	us := make([]*User, 1000000)

	for i := 0; i < 1000000; i++ {
		entryPtr, err := mm.Alloc(uSize)
		if err != nil {
			panic(err)
		}
		user := (*User)(entryPtr)
		f1(user, key, val)
		us[i] = user
	}
	fmt.Println(time.Now().Sub(t1), len(us))
	for _, u := range us {
		if u.Addr != "key" {
			panic(u)
		}
	}
}

func Test_SanPool_Find(t *testing.T) {
	h, err := newXHeap()
	if err != nil {
		t.Fatal(err)
	}
	sp, err := newXSpanPool(h, 0.75)
	if err != nil {
		t.Fatal(err)
	}
	key, val := "key", "val"
	uSize := unsafe.Sizeof(User{})
	usPtr := make([]uintptr, 100000)
	us := make([]*User, 100000)

	for i := 0; i < 100000; i++ {
		entryPtr, err := sp.Alloc(uSize)
		if err != nil {
			panic(err)
		}
		user := (*User)(entryPtr)
		f1(user, key, val)
		usPtr[i] = uintptr(entryPtr)
		us[i] = user
	}
	for i := 0; i < 50000; i++ {
		u := usPtr[i]
		if err := h.free(u); err != nil {
			t.Fatal(i, err)
		}
	}
	fmt.Println("第一次清空结束")

	for i := 0; i < 50000; i++ {
		u := usPtr[i]
		if err := h.free(u); err != nil {
			t.Fatal(i, err)
		}
	}
	fmt.Println("第二次清空结束")
	for i := 50000; i < 100000; i++ {
		u := usPtr[i]
		if err := h.free(u); err != nil {
			t.Fatal(i, err)
		}
	}

	fmt.Println("大span清空结束")

	size := uintptr(_MaxSmallSize + 12)
	p, err := sp.Alloc(size)
	if err != nil {
		panic(err)
	}
	if err := h.free(uintptr(p)); err != nil {
		t.Fatal(err)
	}
	if err := h.free(uintptr(p)); err != nil {
		t.Fatal(err)
	}
}
