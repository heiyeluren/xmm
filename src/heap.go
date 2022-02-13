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
	"fmt"
	"log"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

type xHeap struct {
	lock sync.Mutex

	freeChunks *xTreap // treap树

	rawLinearMemoryAlloc linearAlloc

	//默认RawMemoryL1Bits=0,退化为以为数据。RawMemoryL2Bits = 20个
	addrMap [1 << RawMemoryL1Bits]*[1 << RawMemoryL2Bits]*xRawLinearMemory //addr -> page -> rawMemory 关系

	allChunk []*xChunk

	allChunkAllocator *xSliceAllocator

	chunkAllocator *xAllocator

	spanAllocator *xAllocator

	rawLinearMemoryAllocator *xAllocator

	classSpan [_NumSizeClasses]*xClassSpan

	totalCapacity int64

	freeCapacity int64

	sweepIndex uint32

	sweepCtl int32 // -10000<= sweepCtl < 0 正在扩容  0:非扩容状态

	sweepLastTime time.Time
}

const sweepCtlStatus = -68

func newXHeap() (*xHeap, error) {
	call := func(inuse uintptr) { log.Printf("XSliceAllocator xChunk 扩容了，使用了 inuse:%d\n", inuse) }
	chunkAllocator := newXAllocator(unsafe.Sizeof(xChunk{}))
	valAllocator := newXAllocator(unsafe.Sizeof(treapNode{}))
	spanAllocator := newXAllocator(unsafe.Sizeof(xSpan{}))
	allChunkAllocator, err := newXSliceAllocator(unsafe.Sizeof(&xChunk{}), 16, call)
	rawLinearMemoryAllocator := newXAllocator(unsafe.Sizeof(xRawLinearMemory{}))
	if err != nil {
		return nil, err
	}
	freeChunks := newXTreap(valAllocator)
	heap := &xHeap{allChunkAllocator: allChunkAllocator, chunkAllocator: chunkAllocator, freeChunks: freeChunks,
		spanAllocator: spanAllocator, rawLinearMemoryAllocator: rawLinearMemoryAllocator}
	if err := heap.rawLinearMemoryAlloc.expand(nil, heapRawMemoryBytes); err != nil {
		return nil, err
	}
	if err := heap.initClassSpan(); err != nil {
		return nil, err
	}
	return heap, nil
}

func (xh *xHeap) initClassSpan() error {
	for i := 0; i < _NumSizeClasses; i++ {
		classSpan := &xClassSpan{}
		xh.classSpan[i] = classSpan
		if err := classSpan.Init(uint(i), xh); err != nil {
			return err
		}
	}
	return nil
}

func (xh *xHeap) addFreeCapacity(size int64) {
	for {
		val := atomic.LoadInt64(&xh.freeCapacity)
		newVal := val + size
		if atomic.CompareAndSwapInt64(&xh.freeCapacity, val, newVal) {
			return
		}
	}
}

func (xh *xHeap) addChunks(xChunks []*xChunk) error {
	if xh.allChunk == nil || len(xChunks)+len(xh.allChunk) > cap(xh.allChunk) {
		cap := cap(xh.allChunk) << 1
		//log.Printf("xHeap.addChunks grow cap:%d\n", cap)
		err := xh.allChunkAllocator.grow(uintptr(cap), func(newPtr unsafe.Pointer) error {
			sl := reflect.SliceHeader{
				Data: uintptr(newPtr),
				Len:  len(xh.allChunk),
				Cap:  cap,
			}
			newXChunks := *(*[]*xChunk)(unsafe.Pointer(&sl))
			length := copy(newXChunks, xh.allChunk[:len(xh.allChunk)])
			if length != len(xh.allChunk) {
				return errors.New("copy 数量错误")
			}
			xh.allChunk = newXChunks
			return nil
		})
		if err != nil {
			return err
		}
	}
	xh.allChunk = append(xh.allChunk, xChunks...)
	return nil
}

func (xh *xHeap) allocRawSpan(pageNum uintptr) (span *xSpan, err error) {
	chunkP, err := xh.spanAllocator.alloc()
	if err != nil {
		return nil, err
	}
	// 增加chunks、free、addrMap
	span = (*xSpan)(chunkP)
	chunk, err := xh.allocChunk(pageNum)
	if err != nil {
		return nil, err
	}
	xh.setSpans(chunk.startAddr, pageNum, span)
	// chunk -> xSpan
	span.freeIndex = 0
	span.npages = pageNum
	span.startAddr = chunk.startAddr
	return span, nil
}

