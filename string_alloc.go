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
	"reflect"
	"unsafe"
)

type xStringAllocator struct {
	sp spanPool
}

func newXStringAllocator(sp spanPool) *xStringAllocator {
	return &xStringAllocator{sp: sp}
}

func (sa *xStringAllocator) alloc(size uintptr) (sh *reflect.StringHeader, err error) {
	if size < 0 {
		return nil, errors.New("size == 0")
	}
	p, err := sa.sp.Alloc(unsafe.Sizeof(reflect.StringHeader{}))
	if err != nil {
		return nil, err
	}
	sh = (*reflect.StringHeader)(p)
	sh.Len = int(size)
	dataPtr, err := sa.sp.Alloc(size)
	if err != nil {
		return nil, err
	}
	sh.Data = uintptr(dataPtr)
	return sh, nil
}

//todo  这里有bug，StringHeader没有计算进去，应该使用[]byte
func (sa *xStringAllocator) FromInAddr(addr uintptr, contents ...string) (p []*string, err error) {
	offset := addr
	p = make([]*string, len(contents))
	for i, content := range contents {
		size := uintptr(len(content))
		sh := (*reflect.StringHeader)(unsafe.Pointer(offset))
		sh.Len = int(size)
		offset += unsafe.Sizeof(reflect.StringHeader{})

		sh.Data = offset
		offset += size
		dst := reflect.SliceHeader{Data: sh.Data, Len: sh.Len, Cap: sh.Len}
		dstBytes := *(*[]byte)(unsafe.Pointer(&dst))
		sourcePtr := (*reflect.StringHeader)(unsafe.Pointer(&content))
		source := reflect.SliceHeader{Data: sourcePtr.Data, Len: sourcePtr.Len, Cap: sourcePtr.Len}
		srcByte := *(*[]byte)(unsafe.Pointer(&source))
		if len := copy(dstBytes, srcByte); len != sh.Len {
			return nil, fmt.Errorf("copy length is err: len = %d\n", len)
		}
		p[i] = (*string)(unsafe.Pointer(sh))
	}
	return p, nil
}

func (sa *xStringAllocator) From2(item1 string, item2 string) (newItem1 string, newItem2 string, err error) {
	raw := item1 + item2
	rawSize, item1Size, item2Size := len(raw), len(item1), len(item2)
	if rawSize < 1 {
		return "", "", nil
	}
	dataPtr, err := sa.sp.Alloc(uintptr(rawSize))
	if err != nil {
		return "", "", err
	}
	dst := reflect.SliceHeader{Data: uintptr(dataPtr), Len: rawSize, Cap: rawSize}
	dstBytes := *(*[]byte)(unsafe.Pointer(&dst))
	if len := copy(dstBytes, raw); len != rawSize {
		return "", "", fmt.Errorf("copy length is err: len = %d\n", len)
	}
	dst.Len = item1Size
	str1 := *(*string)(unsafe.Pointer(&dst))
	dst.Len = item2Size
	dst.Data = uintptr(dataPtr) + uintptr(item1Size)
	str2 := *(*string)(unsafe.Pointer(&dst))
	return str1, str2, err
}

func (sa *xStringAllocator) From(content string) (p string, err error) {
	size := uintptr(len(content))
	dataPtr, err := sa.sp.Alloc(size)
	if err != nil {
		return "", err
	}
	length := int(size)
	dst := reflect.SliceHeader{Data: uintptr(dataPtr), Len: length, Cap: length}
	dstBytes := *(*[]byte)(unsafe.Pointer(&dst))
	if len := copy(dstBytes, content); len != length {
		return "", fmt.Errorf("copy length is err: len = %d\n", len)
	}
	return *(*string)(unsafe.Pointer(&dst)), err
}


func (sa *xStringAllocator) FreeString(content string)error{
	sh := (*reflect.StringHeader)(unsafe.Pointer(&content))
	return sa.sp.Free(sh.Data)
}

