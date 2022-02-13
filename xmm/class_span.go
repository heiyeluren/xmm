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
)

type xClassSpan struct {

	//lock sync.Mutex

	classIndex uint // class的索引

	//空的，
	free *mSpanList

	//满的span。
	full *mSpanList

	// nmalloc is the cumulative count of objects allocated from
	// this mcentral, assuming all spans in mcaches are
	// fully-allocated. Written atomically, read under STW.
	nmalloc uint64

	heap *xHeap
}

func (x *xClassSpan) Init(classIndex uint, heap *xHeap) error {
	if heap == nil {
		return errors.New("heap is nil")
	}
	x.classIndex = classIndex
	x.heap = heap
	x.free = &mSpanList{}
	x.full = &mSpanList{}
	return nil
}

func (x *xClassSpan) allocSpan(index int, f float32) (*xSpan, error) {
	if x.free.first != nil {
		span, err := func() (*xSpan, error) {
			return x.free.moveHead(), nil
		}()
		if span != nil && err == nil {
			//log.Printf("xClassSpan class:%d  free申请 span:%d\n", x.classIndex, unsafe.Pointer(span))
			return span, nil
		}
	}
	heap := x.heap
	pageNum := class_to_allocnpages[index]
	size := class_to_size[index]
	pageNum = uint8(Align(Align(uintptr(size), _PageSize)/uintptr(_PageSize), uintptr(pageNum)))
	span, err := heap.allocSpan(uintptr(pageNum), uint(index), uintptr(size), f)
	//log.Printf("xClassSpan heap.allocSpan class:%d  free申请 span:%d\n", x.classIndex, unsafe.Pointer(span))
	if err != nil {
		return nil, err
	}
	return span, nil
}

func (x *xClassSpan) releaseSpan(span *xSpan) {
	/*if logg || span.classIndex == 6 {
		fmt.Println("releaseSpan", uintptr(unsafe.Pointer(span)))
	}*/
	x.full.insert(span)
}

/**
  释放span，只能释放full中的
*/
func (x *xClassSpan) freeSpan(span *xSpan) (swap bool, size uint, err error) {
	if span == nil {
		return false, 0, errors.New("span is nil")
	}
	needFree := false
	err = func() error {
		gcCount := span.countGcMarkBits()
		//没有达到gc阈值或者当前span正在被分配不做GC（异步gc该类span）
		if gcCount <= uintptr(float64(span.nelems)*SpanGCFactor) {
			return nil
		}
		if span.allocCount < span.nelems {
			return nil
		}
		span.lock.Lock()
		defer span.lock.Unlock()
		allocCount := span.allocCount
		gcCount = span.countGcMarkBits()
		x.heap.addFreeCapacity(-int64(allocCount))
		//没有达到gc阈值或者当前span正在被分配不做GC（异步gc该类span）
		if gcCount < uintptr(float64(span.nelems)*SpanGCFactor) {
			return nil
		}
		if span.allocCount < span.nelems {
			return nil
		}
		size = uint(gcCount * span.classSize)
		needFree = true
		span.freeIndex = 0
		span.allocCount = span.nelems - gcCount
		//span.gcmarkBits.show64(span.nelems)
		span.allocBits = span.gcmarkBits
		span.gcmarkBits, err = newMarkBits(span.nelems, true)
		if err != nil {
			return err
		}
		span.refillAllocCache(0)
		//log.Printf("refillAllocCache allocCache:%.64b gcCount:%d allocCount:%d gcCount:%d oldallocCount:%d\n", span.allocCache, gcCount, span.allocCount, gcCount, allocCount)
		return nil
	}()
	if err != nil {
		return false, 0, err
	}
	if !needFree {
		return false, 0, nil
	}
	//判断当前span是否不在使用，不在使用存放进去。在使用则
	//log.Printf("xClassSpan class:%d 回收 span:%d\n", x.classIndex, unsafe.Pointer(span))
	x.free.insert(span)
	return true, size, nil
}
