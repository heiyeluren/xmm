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
	"runtime"
	"syscall"
	"unsafe"
)

const DefaultPhysPageSize = 4096
const _sunosEAGAIN = 11
const _ENOMEM = 12

var (
	errEAGAIN error = syscall.EAGAIN
	errEINVAL error = syscall.EINVAL
	errENOENT error = syscall.ENOENT

	LackOfMemoryErr error = errors.New("内存不足")
)

type linearAlloc struct {
	next   uintptr // next free byte
	mapped uintptr // one byte past end of mapped space
	end    uintptr // end of reserved space
}

func (l *linearAlloc) init(size uintptr) error {
	ptr, err := l.sysReserve(nil, size)
	if err != nil {
		return err
	}
	base := uintptr(ptr)
	l.next, l.mapped = base, base
	l.end = base + size
	return nil
}

func (l *linearAlloc) expand(addr unsafe.Pointer, aligned uintptr) error {
	arenaSizes := []uintptr{512 << 20, 256 << 20}
	p := l.end
	if p < 1 {
		p = uintptr(addr)
	}
	var er error
	for _, arenaSize := range arenaSizes {
		a, size, err := l.sysReserveAligned(unsafe.Pointer(p), arenaSize, aligned)
		if err != nil {
			er = err
			continue
		}
		if a != nil {
			if l.end-l.next < 1 {
				base := uintptr(a)
				l.next, l.mapped = base, base
				l.end = base + size
			} else {
				l.end += size
			}
			p = uintptr(a) + size // For hint below
			break
		}
	}
	return er
}

func (l *linearAlloc) alloc(size, align uintptr) (unsafe.Pointer, error) {
	p := round(l.next, align)
	if p+size > l.end {
		return nil, LackOfMemoryErr
	}
	l.next = p + size
	if pEnd := round(l.next-1, DefaultPhysPageSize); pEnd > l.mapped {
		// We need to map more of the reserved space.
		if err := l.sysMap(unsafe.Pointer(l.mapped), pEnd-l.mapped); err != nil {
			return nil, err
		}
		l.mapped = pEnd
	}
	return unsafe.Pointer(p), nil
}

func (l *linearAlloc) sysMap(addr unsafe.Pointer, length uintptr) error {
	prot, flags, fd, offset := syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_FIXED|syscall.MAP_PRIVATE, -1, 0
	v, _, err := syscall.Syscall6(syscall.SYS_MMAP, uintptr(addr), length, uintptr(prot), uintptr(flags), uintptr(fd), uintptr(offset))
	if err == _ENOMEM || (runtime.GOOS == "solaris" && err == _sunosEAGAIN) {
		return errors.New("runtime: out of memory")
	}
	if uintptr(addr) != v || err != 0 {
		return errors.New("runtime: cannot map pages in arena address space")
	}
	return nil
}

func (l *linearAlloc) errnoErr(e syscall.Errno) error {
	switch e {
	case 0:
		return nil
	case syscall.EAGAIN:
		return errEAGAIN
	case syscall.EINVAL:
		return errEINVAL
	case syscall.ENOENT:
		return errENOENT
	}
	return e
}

func (l *linearAlloc) sysReserve(addr unsafe.Pointer, size uintptr) (unsafe.Pointer, error) {
	ptr, _, err := syscall.Syscall6(syscall.SYS_MMAP, uintptr(addr), size, syscall.PROT_NONE,
		syscall.MAP_ANON|syscall.MAP_PRIVATE, uintptr(0), uintptr(0))
	if err != 0 {
		return nil, l.errnoErr(err)
	}
	return unsafe.Pointer(ptr), nil
}

// sysReserveAligned is like sysReserve, but the returned pointer is
// aligned to align bytes. It may reserve either n or n+align bytes,
// so it returns the size that was reserved.
func (l *linearAlloc) sysReserveAligned(v unsafe.Pointer, size, align uintptr) (unsafe.Pointer, uintptr, error) {
	// Since the alignment is rather large in uses of this
	// function, we're not likely to get it by chance, so we ask
	// for a larger region and remove the parts we don't need.
	retries := 0
retry:
	ptr, err := l.sysReserve(v, size+align)
	if err != nil {
		return nil, 0, err
	}
	p := uintptr(ptr)
	switch {
	case p == 0:
		return nil, 0, nil
	case p&(align-1) == 0:
		// We got lucky and got an aligned region, so we can
		// use the whole thing.
		return unsafe.Pointer(p), size + align, nil
	case runtime.GOOS == "windows":
		// On Windows we can't release pieces of a
		// reservation, so we release the whole thing and
		// re-reserve the aligned sub-region. This may race,
		// so we may have to try again.
		l.sysFree(unsafe.Pointer(p), size+align)
		p = round(p, align)
		p2, err := l.sysReserve(unsafe.Pointer(p), size)
		if err != nil {
			goto retry
		}
		if p != uintptr(p2) {
			// Must have raced. Try again.
			l.sysFree(p2, size)
			if retries++; retries == 100 {
				return nil, 0, errors.New("failed to allocate aligned heap memory; too many retries")
			}
			goto retry
		}
		// Success.
		return p2, size, nil
	default:
		// Trim off the unaligned parts.
		pAligned := round(p, align)
		l.sysFree(unsafe.Pointer(p), pAligned-p)
		end := pAligned + size
		endLen := (p + size + align) - end
		if endLen > 0 {
			l.sysFree(unsafe.Pointer(end), endLen)
		}
		return unsafe.Pointer(pAligned), size, nil
	}
}

func (l *linearAlloc) sysFree(addr unsafe.Pointer, length uintptr) (err error) {
	_, _, e1 := syscall.Syscall(syscall.SYS_MUNMAP, uintptr(addr), length, 0)
	if e1 != 0 {
		err = l.errnoErr(e1)
	}
	return
}
