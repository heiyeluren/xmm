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
	"errors"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"
)

//用来管理mmap申请的内存，用于实际存放地址的元数据
type xRawMemory struct {
	addr uintptr
	down bool
	next *xRawMemory
	mem  []byte
}

func newXRawMemory(byteNum int) (*xRawMemory, error) {

	mem, err := syscall.Mmap(-1, 0, byteNum, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_PRIVATE)
	if err != nil {
		return nil, err
	}
	//log.Printf(" newXRawMemory(mmap) byteNum:%d byte \n", byteNum)
	ptr := unsafe.Pointer(&mem[0])
	xrm := &xRawMemory{addr: uintptr(ptr), mem: mem}
	return xrm, nil
}

func (xrm *xRawMemory) Close() error {
	return syscall.Munmap(xrm.mem)
}

type block struct {
	addr uintptr
	len  uintptr
	end  uintptr
}

var pool *xRawMemoryPool

func init() {
	pool = &xRawMemoryPool{}
}

//
type xRawMemoryPool struct {
	//空闲free
	frees []*block
	//申请内存，头插法
	xrm   *xRawMemory
	index uintptr
	lock  sync.RWMutex
}

// alloc 页对齐
func (xrmp *xRawMemoryPool) alloc(byteSize uintptr) (ptr unsafe.Pointer, err error) {
	xrmp.lock.Lock()
	defer xrmp.lock.Unlock()
	size := byteSize
	offset, err := xrmp.alignOf(size)
	if err != nil {
		return nil, err
	}
	if offset == 0 {
		//扩容
		xrm, err := newXRawMemory(metadataRawMemoryBytes)
		if err != nil {
			return nil, err
		}
		next := xrmp.xrm
		xrm.next = next
		xrmp.xrm = xrm
	}
	return unsafe.Pointer(xrmp.xrm.addr + offset), nil
}

func (xrmp *xRawMemoryPool) release(block *block) error {
	xrmp.lock.Lock()
	defer xrmp.lock.Unlock()
	xrmp.frees = append(xrmp.frees, block)
	return nil
}

func (xrmp *xRawMemoryPool) grow() error {
	xrmp.lock.Lock()
	defer xrmp.lock.Unlock()
	xrm, err := newXRawMemory(metadataRawMemoryBytes)
	if err != nil {
		return err
	}
	next := xrmp.xrm
	xrm.next = next
	xrmp.xrm = xrm
	return nil
}

//通过cas获取空闲的offset
func (xrmp *xRawMemoryPool) freeOffset(size uintptr) (uintptr, error) {
	if size > metadataRawMemoryBytes {
		return 0, errors.New("size is over")
	}
	var swapped bool
	for /*retry := 3; retry > 0; retry--*/ {
		oldVal := atomic.LoadUintptr(&xrmp.index)
		offset := oldVal % metadataRawMemoryBytes
		if _PageSize-offset%_PageSize < size {
			// 当前page不够，挪动到下一个page
			offset = Align(offset, _PageSize)
		}
		index := offset + size
		// 当前xRawMemory 不够，将创建一个RawMemory
		if index > metadataRawMemoryBytes {
			index = size
			offset = 0
		}
		swapped = atomic.CompareAndSwapUintptr(&xrmp.index, oldVal, index)
		if swapped {
			return offset, nil
		}
	}
}

func (xrmp *xRawMemoryPool) alignOf(size uintptr) (uintptr, error) {
	if size > metadataRawMemoryBytes {
		return 0, errors.New("size is over[xRawMemoryPool]")
	}
	offset := xrmp.index % metadataRawMemoryBytes
	if _PageSize-offset%_PageSize < size {
		// 当前page不够，挪动到下一个page
		offset = Align(offset, _PageSize)
	}
	xrmp.index = offset + size
	// 当前xRawMemory 不够，将创建一个RawMemory
	if xrmp.index > metadataRawMemoryBytes {
		xrmp.index = size
		offset = 0
	}
	return offset, nil
}
