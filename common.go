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
	"github.com/spf13/cast"
	"log"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"unsafe"
)

func Recover() {
	if rec := recover(); rec != nil {
		if err, ok := rec.(error); ok {
			log.Printf("PanicRecover Unhandled error: %v\n stack:%v \n", err.Error(), cast.ToString(debug.Stack()))
		} else {
			log.Printf("PanicRecover Panic: %v\n stack:%v \n", rec, cast.ToString(debug.Stack()))
		}
	}
}

// align returns the smallest y >= x such that y % a == 0.
func Align(x, a uintptr) uintptr {
	if a < 1 {
		return x
	}
	y := x + a - 1
	return y - y%a
}

func round(n, a uintptr) uintptr {
	return (n + a - 1) &^ (a - 1)
}

const deBruijn64 = 0x0218a392cd3d5dbf

var deBruijnIdx64 = [64]byte{
	0, 1, 2, 7, 3, 13, 8, 19,
	4, 25, 14, 28, 9, 34, 20, 40,
	5, 17, 26, 38, 15, 46, 29, 48,
	10, 31, 35, 54, 21, 50, 41, 57,
	63, 6, 12, 18, 24, 27, 33, 39,
	16, 37, 45, 47, 30, 53, 49, 56,
	62, 11, 23, 32, 36, 44, 52, 55,
	61, 22, 43, 51, 60, 42, 59, 58,
}

// Ctz64 counts trailing (low-order) zeroes,
// and if all are zero, then 64.
// 找到最后的1在第几个bit位.  https://blog.csdn.net/cyq6239075/article/details/106412814
func Ctz64(x uint64) int {
	x &= -x                      // isolate low-order bit，计算出最后一个bit对应数字
	y := x * deBruijn64 >> 58    // extract part of deBruijn sequence
	i := int(deBruijnIdx64[y])   // convert to bit index
	z := int((x - 1) >> 57 & 64) // adjustment if zero
	return i + z
}

const TotalGCFactor = 0.0004
const SpanGCFactor = 0

// oneBitCount is indexed by byte and produces the
// number of 1 bits in that byte. For example 128 has 1 bit set
// and oneBitCount[128] will holds 1.
var oneBitCount = [256]uint8{
	0, 1, 1, 2, 1, 2, 2, 3,
	1, 2, 2, 3, 2, 3, 3, 4,
	1, 2, 2, 3, 2, 3, 3, 4,
	2, 3, 3, 4, 3, 4, 4, 5,
	1, 2, 2, 3, 2, 3, 3, 4,
	2, 3, 3, 4, 3, 4, 4, 5,
	2, 3, 3, 4, 3, 4, 4, 5,
	3, 4, 4, 5, 4, 5, 5, 6,
	1, 2, 2, 3, 2, 3, 3, 4,
	2, 3, 3, 4, 3, 4, 4, 5,
	2, 3, 3, 4, 3, 4, 4, 5,
	3, 4, 4, 5, 4, 5, 5, 6,
	2, 3, 3, 4, 3, 4, 4, 5,
	3, 4, 4, 5, 4, 5, 5, 6,
	3, 4, 4, 5, 4, 5, 5, 6,
	4, 5, 5, 6, 5, 6, 6, 7,
	1, 2, 2, 3, 2, 3, 3, 4,
	2, 3, 3, 4, 3, 4, 4, 5,
	2, 3, 3, 4, 3, 4, 4, 5,
	3, 4, 4, 5, 4, 5, 5, 6,
	2, 3, 3, 4, 3, 4, 4, 5,
	3, 4, 4, 5, 4, 5, 5, 6,
	3, 4, 4, 5, 4, 5, 5, 6,
	4, 5, 5, 6, 5, 6, 6, 7,
	2, 3, 3, 4, 3, 4, 4, 5,
	3, 4, 4, 5, 4, 5, 5, 6,
	3, 4, 4, 5, 4, 5, 5, 6,
	4, 5, 5, 6, 5, 6, 6, 7,
	3, 4, 4, 5, 4, 5, 5, 6,
	4, 5, 5, 6, 5, 6, 6, 7,
	4, 5, 5, 6, 5, 6, 6, 7,
	5, 6, 6, 7, 6, 7, 7, 8}

//mSpanList 支持并发的操作
type mSpanList struct {
	first *xSpan // first span in list, or nil if none
	//last  *xSpan // last span in list, or nil if none
	lock sync.Mutex
}

// 头插法(first)
func (list *mSpanList) insert(span *xSpan) {
	if span == nil {
		return
	}
	span.next = nil
	list.lock.Lock()
	defer list.lock.Unlock()
	for {
		addr := (*unsafe.Pointer)(unsafe.Pointer(&list.first))
		first := atomic.LoadPointer(addr)
		if first == nil && atomic.CompareAndSwapPointer(addr, nil, unsafe.Pointer(span)) {
			return
		}
		//先将新插入的赋值first，这时候会断裂为两个链。然后再赋值。
		if first != nil && atomic.CompareAndSwapPointer(addr, first, unsafe.Pointer(span)) {
			span.next = (*xSpan)(first)
			return
		}
	}
}

func (list *mSpanList) moveHead() *xSpan {
	list.lock.Lock()
	defer list.lock.Unlock()
	addr := (*unsafe.Pointer)(unsafe.Pointer(&list.first))
	for {
		if list.first != nil {
			head, next := list.first, list.first.next
			if atomic.CompareAndSwapPointer(addr, unsafe.Pointer(head), unsafe.Pointer(next)) {
				return head
			}
		} else {
			return nil
		}
	}
}

func (list *mSpanList) move(span *xSpan) {
	list.lock.Lock()
	defer list.lock.Unlock()
	for {
		var pre, next *xSpan
		for node := list.first; node != nil; node = node.next {
			if node == span {
				next = span.next
				break
			}
			pre = node
		}
		//并发cas
		var addr *unsafe.Pointer
		if pre != nil {
			addr = (*unsafe.Pointer)(unsafe.Pointer(&pre.next))
		} else {
			addr = (*unsafe.Pointer)(unsafe.Pointer(&list.first))
		}
		if atomic.CompareAndSwapPointer(addr, unsafe.Pointer(span), unsafe.Pointer(next)) {
			return
		}
	}
}

func (list *mSpanList) foreach(consumer func(span *xSpan)) {
	for node := list.first; node != nil; node = node.next {
		consumer(node)
	}
}
