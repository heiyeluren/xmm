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
	"testing"
	"unsafe"
)

func TestMarkBits(t *testing.T) {
	num, objIndex := 68, uintptr(10)
	bit, err := newMarkBits(uintptr(num), true)
	if err != nil {
		t.Fatal(err)
	}
	//fmt.Println(String(bit, 4), len(String(bit, 4)))

	bytep, mask := bit.bitp(objIndex)
	mb := markBits{bytep, mask, objIndex}
	fmt.Printf("bytep:%.32b ,  mask:%.32b isMarked:%t \n", *bytep, mask, mb.isMarked())
	mb.setMarked()
	//fmt.Println(String(bit, 4))
	bytep, mask = bit.bitp(objIndex)
	mb = markBits{bytep, mask, objIndex}

	fmt.Printf("bytep:%.32b ,  mask:%.32b isMarked:%t \n", *bytep, mask, mb.isMarked())

	//fmt.Println(String(bit, 4))
	bytep, mask = bit.bitp(uintptr(5000000))
	mb = markBits{bytep, mask, objIndex}

	fmt.Printf("bytep:%.32b ,  mask:%.32b isMarked:%t \n", *bytep, mask, mb.isMarked())
}

func TestMarkBits2(t *testing.T) {
	num, objIndex := 80, uintptr(78)
	bit, err := newMarkBits(uintptr(num), true)
	if err != nil {
		t.Fatal(err)
	}
	bytep, mask := bit.bitp(objIndex)
	mb := markBits{bytep, mask, objIndex}
	fmt.Printf("bytep:%.32b ,  mask:%.32b isMarked:%t \n", *bytep, mask, mb.isMarked())
	mb.setMarked()

	bytep, mask = bit.bitp(objIndex)
	ss := refillAllocCache(78/32, bit)
	fmt.Printf("AllocCache:%.64b ,  bytep:%.32b \n", ss, *(bytep))
}

func refillAllocCache(whichByte uintptr, allocBits *gcBits) uint64 {
	bytes := (*[8]uint8)(unsafe.Pointer(allocBits.uint32p(whichByte)))
	aCache := uint64(0)
	aCache |= uint64(bytes[0])
	aCache |= uint64(bytes[1]) << (1 * 8)
	aCache |= uint64(bytes[2]) << (2 * 8)
	aCache |= uint64(bytes[3]) << (3 * 8)
	aCache |= uint64(bytes[4]) << (4 * 8)
	aCache |= uint64(bytes[5]) << (5 * 8)
	aCache |= uint64(bytes[6]) << (6 * 8)
	aCache |= uint64(bytes[7]) << (7 * 8)
	return ^aCache
}
