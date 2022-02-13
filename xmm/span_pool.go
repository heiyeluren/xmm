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
	"unsafe"
)

type SpanOperation int

const (
	RemoveHead  SpanOperation = -2
	ExpendAsync SpanOperation = -3
	ExpendSync  SpanOperation = -4
)

type xSpanPool struct {
	lock                      [_NumSizeClasses]*sync.RWMutex
	inuse                     [_NumSizeClasses]uint64
	debug                     bool
	spanGen                   [_NumSizeClasses]int32     // 小于0 正在扩容
	spans                     [_NumSizeClasses]*[]*xSpan //预分配,spans很短，不存在引用超长，第一个为当前正在使用的，第二个为预先分配的span
	heap                      *xHeap
	spanFact                  float32
	specialPageNumCoefficient [_NumSizeClasses]uint8
	//1750 + 950
	classSpan [_NumSizeClasses]*xClassSpan
}

func newXSpanPool(heap *xHeap, spanFact float32) (*xSpanPool, error) {
	sp := &xSpanPool{heap: heap, spanFact: spanFact, classSpan: heap.classSpan}
	if err := sp.initLock(); err != nil {
		return nil, err
	}
	return sp, nil
}

func (sp *xSpanPool) initLock() error {
	for i := 0; i < _NumSizeClasses; i++ {
		var l sync.RWMutex
		sp.lock[i] = &l
	}
	return nil
}

func (sp *xSpanPool) allocClassSpan(index int) (ptr *xSpan, err error) {
	pageNum := class_to_allocnpages[index]
	size := class_to_size[index]
	pageNum = uint8(Align(Align(uintptr(size), _PageSize)/uintptr(_PageSize), uintptr(pageNum)))
	span, err := sp.classSpan[index].allocSpan(index, 0.75)
	if err != nil {
		return nil, err
	}
	return span, nil
}

func (sp *xSpanPool) addInuse(sizeClass uint8) uint64 {
	if !sp.debug {
		return 0
	}
	var swapped bool
	for /*retry := 3; retry > 0; retry--*/ {
		addr := &((sp.inuse)[sizeClass])
		oldVal := atomic.LoadUint64(addr)
		newVal := oldVal + 1
		swapped = atomic.CompareAndSwapUint64(addr, oldVal, newVal)
		if swapped {
			return newVal
		}
	}
}
func (sp *xSpanPool) AllocSlice(eleSize uintptr, cap, len uintptr) (p unsafe.Pointer, err error) {
	sl, err := sp.Alloc(eleSize*cap + unsafe.Sizeof(reflect.SliceHeader{}))
	if err != nil {
		return nil, err
	}
	data := uintptr(sl) + unsafe.Sizeof(reflect.SliceHeader{})
	sh := (*reflect.SliceHeader)(sl)
	sh.Cap = int(cap)
	sh.Data = data
	sh.Len = int(len)
	return sl, nil
}

var is bool

