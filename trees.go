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
	"math/rand"
)

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Page heap.
//
// See malloc.go for the general overview.
//
// Large spans are the subject of this file. Spans consisting of less than
// _MaxMHeapLists are held in lists of like sized spans. Larger spans
// are held in a treap. See https://en.wikipedia.org/wiki/Treap or
// https://faculty.washington.edu/aragon/pubs/rst89.pdf for an overview.
// sema.go also holds an implementation of a treap.
//
// Each treapNode holds a single span. The treap is sorted by page size
// and for spans of the same size a secondary sort based on start address
// is done.
// Spans are returned based on a best fit algorithm and for spans of the same
// size the one at the lowest address is selected.
//
// The primary routines are
// insert: adds a span to the treap
// remove: removes the span from that treap that best fits the required size
// removeSpan: which removes a specific span from the treap
//
// _mheap.lock must be held when manipulating this data structure.

var EmptyError = errors.New("not found")

type xTreap struct {
	treap        *treapNode
	valAllocator *xAllocator
}

func newXTreap(valAllocator *xAllocator) *xTreap {
	return &xTreap{valAllocator: valAllocator}
}

func (s *xChunk) base() uintptr {
	return s.startAddr
}

type treapNode struct {
	right     *treapNode // all treapNodes > this treap node
	left      *treapNode // all treapNodes < this treap node
	parent    *treapNode // direct parent of this node, nil if root
	npagesKey uintptr    // number of pages in spanKey, used as primary sort key
	priority  uint32     // random number used by treap algorithm to keep tree probabilistically balanced
	chunk     *xChunk
}

func (t *treapNode) pred() (*treapNode, error) {
	if t.left != nil {
		// If it has a left child, its predecessor will be
		// its right most left (grand)child.
		t = t.left
		for t.right != nil {
			t = t.right
		}
		return t, nil
	}
	// If it has no left child, its predecessor will be
	// the first grandparent who's right child is its
	// ancestor.
	//
	// We compute this by walking up the treap until the
	// current node's parent is its parent's right child.
	//
	// If we find at any point walking up the treap
	// that the current node doesn't have a parent,
	// we've hit the root. This means that t is already
	// the left-most node in the treap and therefore
	// has no predecessor.
	for t.parent != nil && t.parent.right != t {
		if t.parent.left != t {
			println("runtime: predecessor t=", t, "t.chunk=", t.chunk)
			return nil, errors.New("node is not its parent's child")
		}
		t = t.parent
	}
	return t.parent, nil
}

func (t *treapNode) succ() (*treapNode, error) {
	if t.right != nil {
		// If it has a right child, its successor will be
		// its left-most right (grand)child.
		t = t.right
		for t.left != nil {
			t = t.left
		}
		return t, nil
	}
	// See pred.
	for t.parent != nil && t.parent.left != t {
		if t.parent.right != t {
			println("runtime: predecessor t=", t, "t.chunk=", t.chunk)
			return nil, errors.New("node is not its parent's child")
		}
		t = t.parent
	}
	return t.parent, nil
}

// isSpanInTreap is handy for debugging. One should hold the heap lock, usually
// mheap_.lock().
func (t *treapNode) isChunkInTreap(s *xChunk) bool {
	if t == nil {
		return false
	}
	return t.chunk == s || t.left.isChunkInTreap(s) || t.right.isChunkInTreap(s)
}

// walkTreap is handy for debugging.
// Starting at some treapnode t, for example the root, do a depth first preorder walk of
// the tree executing fn at each treap node. One should hold the heap lock, usually
// mheap_.lock().
func (t *treapNode) walkTreap(fn func(tn *treapNode)) {
	if t == nil {
		return
	}
	fn(t)
	t.left.walkTreap(fn)
	t.right.walkTreap(fn)
}

