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
	"unsafe"
)

// persistentChunkSize is the number of bytes we allocate when we grow
// a persistentAlloc.
const metadataRawMemoryBytes = 256 << 20

// 申请固定大小申请
type xFixedAllocator struct {

	//固定的
	size uintptr

	addr uintptr

	//所有mmap的内存
	free *mRawlink

	//使用的
	chunk uintptr

	//未用的
	nchunk uintptr

	//mmap的内存
	freeRawMemory *xRawMemory

	growCall func(inuse uintptr)

	inuse uintptr

	lock sync.Mutex
}

func newXFixedAllocator(size uintptr, growCall func(inuse uintptr)) (*xFixedAllocator, error) {
	if size > metadataRawMemoryBytes {
		return nil, errors.New("size 超过了最大newXAllocator freeRawMemory 内存")
	}
	xrm, err := newXRawMemory(metadataRawMemoryBytes)
	if err != nil {
		return nil, err
	}
	return &xFixedAllocator{size: size, freeRawMemory: xrm, chunk: 0, nchunk: metadataRawMemoryBytes, growCall: growCall}, nil
}

type mRawlink struct {
	next *mRawlink
}

func (xa *xFixedAllocator) alloc() (unsafe.Pointer, error) {
	nchunk := xa.casNchunk()
	if nchunk < xa.size {
		if err := xa.grow(); err != nil {
			return nil, err
		}
	}
	chunk := xa.casChunk()
	offset := atomic.LoadUintptr(&xa.freeRawMemory.addr) + chunk
	xa.inuse += xa.size
	return unsafe.Pointer(offset), nil
}

func (xa *xFixedAllocator) grow() error {
	xa.lock.Lock()
	defer xa.lock.Unlock()
	if xa.nchunk >= xa.size {
		return nil
	}
	// todo 原子化
	if xa.growCall != nil {
		xa.growCall(xa.inuse)
	}
	xrm, err := newXRawMemory(metadataRawMemoryBytes)
	if err != nil {
		return err
	}
	xa.freeRawMemory = xrm
	xa.chunk = 0
	xa.nchunk = metadataRawMemoryBytes
	return nil
}

func (xa *xFixedAllocator) casNchunk() (old uintptr) {
	size := xa.size //不变的size
	var swapped bool
	for /*retry := 3; retry > 0; retry--*/ {
		oldVal := atomic.LoadUintptr(&xa.nchunk)
		newVal := oldVal - size
		swapped = atomic.CompareAndSwapUintptr(&xa.nchunk, oldVal, newVal)
		if swapped {
			return oldVal
		}
	}
}

func (xa *xFixedAllocator) casChunk() (old uintptr) {
	size := xa.size //不变的size
	var swapped bool
	for /*retry := 3; retry > 0; retry--*/ {
		oldVal := atomic.LoadUintptr(&xa.chunk)
		newVal := oldVal + size
		swapped = atomic.CompareAndSwapUintptr(&xa.chunk, oldVal, newVal)
		if swapped {
			return oldVal
		}
	}
}

func (xa *xFixedAllocator) Close() error {
	return xa.freeRawMemory.Close()
}

type xSliceAllocator struct {
	*xFixedAllocator
	initSize uintptr
	lock     sync.Mutex
}

func newXSliceAllocator(elementSize uintptr, initSize uintptr, growCall func(inuse uintptr)) (*xSliceAllocator, error) {
	a, err := newXFixedAllocator(elementSize, growCall)
	if err != nil {
		return nil, err
	}
	if initSize < metadataRawMemoryBytes/elementSize {
		initSize = metadataRawMemoryBytes / elementSize
	}
	return &xSliceAllocator{initSize: initSize, xFixedAllocator: a}, nil
}

func (xa *xSliceAllocator) batchAlloc(size, cap uintptr) (ptr unsafe.Pointer, old *xRawMemory, err error) {
	xa.lock.Lock()
	defer xa.lock.Unlock()
	if xa.nchunk < size*xa.size {
		if xa.growCall != nil {
			xa.growCall(xa.inuse)
		}
		if cap < xa.initSize {
			cap = xa.initSize << 1
			xa.initSize = cap
		}
		if cap < 1 {
			return nil, nil, errors.New("cap == 0")
		}
		nc := int(cap * xa.size)
		xrm, err := newXRawMemory(nc)
		if err != nil {
			return nil, nil, err
		}
		old = xa.freeRawMemory
		xa.freeRawMemory = xrm
		xa.chunk = 0
		xa.nchunk = uintptr(nc)
	}
	offset := xa.freeRawMemory.addr + xa.chunk
	xa.chunk += size * xa.size
	xa.nchunk -= size * xa.size
	xa.inuse += size * xa.size
	return unsafe.Pointer(offset), nil, nil
}

func (xa *xSliceAllocator) alloc(cap uintptr) (ptr unsafe.Pointer, old *xRawMemory, err error) {
	xa.lock.Lock()
	defer xa.lock.Unlock()
	if xa.nchunk < xa.size {
		xa.growCall(xa.inuse)
		nc := int(cap * xa.size)
		xrm, err := newXRawMemory(nc)
		if err != nil {
			return nil, nil, err
		}
		old = xa.freeRawMemory
		xa.freeRawMemory = xrm
		xa.chunk = 0
		xa.nchunk = uintptr(nc)
	}
	offset := xa.freeRawMemory.addr + xa.chunk
	xa.chunk += xa.size
	xa.nchunk -= xa.size
	xa.inuse += xa.size
	return unsafe.Pointer(offset), nil, nil
}

func (xa *xSliceAllocator) grow(newCap uintptr, copy func(new unsafe.Pointer) error) error {
	if newCap < xa.initSize {
		newCap = xa.initSize
	}
	ptr, old, err := xa.alloc(newCap)
	if err != nil {
		return err
	}
	if err := copy(ptr); err != nil {
		xa.freeRawMemory.Close()
		xa.freeRawMemory = old
		return err
	}
	if old != nil {
		if err := old.Close(); err != nil {
			return err
		}
	}
	return nil
}

type xAllocator struct {
	size  uintptr
	inuse uintptr
}

func newXAllocator(size uintptr) *xAllocator {
	return &xAllocator{size: size}
}

func (xa *xAllocator) alloc() (unsafe.Pointer, error) {
	xa.inuse += xa.size
	return pool.alloc(xa.size)
}
