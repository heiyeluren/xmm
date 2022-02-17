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
	"unsafe"
)

var NilError = errors.New("params is illegal")

type spanPool interface {
	Alloc(byteSize uintptr) (p unsafe.Pointer, err error)
	AllocSlice(eleSize uintptr, cap, len uintptr) (p unsafe.Pointer, err error)
	Free(addr uintptr) error
	Copy2(item1 []byte, item2 []byte) (newItem1 []byte, newItem2 []byte, err error)
}

type stringAllocator interface {
	From(content string) (p string, err error)
	From2(item1 string, item2 string) (newItem1 string, newItem2 string, err error)
	FromInAddr(addr uintptr, contents ...string) (p []*string, err error)
	FreeString(content string) error
}

type Chunk struct {
	StartAddr uintptr
	Npages    uintptr
}

type XMemory interface {
	spanPool
	stringAllocator
	RawAlloc(pageNum uintptr) (p *Chunk, err error)
	GetPageSize() uintptr
}

type mm struct {
	sp spanPool
	sa stringAllocator
	h  *xHeap
}

func (m *mm) Copy2(item1 []byte, item2 []byte) (newItem1 []byte, newItem2 []byte, err error) {
	return m.sp.Copy2(item1, item2)
}

func (m *mm) Alloc(byteSize uintptr) (p unsafe.Pointer, err error) {
	if byteSize < 1 {
		return nil, NilError
	}
	return m.sp.Alloc(byteSize)
}

func (m *mm) AllocSlice(eleSize uintptr, cap, len uintptr) (p unsafe.Pointer, err error) {
	if eleSize < 1 || cap < 1 {
		return nil, NilError
	}
	return m.sp.AllocSlice(eleSize, cap, len)
}

func (m *mm) From(content string) (p string, err error) {
	if len(content) < 1 {
		return "", NilError
	}
	return m.sa.From(content)
}

func (m *mm) From2(item1 string, item2 string) (newItem1 string, newItem2 string, err error) {
	return m.sa.From2(item1, item2)
}

func (m *mm) FromInAddr(addr uintptr, contents ...string) (p []*string, err error) {
	if addr < 1 {
		return p, NilError
	}
	return m.sa.FromInAddr(addr, contents...)
}

func (m *mm) RawAlloc(pageNum uintptr) (p *Chunk, err error) {
	if pageNum < 1 {
		return p, NilError
	}
	if c, err := m.h.allocRawSpan(pageNum); err != nil {
		return nil, err
	} else {
		return &Chunk{StartAddr: c.startAddr, Npages: c.npages}, nil
	}
}

func (m *mm) Free(addr uintptr) error {
	if addr < 1 {
		return NilError
	}
	return m.sp.Free(addr)
}

func (m *mm) FreeString(content string) error {
	return m.sa.FreeString(content)
}



func (m *mm) GetPageSize() uintptr {
	return _PageSize
}

type Factory struct {
	sp *xSpanPool
}

func (s *Factory) CreateMemory(spanFact float32) (XMemory, error) {
	if spanFact <= 0 {
		return nil, NilError
	}
	h, err := newXHeap()
	if err != nil {
		return nil, err
	}
	sp, err := newXSpanPool(h, spanFact)
	if err != nil {
		return nil, err
	}
	s.sp = sp
	sa := newXStringAllocator(sp)
	return &mm{sp: sp, sa: sa, h: h}, nil
}

//NewXConcurrentHashMapSpanPool
func (s *Factory) CreateConcurrentHashMapMemory(spanFact float32, pageNumCoefficient uint8) (XMemory, error) {
	if spanFact <= 0 {
		return nil, NilError
	}
	h, err := newXHeap()
	if err != nil {
		return nil, err
	}
	sp, err := newXConcurrentHashMapSpanPool(h, spanFact, pageNumCoefficient)
	if err != nil {
		return nil, err
	}
	sa := newXStringAllocator(sp)
	s.sp = sp
	return &mm{sp: sp, sa: sa, h: h}, nil
}

func (s *Factory) PrintStatus() {
	for index, u := range s.sp.inuse {
		if u < 100 {
			continue
		}
		pageNum := class_to_allocnpages[index]
		size := class_to_size[index]
		pageNum = uint8(Align(Align(uintptr(size), _PageSize)/uintptr(_PageSize), uintptr(pageNum)))
		fmt.Println(index, u, uintptr(pageNum)*_PageSize/uintptr(size))
	}
}