func (xh *xHeap) setSpans(base, npage uintptr, s *xSpan) {
	p := base / _PageSize
	ai := RawMemoryIndex(base)
	ha := xh.addrMap[ai.l1()][ai.l2()]
	for n := uintptr(0); n < npage; n++ {
		i := (p + n) % pagesPerRawMemory
		if i == 0 {
			ai = RawMemoryIndex(base + n*_PageSize)
			ha = xh.addrMap[ai.l1()][ai.l2()]
		}
		ha.spans[i] = s
	}
}

func (xh *xHeap) spanOf(p uintptr) (*xSpan, error) {
	ri := RawMemoryIndex(p)
	if RawMemoryL1Bits == 0 {
		if ri.l2() >= uint(len(xh.addrMap[0])) {
			return nil, fmt.Errorf("err: l2(%d)is err", ri.l2())
		}
	} else {
		if ri.l1() >= uint(len(xh.addrMap)) {
			return nil, fmt.Errorf("err: l1(%d)is err", ri.l1())
		}
	}
	l2 := xh.addrMap[ri.l1()]
	if RawMemoryL1Bits != 0 && l2 == nil { // Should never happen if there's no L1.
		return nil, fmt.Errorf("err: l1(%d)is nil", ri.l2())
	}
	ha := l2[ri.l2()]
	if ha == nil {
		return nil, fmt.Errorf("err: l2(%d)is nil", ri.l2())
	}
	return ha.spans[(p/_PageSize)%pagesPerRawMemory], nil
}

func (xh *xHeap) allocSpan(pageNum uintptr, index uint, class uintptr, fact float32) (span *xSpan, err error) {
	chunkP, err := xh.spanAllocator.alloc()
	if err != nil {
		return nil, err
	}
	// 增加chunks、free、addrMap
	span = (*xSpan)(chunkP)
	if pageNum > 0 {
		chunk, err := xh.allocChunk(pageNum)
		if err != nil {
			return nil, err
		}
		xh.setSpans(chunk.startAddr, pageNum, span)
		span.startAddr = chunk.startAddr
	}
	span.classIndex = index
	span.classSize = class
	span.freeIndex = 0
	span.npages = pageNum
	if err := span.Init(fact, xh); err != nil {
		return nil, err
	}
	return span, nil
}

func (xh *xHeap) allocChunk(pageNum uintptr) (ptr *xChunk, err error) {
	chunk, err := xh.freeChunk(pageNum)
	if err != nil {
		return nil, err
	}
	return chunk, err
}

// 必须加锁分配
func (xh *xHeap) freeChunk(pageNum uintptr) (ptr *xChunk, err error) {
	xh.lock.Lock()
	defer xh.lock.Unlock()
	node, err := xh.freeChunks.find(pageNum)
	if err == EmptyError {
		if err := xh.grow2(pageNum); err != nil {
			return nil, err
		}
		node, err = xh.freeChunks.find(pageNum)
	}
	if err != nil {
		return nil, err
	}
	if node.chunk == nil {
		return nil, errors.New("node val is nil")
	}
	if node.npagesKey < pageNum {
		return nil, errors.New("node val is small")
	}
	startAddr, npages := node.chunk.startAddr, node.chunk.npages
	if err := xh.freeChunks.removeNode(node); err != nil {
		return nil, err
	}
	if node.npagesKey == pageNum {
		return &xChunk{startAddr: startAddr, npages: pageNum}, nil
	}
	node.chunk.npages = npages - pageNum
	node.chunk.startAddr = startAddr + pageNum*_PageSize
	if err := xh.freeChunks.insert(node.chunk); err != nil {
		return nil, err
	}
	return &xChunk{startAddr: startAddr, npages: pageNum}, nil
}

//todo 释放：地址中保存len、保存空闲地址、要么直接复用，要么合并page再复用
func (xh *xHeap) free(addr uintptr) error {
	//todo 标记完成，接下来触发清理
	//panic("todo 释放：地址中保存len、保存空闲地址、要么直接复用，要么合并page再复用,存放到树状结构")
	//todo 还给 span，比较高效。
	//key:开始地址  value:结束地址   存放到红黑树中。
	//key找key最相近的，找到判断value。有则更新，没有则插入。
	if err := xh.mark(addr); err != nil {
		return err
	}
	// 统计
	xh.sweep()
	return nil
}

