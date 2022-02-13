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
)

func TestFreeOffset(t *testing.T) {
	span := &xSpan{}
	span.freeIndex = 2
	span.npages = 4
	span.classSize = uintptr(class_to_size[span.freeIndex])

	span.startAddr = 1
	span.Init(0.75, nil)
	for i := 0; i < 700; i++ {
		ptr, has := span.freeOffset()
		if !has && uintptr(i) <= span.npages*(_PageSize)/(span.classSize)-1 {
			t.Fatal("+++++", ptr, has)
		}
		if has && uintptr(i) > (span.npages)*uintptr(_PageSize)/(span.classSize)-1 {
			t.Fatal("+++++", ptr, has)
		}
	}
}

func TestGcFreeOffset(t *testing.T) {
	span := &xSpan{}
	span.freeIndex = 2
	span.npages = 1
	span.classSize = uintptr(class_to_size[span.freeIndex])
	span.startAddr = 1
	span.Init(0.75, nil)

	bit := span.gcmarkBits
	objIndexs := []int{1, 6, 3, 2, 10, 80}
	for _, index := range objIndexs {
		objIndex := uintptr(index)
		bytep, mask := bit.bitp(objIndex)
		mb := markBits{bytep, mask, objIndex}
		fmt.Printf("bytep:%.32b ,  mask:%.32b isMarked:%t \n", *bytep, mask, mb.isMarked())
		mb.setMarked()
	}
	span.allocBits = span.gcmarkBits
	span.refillAllocCache(0)
	max := span.npages*(_PageSize)/(span.classSize) - 1 - uintptr(len(objIndexs))
	hasOffset := 0
	for i := 0; i < 700; i++ {
		ptr, has := span.freeOffset()
		if !has && uintptr(i) <= max {
			t.Fatal("+++++", ptr, has)
		}
		if has && uintptr(i) > max {
			t.Fatal("-----", ptr, has)
		}
		if has {
			hasOffset = i
		}
	}
	fmt.Println(span.npages*(_PageSize)/(span.classSize), hasOffset, span.npages*(_PageSize)/(span.classSize)-1-uintptr(hasOffset))
}
