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
	"testing"
	"time"
	"unsafe"
	//"xmm/src"
)

func TestLinearAlloc(t *testing.T) {
	var la linearAlloc
	if er := la.init(heapRawMemoryBytes * 10); er != nil {
		t.Fatal(er)
	}
	userSize, num := unsafe.Sizeof(User{}), 10000
	p, err := la.alloc(userSize*uintptr(num), 4096)
	if err != nil {
		t.Fatal(err)
	}
	ret := uintptr(p)
	us := make([]uintptr, num)

	for n := 0; n < num; n++ {
		user := (*User)(unsafe.Pointer(ret))
		user.Age = 11
		user.Name = n
		//addr += uintptr(length)
		us[n] = ret
		ret += userSize
	}
	for i, u := range us {
		user := (*User)(unsafe.Pointer(u))
		if user == nil || user.Age != 11 || user.Name != i {
			t.Error(i, user)
		}
	}
}

func TestLinearAlloc2(t *testing.T) {
	var la linearAlloc
	if er := la.init(heapRawMemoryBytes * 10); er != nil {
		t.Fatal(er)
	}
	p, err := la.alloc(heapRawMemoryBytes, 4096)
	if err != nil {
		t.Fatal(err)
	}
	node := (*entry.NodeEntry)(p)

	var wait sync.WaitGroup
	wait.Add(10)
	for j := 0; j < 10; j++ {
		go func(z int) {
			defer wait.Done()
			for i := 0; i < 800000; i++ {
				node.Key = []byte("keyPtr")
				node.Value = []byte("valPtr")
				node.Hash = 12121
			}
		}(j)
	}
	wait.Wait()
}

type A struct {
	a byte
	b int
	c bool
}

func TestLinearAlloc3(t *testing.T) {
	pageSize := uintptr(4096)
	var la linearAlloc
	if er := la.expand(nil, heapRawMemoryBytes); er != nil {
		t.Fatal(er)
	}

	p, err := la.alloc(heapRawMemoryBytes, pageSize)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(unsafe.Sizeof(entry.NodeEntry{}), uintptr(p), round(uintptr(p), pageSize), unsafe.Offsetof(entry.NodeEntry{}.Next))
	fmt.Println(round(unsafe.Sizeof(entry.NodeEntry{}), 8), round(unsafe.Sizeof(A{}), 8))

	//todo 怀疑是这个内存没有对齐     -120 cache line 0.01s    -80 0.09s     -20 0.22s
	//todo 对齐
	ptr := uintptr(p) + pageSize*101 - 1500
	fmt.Println(ptr, ptr%8, round(ptr, pageSize), round(ptr+80, pageSize))
	node := (*entry.NodeEntry)(unsafe.Pointer(ptr))
	var wait sync.WaitGroup
	wait.Add(10)
	t11 := time.Now()
	for j := 0; j < 10; j++ {
		go func(z int) {
			defer wait.Done()
			for i := 0; i < 800000; i++ {
				node.Key = []byte("keyPtr")
				node.Value = []byte("valPtr")
				node.Hash = 12121
			}
		}(j)
	}
	wait.Wait()
	fmt.Println(time.Now().Sub(t11), uintptr(unsafe.Pointer(&node.Value)))
}

func TestLinearCopyAlloc4(t *testing.T) {
	pageSize := uintptr(4096)
	var la linearAlloc
	if er := la.expand(nil, heapRawMemoryBytes); er != nil {
		t.Fatal(er)
	}

	p, err := la.alloc(heapRawMemoryBytes, pageSize)
	if err != nil {
		t.Fatal(err)
	}
	ptr := uintptr(p) + pageSize*101 - 1500
	fmt.Println(float64(pageSize*101)*0.1/8.0, ptr)

}