func (xh *xHeap) needSweep() bool {
	val, sweepThreshold := atomic.LoadInt64(&xh.freeCapacity), float64(xh.totalCapacity)*TotalGCFactor
	if sweepThreshold > float64(val) {
		return false
	}
	if time.Now().Sub(xh.sweepLastTime).Seconds() <= 1 {
		return false
	}
	sweepCtl := atomic.LoadInt32(&xh.sweepCtl)
	if sweepCtl < 0 {
		//扩容线程+1
		if atomic.CompareAndSwapInt32(&xh.sweepCtl, sweepCtl, sweepCtl+1) {
			return true
		}
	}
	if sweepCtl > 0 {
		return false
	}
	if atomic.CompareAndSwapInt32(&xh.sweepCtl, sweepCtl, sweepCtlStatus) {
		return true
	}
	return false
}

var logg bool

// todo classSpan中并发支持
func (xh *xHeap) sweep() {
	//统计判断
	if !xh.needSweep() {
		return
	}
	var sweepIndex uint32
	var total uint
	for sweepIndex = atomic.LoadUint32(&xh.sweepIndex); sweepIndex < _NumSizeClasses; sweepIndex = atomic.LoadUint32(&xh.sweepIndex) {
		if !atomic.CompareAndSwapUint32(&xh.sweepIndex, sweepIndex, sweepIndex+1) {
			continue
		}
		classSpan := xh.classSpan[sweepIndex]
		//todo 环循环
		for span := classSpan.full.first; span != nil; span = span.next {
			if logg {
				fmt.Println("sweep", sweepIndex, uintptr(unsafe.Pointer(span)))
			}
			if sweep, size, err := xh.sweepFullSpan(span); err != nil {
				log.Printf("xHeap.sweep err:%s\n ", err)
				continue
			} else if sweep {
				total += size
				classSpan.full.move(span)
			}
		}
	}
	fmt.Println("--------sweep---------", time.Now().String(), total)
	xh.sweepLastTime = time.Now()
	if total < 1 {
		return
	}
	//logg = true
	xh.addFreeCapacity(0 - int64(total))
	for {
		sweepCtl := atomic.LoadInt32(&xh.sweepCtl)
		if sweepCtl >= 0 {
			return
		}
		if atomic.CompareAndSwapInt32(&xh.sweepCtl, sweepCtl, sweepCtl-1) {
			if sweepCtl != sweepCtlStatus {
				return
			}
			//sweep执行结束
			atomic.StoreInt32(&xh.sweepCtl, 0)
			atomic.StoreUint32(&xh.sweepIndex, 0)
			return
		}
	}
}
func (xh *xHeap) ChunkInsert(chunk *xChunk) error {
	xh.lock.Lock()
	defer xh.lock.Unlock()
	return xh.freeChunks.insert(chunk)
}

// 清理span（span级别锁）
func (xh *xHeap) sweepFullSpan(span *xSpan) (sweep bool, size uint, err error) {
	//fmt.Println("======================")
	if span.classIndex > 0 {
		// 所有还给classspan
		return xh.classSpan[span.classIndex].freeSpan(span)
	} else if span.classIndex == 0 && span.nelems > 0 {
		//大对象释放
		span.lock.Lock()
		defer span.lock.Unlock()
		if span.nelems != span.countGcMarkBits() {
			return
		}
		chunkP, err := xh.chunkAllocator.alloc()
		if err != nil {
			return false, 0, err
		}
		// 增加chunks、free、addrMap
		chunk := (*xChunk)(chunkP)
		if TestBbulks == span.startAddr {
			fmt.Println("ssssssss")
		}
		chunk.startAddr = span.startAddr
		chunk.npages = span.npages
		if err := xh.ChunkInsert(chunk); err != nil {
			return false, 0, err
		}
	}
	return true, uint(span.npages * _PageSize), nil
}

func (xh *xHeap) mark(addr uintptr) error {
	err := markBitsForAddr(addr, xh)
	if err != nil {
		return err
	}
	return nil
}