//通过增加key、value长度使得分配到不同span
//todo 擦除数据
func (sp *xSpanPool) Alloc(size uintptr) (p unsafe.Pointer, err error) {
	if size > _MaxSmallSize {
		pageNum := Align(size, _PageSize) / _PageSize
		chunk, err := sp.heap.allocRawSpan(pageNum)
		if err != nil {
			return nil, err
		}
		if err := chunk.Init(sp.spanFact, sp.heap); err != nil {
			return nil, err
		}
		chunk.allocCount = 1
		chunk.nelems = 1
		sp.classSpan[0].releaseSpan(chunk)
		return unsafe.Pointer(chunk.startAddr), nil
	}
	var sizeclass uint8
	if size <= smallSizeMax-8 {
		sizeclass = size_to_class8[(size+smallSizeDiv-1)/smallSizeDiv]
	} else {
		sizeclass = size_to_class128[(size-smallSizeMax+largeSizeDiv-1)/largeSizeDiv]
	}
	size = uintptr(class_to_size[sizeclass])
	spans, spanGen := sp.getSpan(sizeclass)
	var ptr, idex uintptr
	var has, needGrow bool
	for i, span := range spans {
		if span == nil {
			continue
		}
		if ptr, has = span.freeOffset(); has {
			if err := sp.clear(ptr, size); err != nil {
				log.Printf("xSpanPool.Alloc clear err:%s", err)
			}
			needGrow = span.needGrow()
			idex = uintptr(i)
			break
		}
	}
	if needGrow {
		if _, need, _ := sp.needExpendAsync(sizeclass, ExpendAsync); need {
			go sp.growSpan(sizeclass, ExpendAsync, spanGen)
		}
	}
	if idex < 1 && has {
		sp.addInuse(sizeclass)
		return unsafe.Pointer(ptr), nil
	}
	if idex > 0 && has {
		//idx前已经使用完了,删除前面满了的。
		sp.addInuse(sizeclass)
		sp.growSpan(sizeclass, RemoveHead, spanGen)
		return unsafe.Pointer(ptr), nil
	}
	// 同步扩容
	if !has {
		//所有的span都没有空位，需要阻塞同步分配 产生新的class span，释放所有的span
		if err := sp.growSpan(sizeclass, ExpendSync, spanGen); err != nil {
			return nil, err
		}
		return sp.Alloc(size)
	}
	return nil, fmt.Errorf("idex:%d has:%t is err", idex, has)
}

func (sp *xSpanPool) clear(ptr, size uintptr) (err error) {
	dst := *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{Data: ptr, Len: int(size), Cap: int(size)}))
	if length := copy(dst, make([]byte, size)); int(size) != length {
		return fmt.Errorf("xSpanPool.clear copy err:copy len is err")
	}
	return
}

/**
1、同步增、异步增、删除等方式
 todo 考虑并发扩容的问题(异步线程数控制)
*/
func (sp *xSpanPool) needExpendAsync(sizeClass uint8, op SpanOperation) (int32, bool, error) {
	if op != ExpendAsync {
		return -1, false, errors.New("not ExpendAsync")
	}
	addr := &sp.spanGen[sizeClass]
	currentSpanGen, newValue := atomic.LoadInt32(addr), op
	if currentSpanGen < 0 {
		return currentSpanGen, false, nil
	}
	if !atomic.CompareAndSwapInt32(addr, currentSpanGen, int32(newValue)) {
		return currentSpanGen, false, nil
	}
	return currentSpanGen, true, nil
}

func (sp *xSpanPool) growSpan(sizeClass uint8, op SpanOperation, spanGen int32) error {
	sp.lock[sizeClass].Lock()
	defer sp.lock[sizeClass].Unlock()
	addr := (*unsafe.Pointer)(unsafe.Pointer(&sp.spans[sizeClass]))
	val := atomic.LoadPointer(addr)
	var span []*xSpan
	if val != nil {
		span = *(*[]*xSpan)(val)
	}
	inuse := sp.inuse[sizeClass]
	if spanGen < 0 {
		spanGen = 1
	}
	switch op {
	case RemoveHead:
		var newIdex int
		if len(span) < 1 || !span[0].isFull() {
			return nil
		}
		sp.classSpan[sizeClass].releaseSpan(span[0])
		arr := span[1:]
		atomic.StorePointer(addr, unsafe.Pointer(&arr))
		sp.spanGen[sizeClass] = spanGen + 1
		if sp.debug {
			log.Printf("INFO modifySpan RemoveHead  sizeClass: %d   inuse:%d   spanGen:%d newIdex:%d\n", sizeClass, inuse, spanGen, newIdex)
		}
		return nil
	case ExpendAsync:
		if len(span) > 0 && !span[len(span)-1].needGrow() {
			return nil
		}
		if err := sp.growClassSpan(int(sizeClass), span); err != nil {
			log.Printf("ERR classSpan err, err: %s\n", err)
		}
		if sp.debug {
			log.Printf("INFO modifySpan ExpendAsync  sizeClass: %d   inuse:%d   spanGen:%d newspanGen:%d\n", sizeClass, inuse, spanGen, spanGen+1)
		}
		sp.spanGen[sizeClass] = spanGen + 1
		return nil
	case ExpendSync:
		if len(span) > 0 && !span[len(span)-1].needGrow() {
			return nil
		}
		if err := sp.growClassSpan(int(sizeClass), span); err != nil {
			return err
		}
		sp.spanGen[sizeClass] = spanGen + 1
		if sp.debug {
			log.Printf("INFO modifySpan ExpendSync  sizeClass: %d   inuse:%d   spanGen:%d newspanGen:%d\n", sizeClass, inuse, spanGen, spanGen+1)
		}
		return nil
	default:
		log.Println("ERR modifySpan modifySpan err: op is not support")
		return fmt.Errorf("op[%d] is not support", op)
	}
}