// checkTreapNode when used in conjunction with walkTreap can usually detect a
// poorly formed treap.
func checkTreapNode(t *treapNode) error {
	// lessThan is used to order the treap.
	// npagesKey and npages are the primary keys.
	// spanKey and span are the secondary keys.
	// span == nil (0) will always be lessThan all
	// spans of the same size.
	lessThan := func(npages uintptr, chunk *xChunk) bool {
		if t.npagesKey != npages {
			return t.npagesKey < npages
		}
		// t.npagesKey == npages
		return t.chunk.base() < chunk.base()
	}

	if t == nil {
		return nil
	}
	if t.chunk.npages != t.npagesKey {
		println("runtime: checkTreapNode treapNode t=", t, "     t.npagesKey=", t.npagesKey,
			"t.chunk.npages=", t.chunk.npages)
		return errors.New("span.npages and treap.npagesKey do not match")
	}
	if t.left != nil && lessThan(t.left.npagesKey, t.left.chunk) {
		return errors.New("t.lessThan(t.left.npagesKey, t.left.chunk) is not false")
	}
	if t.right != nil && !lessThan(t.right.npagesKey, t.right.chunk) {
		return errors.New("!t.lessThan(t.left.npagesKey, t.left.chunk) is not false")
	}
	return nil
}

// treapIter is a bidirectional iterator type which may be used to iterate over a
// an xTreap in-order forwards (increasing order) or backwards (decreasing order).
// Its purpose is to hide details about the treap from users when trying to iterate
// over it.
//
// To create iterators over the treap, call start or end on an xTreap.
type treapIter struct {
	t *treapNode
}

// span returns the span at the current position in the treap.
// If the treap is not valid, span will panic.
func (i *treapIter) span() *xChunk {
	return i.t.chunk
}

// valid returns whether the iterator represents a valid position
// in the xTreap.
func (i *treapIter) valid() bool {
	return i.t != nil
}

// next moves the iterator forward by one. Once the iterator
// ceases to be valid, calling next will panic.
func (i treapIter) next() (treapIter, error) {
	var err error
	i.t, err = i.t.succ()
	return i, err
}

// prev moves the iterator backwards by one. Once the iterator
// ceases to be valid, calling prev will panic.
func (i treapIter) prev() (treapIter, error) {
	var err error
	i.t, err = i.t.pred()
	return i, err
}

// start returns an iterator which points to the start of the treap (the
// left-most node in the treap).
func (root *xTreap) start() treapIter {
	t := root.treap
	if t == nil {
		return treapIter{}
	}
	for t.left != nil {
		t = t.left
	}
	return treapIter{t: t}
}

// end returns an iterator which points to the end of the treap (the
// right-most node in the treap).
func (root *xTreap) end() treapIter {
	t := root.treap
	if t == nil {
		return treapIter{}
	}
	for t.right != nil {
		t = t.right
	}
	return treapIter{t: t}
}

// insert adds span to the large span treap.
func (root *xTreap) insert(chunk *xChunk) error {
	npages := chunk.npages
	var last *treapNode
	pt := &root.treap
	for t := *pt; t != nil; t = *pt {
		last = t
		if t.npagesKey < npages {
			pt = &t.right
		} else if t.npagesKey > npages {
			pt = &t.left
		} else if t.chunk.base() < chunk.base() {
			// t.npagesKey == npages, so sort on span addresses.
			pt = &t.right
		} else if t.chunk.base() > chunk.base() {
			pt = &t.left
		} else {
			return errors.New("inserting span already in treap")
		}
	}

	// Add t as new leaf in tree of span size and unique addrs.
	// The balanced tree is a treap using priority as the random heap priority.
	// That is, it is a binary tree ordered according to the npagesKey,
	// but then among the space of possible binary trees respecting those
	// npagesKeys, it is kept balanced on average by maintaining a heap ordering
	// on the priority: s.priority <= both s.right.priority and s.right.priority.
	// https://en.wikipedia.org/wiki/Treap
	// https://faculty.washington.edu/aragon/pubs/rst89.pdf

	ptr, err := root.valAllocator.alloc()
	if err != nil {
		return err
	}
	t := (*treapNode)(ptr)
	t.npagesKey = chunk.npages
	t.priority = rand.Uint32()
	t.chunk = chunk
	t.left = nil
	t.right = nil
	t.parent = nil
	t.parent = last
	*pt = t // t now at a leaf.

	// Rotate up into tree according to priority.
	for t.parent != nil && t.parent.priority > t.priority {
		if t != nil && t.chunk.npages != t.npagesKey {
			println("runtime: insert t=", t, "t.npagesKey=", t.npagesKey)
			println("runtime:      t.chunk=", t.chunk, "t.chunk.npages=", t.chunk.npages)
			return errors.New("span and treap sizes do not match?")
		}
		if t.parent.left == t {
			root.rotateRight(t.parent)
		} else {
			if t.parent.right != t {
				return errors.New("treap insert finds a broken treap")
			}
			root.rotateLeft(t.parent)
		}
	}
	return nil
}