func (xh *xHeap) grow2(pageNum uintptr) error {
	size := pageNum * _PageSize
	if size < heapRawMemoryBytes {
		size = heapRawMemoryBytes
	}
	p, err := xh.rawLinearMemoryAlloc.alloc(size, heapRawMemoryBytes)
	xh.totalCapacity += int64(size)
	if err != nil && err != LackOfMemoryErr {
		return err
	}
	if err == LackOfMemoryErr {
		if err := xh.rawLinearMemoryAlloc.expand(nil, heapRawMemoryBytes); err != nil {
			var la linearAlloc
			la.expand(nil, heapRawMemoryBytes)
			xh.rawLinearMemoryAlloc = la
		}
		return xh.grow2(pageNum)
	}
	chunkP, err := xh.chunkAllocator.alloc()
	if err != nil {
		return err
	}
	// 增加chunks、free、addrMap
	chunk := (*xChunk)(chunkP)
	chunk.startAddr = uintptr(p)
	chunk.npages = size / _PageSize
	xh.freeChunks.insert(chunk)
	if err := xh.addChunks([]*xChunk{chunk}); err != nil {
		return err
	}
	offset := uintptr(p)
	for i := 0; i < int(Align(size, heapRawMemoryBytes)/heapRawMemoryBytes); i++ {
		//页地址到RawMemory保存
		rawLinearMemoryPtr, err := xh.rawLinearMemoryAllocator.alloc()
		if err != nil {
			return err
		}
		//addrMap 初始化xRawLinearMemory
		rlm := (*xRawLinearMemory)(rawLinearMemoryPtr)
		index := RawMemoryIndex(offset)
		if addrs := xh.addrMap[index.l1()]; addrs == nil {
			var a [1 << RawMemoryL2Bits]*xRawLinearMemory
			xh.addrMap[index.l1()] = &a
			addrs = &a
		}
		xh.addrMap[index.l1()][index.l2()] = rlm
		offset += heapRawMemoryBytes
	}
	return nil
}

func (xh *xHeap) grow() error {
	p, err := xh.rawLinearMemoryAlloc.alloc(heapRawMemoryBytes, heapRawMemoryBytes)
	xh.totalCapacity += heapRawMemoryBytes
	if err != nil && err != LackOfMemoryErr {
		return err
	}
	if err == LackOfMemoryErr {
		if err := xh.rawLinearMemoryAlloc.expand(nil, heapRawMemoryBytes); err != nil {
			var la linearAlloc
			la.expand(nil, heapRawMemoryBytes)
			xh.rawLinearMemoryAlloc = la
		}
		return xh.grow()
	}
	chunkP, err := xh.chunkAllocator.alloc()
	if err != nil {
		return err
	}
	// 增加chunks、free、addrMap
	chunk := (*xChunk)(chunkP)
	chunk.startAddr = uintptr(p)
	chunk.npages = pagesPerRawMemory
	xh.freeChunks.insert(chunk)
	if err := xh.addChunks([]*xChunk{chunk}); err != nil {
		return err
	}
	//页地址到RawMemory保存
	rawLinearMemoryPtr, err := xh.rawLinearMemoryAllocator.alloc()
	if err != nil {
		return err
	}
	rlm := (*xRawLinearMemory)(rawLinearMemoryPtr)
	offset := uintptr(p)
	for i := 0; i < pagesPerRawMemory; i++ {
		index := RawMemoryIndex(offset)
		addrs := xh.addrMap[index.l1()]
		if addrs == nil {
			var a [1 << RawMemoryL2Bits]*xRawLinearMemory
			xh.addrMap[index.l1()] = &a
			addrs = &a
		}
		addrs[index.l2()] = rlm
	}
	return nil
}

func RawMemoryIndex(p uintptr) RawMemoryIdx {
	return RawMemoryIdx((p + RawMemoryBaseOffset) / heapRawMemoryBytes)
}

// RawMemoryBase returns the low address of the region covered by heap
// RawMemory i.
func RawMemoryBase(i RawMemoryIdx) uintptr {
	return uintptr(i)*heapRawMemoryBytes - RawMemoryBaseOffset
}

type RawMemoryIdx uint

func (i RawMemoryIdx) l1() uint {
	if RawMemoryL1Bits == 0 {
		// Let the compiler optimize this away if there's no
		// L1 map.
		return 0
	} else {
		return uint(i) >> RawMemoryL1Shift
	}
}

func (i RawMemoryIdx) l2() uint {
	if RawMemoryL1Bits == 0 {
		return uint(i)
	} else {
		return uint(i) & (1<<RawMemoryL2Bits - 1)
	}
}