func (sp *xSpanPool) getSpan(sizeClass uint8) (spans []*xSpan, spanGen int32) {
	addr := (*unsafe.Pointer)(unsafe.Pointer(&sp.spans[sizeClass]))
	val := atomic.LoadPointer(addr)
	if val == nil {
		return
	}
	spans = *(*[]*xSpan)(val)
	spanGen = sp.spanGen[sizeClass]
	return spans, spanGen
}

func (sp *xSpanPool) growClassSpan(sizeClass int, spans []*xSpan) error {
	span, err := sp.allocClassSpan(sizeClass)
	if err != nil {
		return err
	}
	length := len(spans)
	newSpans := make([]*xSpan, length, length+1)
	if copy(newSpans, spans) != length {
		return fmt.Errorf("copy err")
	}
	newSpans = append(newSpans, span)
	addr := (*unsafe.Pointer)(unsafe.Pointer(&sp.spans[sizeClass]))
	atomic.StorePointer(addr, unsafe.Pointer(&newSpans))
	return nil
}

func (sp *xSpanPool) Copy2(item1 []byte, item2 []byte) (newItem1 []byte, newItem2 []byte, err error) {
	item1Size, item2Size := len(item1), len(item2)
	rawSize := item1Size + item2Size
	if rawSize < 1 {
		return nil, nil, nil
	}
	dataPtr, err := sp.Alloc(uintptr(rawSize))
	if err != nil {
		return nil, nil, err
	}
	dst := reflect.SliceHeader{Data: uintptr(dataPtr), Len: 0, Cap: rawSize}
	dstBytes := *(*[]byte)(unsafe.Pointer(&dst))
	dstBytes = append(dstBytes, item1...)
	dstBytes = append(dstBytes, item2...)
	dst.Len = item1Size
	dst.Cap = item1Size
	str1 := *(*[]byte)(unsafe.Pointer(&dst))
	dst.Len = item2Size
	dst.Cap = item2Size
	dst.Data = uintptr(dataPtr) + uintptr(item1Size)
	str2 := *(*[]byte)(unsafe.Pointer(&dst))
	return str1, str2, err
}

var TestBbulks uintptr

func (sp *xSpanPool) Free(addr uintptr) error {
	return sp.heap.free(addr)
}

func newXConcurrentHashMapSpanPool(heap *xHeap, spanFact float32, pageNumCoefficient uint8) (*xSpanPool, error) {
	sp := &xSpanPool{heap: heap, spanFact: spanFact, classSpan: heap.classSpan}
	sp.specialPageNumCoefficient[1], sp.specialPageNumCoefficient[2], sp.specialPageNumCoefficient[4], sp.specialPageNumCoefficient[6] =
		pageNumCoefficient*10, pageNumCoefficient*5, pageNumCoefficient*1, pageNumCoefficient*1
	if err := sp.initLock(); err != nil {
		return nil, err
	}
	return sp, nil
}

//启动利用class_to_allocnpages 预先分配span。
//alloc超过阈值，异步预分配
//alloc没有空闲时候，同步分配（防止分配太多）。
