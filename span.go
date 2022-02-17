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
	"sync"
	"sync/atomic"
	"unsafe"
)

// 写无锁atomic,扩容必须得全局锁
type xSpan struct {
	lock           sync.Mutex
	classIndex     uint    // class的索引
	classSize      uintptr //  classSpan的长度
	startAddr      uintptr //bit索引
	npages         uintptr
	freeIndex      uintptr
	extensionPoint uintptr // 扩容负载因子

	//span中可以分配的个数
	nelems uintptr

	//bitmap 每个bit标识地址是否被分配(用来分配计算空闲地址)
	//1111111111111111111111111111111111111111111111111111111100000000 从右往左，遇到第一个1开始可以分配。
	allocCache uint64

	//allocCache 保存的指针
	allocBits *gcBits

	//gc标记的bitmap（1为不需要的）
	gcmarkBits *gcBits

	// gc
	baseMask  uint16 // if non-0, elemsize is a power of 2, & this will get object allocation base
	divMul    uint16 // for divide by elemsize - divMagic.mul
	divShift  uint8  // for divide by elemsize - divMagic.shift
	divShift2 uint8  // for divide by elemsize - divMagic.shift2

	allocCount uintptr

	next *xSpan
	//pre  *xSpan
	heap *xHeap
}

func (s *xSpan) layout() (size, n, total uintptr) {
	total = s.npages << _PageShift
	size = s.classSize
	if size > 0 {
		n = total / size
	}
	return
}
func (s *xSpan) Init(fact float32, heap *xHeap) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	_, n, _ := s.layout()
	s.extensionPoint = uintptr(float32(n) * fact)
	// Init the markbit structures
	s.heap = heap
	s.freeIndex = 0
	s.allocCache = ^uint64(0) // all 1s indicating all free.
	s.nelems = n
	var err error
	s.gcmarkBits, err = newMarkBits(s.nelems, true)
	if err != nil {
		return err
	}
	s.allocBits, err = newAllocBits(s.nelems)
	if index := s.classIndex; index == 0 {
		s.divShift = 0
		s.divMul = 0
		s.divShift2 = 0
		s.baseMask = 0
	} else {
		m := &class_to_divmagic[index]
		s.divShift = m.shift
		s.divMul = m.mul
		s.divShift2 = m.shift2
		s.baseMask = m.baseMask
	}
	return err
}

func (s *xSpan) freeOffset() (ptr uintptr, has bool) {
	ptr = s.nextFreeFast()
	if ptr == 0 {
		return s.nextFree()
	}
	return ptr, true
}

func (s *xSpan) needGrow() bool {
	val := atomic.LoadUintptr(&s.allocCount)
	return s.extensionPoint <= val
}

func (s *xSpan) isFull() bool {
	val := atomic.LoadUintptr(&s.allocCount)
	return (val+1)*s.classSize >= _PageSize*s.npages
}

//freeIndex增加：
// 1、首先使用自旋CAS获取空闲索引
// 2、CAS失败，则锁加重为排他锁
// 3、必须内存对齐
func (s *xSpan) getFree(bitSize uintptr) (oldIndex uintptr, has bool) {
	var swapped bool
	for {
		oldVal := atomic.LoadUintptr(&s.freeIndex)
		//  page对齐
		if aligned := (round(oldVal, _PageSize)) - oldVal; aligned < bitSize {
			bitSize += aligned
		}
		if oldVal+bitSize > _PageSize*(s.npages) {
			return 0, false
		}
		newVal := oldVal + bitSize
		swapped = atomic.CompareAndSwapUintptr(&s.freeIndex, oldVal, newVal)
		if swapped {
			return oldVal, true
		}
	}
}

func (s *xSpan) base() uintptr {
	return s.startAddr
}

func (s *xSpan) objIndex(p uintptr) uintptr {
	byteOffset := p - s.base()
	if byteOffset == 0 {
		return 0
	}
	if s.baseMask != 0 {
		// s.baseMask is non-0, elemsize is a power of two, so shift by s.divShift
		return byteOffset >> s.divShift
	}
	return uintptr(((uint64(byteOffset) >> s.divShift) * uint64(s.divMul)) >> s.divShift2)
}

func (s *xSpan) markBitsForIndex(objIndex uintptr) markBits {
	uint32p, mask := s.gcmarkBits.bitp(objIndex)
	return markBits{uint32p, mask, objIndex}
}