func (root *xTreap) removeNode(t *treapNode) error {
	if t.chunk.npages != t.npagesKey {
		return errors.New("span and treap node npages do not match")
	}
	// Rotate t down to be leaf of tree for removal, respecting priorities.
	for t.right != nil || t.left != nil {
		if t.right == nil || t.left != nil && t.left.priority < t.right.priority {
			root.rotateRight(t)
		} else {
			root.rotateLeft(t)
		}
	}
	// Remove t, now a leaf.
	if t.parent != nil {
		if t.parent.left == t {
			t.parent.left = nil
		} else {
			t.parent.right = nil
		}
	} else {
		root.treap = nil
	}
	// Return the found treapNode's span after freeing the treapNode.
	//mheap_.treapalloc.free(unsafe.Pointer(t))
	return nil
}

// find searches for, finds, and returns the treap node containing the
// smallest span that can hold npages. If no span has at least npages
// it returns nil.
// This is a simple binary tree search that tracks the best-fit node found
// so far. The best-fit node is guaranteed to be on the path to a
// (maybe non-existent) lowest-base exact match.
func (root *xTreap) find(npages uintptr) (*treapNode, error) {
	var best *treapNode
	t := root.treap
	for t != nil {
		if t.chunk == nil {
			return nil, errors.New("treap node with nil spanKey found")
		}
		// If we found an exact match, try to go left anyway. There could be
		// a span there with a lower base address.
		//
		// Don't bother checking nil-ness of left and right here; even if t
		// becomes nil, we already know the other path had nothing better for
		// us anyway.
		if t.npagesKey >= npages {
			best = t
			t = t.left
		} else {
			t = t.right
		}
	}
	if best == nil {
		return nil, EmptyError
	}
	return best, nil
}

// removeSpan searches for, finds, deletes span along with
// the associated treap node. If the span is not in the treap
// then t will eventually be set to nil and the t.chunk
// will throw.
func (root *xTreap) removeChunk(chunk *xChunk) error {
	npages := chunk.npages
	t := root.treap
	for t.chunk != chunk {
		if t.npagesKey < npages {
			t = t.right
		} else if t.npagesKey > npages {
			t = t.left
		} else if t.chunk.base() < chunk.base() {
			t = t.right
		} else if t.chunk.base() > chunk.base() {
			t = t.left
		}
	}
	return root.removeNode(t)
}

// erase removes the element referred to by the current position of the
// iterator. This operation consumes the given iterator, so it should no
// longer be used. It is up to the caller to get the next or previous
// iterator before calling erase, if need be.
func (root *xTreap) erase(i treapIter) {
	root.removeNode(i.t)
}

// rotateLeft rotates the tree rooted at node x.
// turning (x a (y b c)) into (y (x a b) c).
func (root *xTreap) rotateLeft(x *treapNode) error {
	// p -> (x a (y b c))
	p := x.parent
	a, y := x.left, x.right
	b, c := y.left, y.right

	y.left = x
	x.parent = y
	y.right = c
	if c != nil {
		c.parent = y
	}
	x.left = a
	if a != nil {
		a.parent = x
	}
	x.right = b
	if b != nil {
		b.parent = x
	}

	y.parent = p
	if p == nil {
		root.treap = y
	} else if p.left == x {
		p.left = y
	} else {
		if p.right != x {
			return errors.New("large span treap rotateLeft")
		}
		p.right = y
	}
	return nil
}

// rotateRight rotates the tree rooted at node y.
// turning (y (x a b) c) into (x a (y b c)).
func (root *xTreap) rotateRight(y *treapNode) error {
	// p -> (y (x a b) c)
	p := y.parent
	x, c := y.left, y.right
	a, b := x.left, x.right

	x.left = a
	if a != nil {
		a.parent = x
	}
	x.right = y
	y.parent = x
	y.left = b
	if b != nil {
		b.parent = y
	}
	y.right = c
	if c != nil {
		c.parent = y
	}

	x.parent = p
	if p == nil {
		root.treap = x
	} else if p.left == y {
		p.left = x
	} else {
		if p.right != y {
			return errors.New("large span treap rotateRight")
		}
		p.right = x
	}
	return nil
}
