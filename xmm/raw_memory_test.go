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
	"syscall"
	"testing"
	"time"
	"unsafe"
)

func TestSet(t *testing.T) {
	var first, last unsafe.Pointer
	for i := 0; i < 10000; i++ {
		ptr, err := pool.alloc(_PageSize >> 3)
		if err != nil {
			t.Fatal(err)
		}
		if first == nil {
			fmt.Printf("first: %d\n", ptr)
			first = ptr
		}
		last = ptr
	}
	fmt.Printf("last: %d   \n  ", metadataRawMemoryBytes/_PageSize)
	num := uintptr(last) - uintptr(first)
	if num != metadataRawMemoryBytes {
		t.Fatal("errr")
	}

}

func Test_Syscall6(t *testing.T) {
	/*arenaSizes := []uintptr{
		512 << 20,
		256 << 20,
		128 << 20,
	}
	for _, arenaSize := range arenaSizes {
		a, size := sysReserveAligned(unsafe.Pointer(p), arenaSize, heapArenaBytes)
		if a != nil {
			mheap_.arena.init(uintptr(a), size)
			p = uintptr(a) + size // For hint below
			break
		}
	}*/
	size := int(round(23232, 4096))
	addr := uintptr(sysReserve(size)) // 必须预留的多才可以   	arena linearAlloc学习这个
	for i := 0; i < 8; i++ {
		prot, flags, fd, offset, length := syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_FIXED|syscall.MAP_PRIVATE, -1, 0, size
		r0, _, e1 := syscall.Syscall6(syscall.SYS_MMAP, uintptr(addr), uintptr(length), uintptr(prot), uintptr(flags), uintptr(fd), uintptr(offset))
		ret := r0
		var err error
		if e1 != 0 {
			err = errnoErr(e1)
		}
		if err != nil {
			panic(err)
		}
		//fmt.Println(ret, err)
		user := (*User)(unsafe.Pointer(ret))
		user.Age = 11
		user.Name = 13
		addr += uintptr(length)
		fmt.Printf("user:%+v addr:%d\n", (*User)(unsafe.Pointer(ret)), ret)
	}
}

func Test_LineAlloc(t *testing.T) {
	userSize, num := unsafe.Sizeof(User{}), 1000000000
	fmt.Println(heapRawMemoryBytes*8, unsafe.Sizeof(User{})*1000000000)
	size := int(round(uintptr(heapRawMemoryBytes*8), 4096))
	addr := uintptr(sysReserve(size)) // 必须预留的多才可以   	arena linearAlloc学习这个
	prot, flags, fd, offset, length := syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_FIXED|syscall.MAP_PRIVATE, -1, 0, size
	r0, _, e1 := syscall.Syscall6(syscall.SYS_MMAP, uintptr(addr), uintptr(length), uintptr(prot), uintptr(flags), uintptr(fd), uintptr(offset))
	ret := r0
	var err error
	if e1 != 0 {
		err = errnoErr(e1)
	}
	if err != nil {
		panic(err)
	}
	for i := 0; i < num; i++ {
		user := (*User)(unsafe.Pointer(ret))
		user.Age = 11
		user.Name = 13
		//addr += uintptr(length)
		ret += userSize
		//fmt.Printf("user:%+v addr:%d\n", (*User)(unsafe.Pointer(ret)), ret)
	}
}

func Test_MMapAlloc(t *testing.T) {
	arenaSizes := []uintptr{
		512 << 20,
		256 << 20,
		128 << 20,
	}
	for i, size := range arenaSizes {
		fmt.Println(i, size)
	}

	userSize, num := unsafe.Sizeof(User{}), 1000000000
	size := int(round(userSize*uintptr(num), 4096))
	mem, err := syscall.Mmap(-1, 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON /*|syscall.MAP_FIXED*/ |syscall.MAP_PRIVATE)
	if err != nil {
		t.Fatal(err)
	}
	ptr := unsafe.Pointer(&mem[0])
	ret := uintptr(ptr)
	if err != nil {
		panic(err)
	}
	for i := 0; i < num; i++ {
		user := (*User)(unsafe.Pointer(ret))
		user.Age = 11
		user.Name = 13
		//addr += uintptr(length)
		ret += userSize
		//fmt.Printf("user:%+v addr:%d\n", (*User)(unsafe.Pointer(ret)), ret)
	}
}

func Test_GoAlloc(t *testing.T) {
	t1 := time.Now()
	us := make([]*User, 1000000000)
	for i := 0; i < 1000000000; i++ {
		user := &User{}
		user.Age = 11
		user.Name = 13
		us[i] = user
	}
	fmt.Println(time.Now().Sub(t1))
}

func errnoErr(e syscall.Errno) error {
	switch e {
	case 0:
		return nil
	case syscall.EAGAIN:
		return errEAGAIN
	case syscall.EINVAL:
		return errEINVAL
	case syscall.ENOENT:
		return errENOENT
	}
	return e
}

func sysReserve(size int) unsafe.Pointer {
	/*if raceenabled && GOOS == "darwin" {
		// Currently the race detector expects memory to live within a certain
		// range, and on Darwin 10.10 mmap is prone to ignoring hints, moreso
		// than later versions and other BSDs (#26475). So, even though it's
		// potentially dangerous to MAP_FIXED, we do it in the race detection
		// case because it'll help maintain the race detector's invariants.
		//
		// TODO(mknyszek): Drop this once support for Darwin 10.10 is dropped,
		// and reconsider this when #24133 is addressed.
		flags |= _MAP_FIXED
	}*/
	mem, err := syscall.Mmap(-1, 0, size, syscall.PROT_NONE, syscall.MAP_ANON|syscall.MAP_PRIVATE)
	if err != nil {
		panic(err)
	}
	ptr := unsafe.Pointer(&mem[0])
	return ptr
}