func (s *xSpan) setMarkBitsForIndex(objIndex uintptr) {
	uint32p, mask := s.gcmarkBits.bitp(objIndex)
	//fmt.Printf("xSpan:%d size:%d\n", uintptr(unsafe.Pointer(s)), s.classSize)
	s.heap.addFreeCapacity(int64(s.classSize))
	markBits{uint32p, mask, objIndex}.setMarked()
}

func (s *xSpan) markBitsForBase() markBits {
	return markBits{(*uint32)(s.gcmarkBits), uint32(1), 0}
}

//当前的allocCache中的最后一个bit空位和free位置。
func (s *xSpan) nextFreeFast() uintptr {
	if s.freeIndex >= s.nelems {
		return 0
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.freeIndex >= s.nelems {
		return 0
	}
	theBit := Ctz64(s.allocCache) // Is there a free object in the allocCache?
	if theBit < 64 {
		result := s.freeIndex + uintptr(theBit)
		if result < s.nelems {
			freeidx := result + 1
			if freeidx%64 == 0 && freeidx != s.nelems {
				return 0
			}
			s.allocCache >>= uint(theBit + 1)
			s.freeIndex = freeidx
			s.allocCount++
			return result*s.classSize + s.base()
		}
	}
	return 0
}

func (s *xSpan) nextFree() (v uintptr, has bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	offset, has := s.nextFreeIndex()
	if has {
		s.allocCount++
		return offset*s.classSize + s.base(), has
	}
	return
}

//当前allocCache没该内容
func (s *xSpan) nextFreeIndex() (uintptr, bool) {
	sfreeindex := s.freeIndex
	snelems := s.nelems
	if sfreeindex >= snelems {
		return sfreeindex, false
	}
	aCache := s.allocCache
	bitIndex := Ctz64(aCache)
	for bitIndex == 64 {
		// Move index to start of next cached bits.
		sfreeindex = (sfreeindex + 64) &^ (64 - 1)
		if sfreeindex >= snelems {
			s.freeIndex = snelems
			return snelems, false
		}
		whichByte := sfreeindex / 32
		// Refill s.allocCache with the next 64 alloc bits.
		s.refillAllocCache(whichByte)
		aCache = s.allocCache
		bitIndex = Ctz64(aCache)
	}
	result := sfreeindex + uintptr(bitIndex)
	if result >= snelems {
		s.freeIndex = snelems
		return snelems, false
	}
	s.allocCache >>= uint(bitIndex + 1)
	sfreeindex = result + 1
	if sfreeindex%64 == 0 && sfreeindex != snelems {
		whichByte := sfreeindex / 32
		s.refillAllocCache(whichByte)
	}
	s.freeIndex = sfreeindex
	return result, true
}

func (s *xSpan) refillAllocCache(whichByte uintptr) {
	bytes := (*[8]uint8)(unsafe.Pointer(s.allocBits.uint32p(whichByte)))
	aCache := uint64(0)
	aCache |= uint64(bytes[0])
	aCache |= uint64(bytes[1]) << (1 * 8)
	aCache |= uint64(bytes[2]) << (2 * 8)
	aCache |= uint64(bytes[3]) << (3 * 8)
	aCache |= uint64(bytes[4]) << (4 * 8)
	aCache |= uint64(bytes[5]) << (5 * 8)
	aCache |= uint64(bytes[6]) << (6 * 8)
	aCache |= uint64(bytes[7]) << (7 * 8)
	s.allocCache = aCache
}

//得到GC标记数目
func (s *xSpan) countGcMarkBits() uintptr {
	count := uintptr(0)
	maxIndex := s.nelems / 32
	for i := uintptr(0); i < maxIndex; i++ {
		mrkBits := s.gcmarkBits.uint32p(i)
		for _, b := range *(*[4]byte)(unsafe.Pointer(mrkBits)) {
			count += uintptr(oneBitCount[b])
		}
	}
	if bitsInLastByte := s.nelems % 32; bitsInLastByte != 0 {
		mrkBits := *s.gcmarkBits.uint32p(maxIndex)
		mask := uint32((1 << bitsInLastByte) - 1)
		bits := mrkBits & mask
		for _, b := range *(*[4]byte)(unsafe.Pointer(&bits)) {
			count += uintptr(oneBitCount[b])
		}
	}
	return count
}
