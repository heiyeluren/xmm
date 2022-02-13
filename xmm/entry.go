/*
Copyright 2014 Gavin Bong.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
either express or implied. See the License for the specific
language governing permissions and limitations under the
License.
*/

// Package redblacktree provides a pure Golang implementation
// of a red-black tree as described by Thomas H. Cormen's et al.
// in their seminal Algorithms book (3rd ed). This data structure
// is not multi-goroutine safe.
package xmm

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"strings"
	"sync"
)

type NodeEntry struct {
	Key   []byte
	Value []byte
	Hash  uint64
	Next  *NodeEntry

	right  *NodeEntry
	left   *NodeEntry
	parent *NodeEntry
	color  Color //比二叉查找树要多出一个颜色属性
}


type encodedNodeEntry struct {
	TotalLen uint64
	KeyLen   uint64
	ValueLen uint64
	Hash     uint64
	Next     *NodeEntry

	right  *NodeEntry
	left   *NodeEntry
	parent *NodeEntry
	color  Color //比二叉查找树要多出一个颜色属性

	//追加string  key、value 要不然就内存泄露了

}

/*// total'len(64) + key'len(64) + key'content +
func (n *NodeEntry) encode() []byte {
	var keyLen, valLen = round(uint64(len(n.Key)), 8), round(uint64(len(n.Value)), 8)
	total := keyLen + valLen + 8*8
	encoded := encodedNodeEntry{
		TotalLen: total,
		KeyLen:   keyLen,
		ValueLen: valLen,
		Hash:     0,
		Next:     nil,
		right:    nil,
		left:     nil,
		parent:   nil,
		color:    false,
	}

}*/

var BytesAscSort Comparator = func(o1, o2 interface{}) int {
	key1, key2 := o1.([]byte), o2.([]byte)
	return bytes.Compare(key1, key2)
}

// Color of a redblack tree node is either
// `Black` (true) & `Red` (false)
type Color bool

// Direction points to either the Left or Right subtree
type Direction byte

func (c Color) String() string {
	switch c {
	case true:
		return "Black"
	default:
		return "Red"
	}
}

func (d Direction) String() string {
	switch d {
	case LEFT:
		return "left"
	case RIGHT:
		return "right"
	case NODIR:
		return "center"
	default:
		return "not recognized"
	}
}

const (
	BLACK, RED Color = true, false
)
const (
	LEFT Direction = iota
	RIGHT
	NODIR
)

// A node needs to be able to answer the query:
// (i) Who is my parent node ?
// (ii) Who is my grandparent node ?
// The zero value for Node has color Red.

func (n *NodeEntry) String() string {
	if n == nil {
		return ""
	}
	return fmt.Sprintf("(%#v : %s)", n.Key, n.Color())
}

func (n *NodeEntry) Parent() *NodeEntry {
	return n.parent
}

func (n *NodeEntry) SetColor(color Color) {
	n.color = color
}

func (n *NodeEntry) Color() Color {
	return n.color
}

func (n *NodeEntry) Left() *NodeEntry {
	return n.left
}

func (n *NodeEntry) Right() *NodeEntry {
	return n.right
}

type Visitor interface {
	Visit(*NodeEntry)
}

// A redblack tree is `Visitable` by a `Visitor`.
type Visitable interface {
	Walk(Visitor)
}

// Keys must be comparable. It's mandatory to provide a Comparator,
// which returns zero if o1 == o2, -1 if o1 < o2, 1 if o1 > o2
type Comparator func(o1, o2 interface{}) int

// Default comparator expects keys to be of type `int`.
// Warning: if either one of `o1` or `o2` cannot be asserted to `int`, it panics.
func IntComparator(o1, o2 interface{}) int {
	i1 := o1.(int)
	i2 := o2.(int)
	switch {
	case i1 > i2:
		return 1
	case i1 < i2:
		return -1
	default:
		return 0
	}
}

// Keys of type `string`.
// Warning: if either one of `o1` or `o2` cannot be asserted to `string`, it panics.
func StringComparator(o1, o2 interface{}) int {
	s1 := o1.(string)
	s2 := o2.(string)
	return bytes.Compare([]byte(s1), []byte(s2))
}

// Tree encapsulates the data structure.
type Tree struct {
	root *NodeEntry // tip of the tree
	cmp  Comparator // required function to order keys
	lock sync.RWMutex
}

// `lock` protects `logger`
var loggerlock sync.Mutex
var logger *log.Logger

func init() {
	logger = log.New(ioutil.Discard, "", log.LstdFlags)
}

// TraceOn turns on logging output to Stderr
func TraceOn() {
	SetOutput(os.Stderr)
}

// TraceOff turns off logging.
// By default logging is turned off.
func TraceOff() {
	SetOutput(ioutil.Discard)
}

// SetOutput redirects log output
func SetOutput(w io.Writer) {
	loggerlock.Lock()
	defer loggerlock.Unlock()
	logger = log.New(w, "", log.LstdFlags)
}

// NewTree returns an empty Tree with default comparator `IntComparator`.
// `IntComparator` expects keys to be type-assertable to `int`.
func NewTree() *Tree {
	return &Tree{root: nil, cmp: IntComparator}
}

// NewTreeWith returns an empty Tree with a supplied `Comparator`.
func NewTreeWith(c Comparator) *Tree {
	return &Tree{root: nil, cmp: c}
}

func (t *Tree) SetComparator(c Comparator) {
	t.cmp = c
}

// Get looks for the node with supplied key and returns its mapped payload.
// Return value in 1st position indicates whether any payload was found.
func (t *Tree) Get(key []byte) (bool, []byte) {
	if err := mustBeValidKey(key); err != nil {
		//logger.Printf("Get was prematurely aborted: %s\n", err.Error())
		return false, nil
	}
	ok, node := t.getNode(key)
	if ok {
		return true, node.Value
	} else {
		return false, nil
	}
}

func (t *Tree) getNode(key interface{}) (bool, *NodeEntry) {
	found, parent, dir := t.GetParent(key)
	if found {
		if parent == nil {
			return true, t.root
		} else {
			var node *NodeEntry
			switch dir {
			case LEFT:
				node = parent.left
			case RIGHT:
				node = parent.right
			}

			if node != nil {
				return true, node
			}
		}
	}
	return false, nil
}

// getMinimum returns the node with minimum key starting
// at the subtree rooted at node x. Assume x is not nil.
func (t *Tree) getMinimum(x *NodeEntry) *NodeEntry {
	for {
		if x.left != nil {
			x = x.left
		} else {
			return x
		}
	}
}

// GetParent looks for the node with supplied key and returns the parent node.
func (t *Tree) GetParent(key interface{}) (found bool, parent *NodeEntry, dir Direction) {
	if err := mustBeValidKey(key); err != nil {
		//logger.Printf("GetParent was prematurely aborted: %s\n", err.Error())
		return false, nil, NODIR
	}

	if t.root == nil {
		return false, nil, NODIR
	}
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.internalLookup(nil, t.root, key, NODIR)
}

func (t *Tree) internalLookup(parent *NodeEntry, this *NodeEntry, key interface{}, dir Direction) (bool, *NodeEntry, Direction) {
	switch {
	case this == nil:
		return false, parent, dir
	case t.cmp(key, this.Key) == 0:
		return true, parent, dir
	case t.cmp(key, this.Key) < 0:
		return t.internalLookup(this, this.left, key, LEFT)
	case t.cmp(key, this.Key) > 0:
		return t.internalLookup(this, this.right, key, RIGHT)
	default:
		return false, parent, NODIR
	}
}

// Reverses actions of RotateLeft
func (t *Tree) RotateRight(y *NodeEntry) {
	if y == nil {
		//logger.Printf("RotateRight: nil arg cannot be rotated. Noop\n")
		return
	}
	if y.left == nil {
		//logger.Printf("RotateRight: y has nil left subtree. Noop\n")
		return
	}
	//logger.Printf("\t\t\trotate right of %s\n", y)
	x := y.left
	y.left = x.right
	if x.right != nil {
		x.right.parent = y
	}
	x.parent = y.parent
	if y.parent == nil {
		t.root = x
	} else {
		if y == y.parent.left {
			y.parent.left = x
		} else {
			y.parent.right = x
		}
	}
	x.right = y
	y.parent = x
}

// Side-effect: red-black tree properties is maintained.
func (t *Tree) RotateLeft(x *NodeEntry) {
	if x == nil {
		//logger.Printf("RotateLeft: nil arg cannot be rotated. Noop\n")
		return
	}
	if x.right == nil {
		//logger.Printf("RotateLeft: x has nil right subtree. Noop\n")
		return
	}
	//logger.Printf("\t\t\trotate left of %s\n", x)

	y := x.right
	x.right = y.left
	if y.left != nil {
		y.left.parent = x
	}
	y.parent = x.parent
	if x.parent == nil {
		t.root = y
	} else {
		if x == x.parent.left {
			x.parent.left = y
		} else {
			x.parent.right = y
		}
	}
	y.left = x
	x.parent = y
}

// Put saves the mapping (key, data) into the tree.
// If a mapping identified by `key` already exists, it is overwritten.
// Constraint: Not everything can be a key.
func (t *Tree) Put(node *NodeEntry) error {
	node.parent, node.left, node.Next, node.right = nil, nil, nil, nil
	key, data := node.Key, node.Value
	if err := mustBeValidKey(key); err != nil {
		//logger.Printf("Put was prematurely aborted: %s\n", err.Error())
		return err
	}
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.root == nil {
		node.color = BLACK
		t.root = node
		//logger.Printf("Added %s as root node\n", t.root.String())
		return nil
	}

	found, parent, dir := t.internalLookup(nil, t.root, key, NODIR)
	if found {
		if parent == nil {
			//logger.Printf("Put: parent=nil & found. Overwrite ROOT node\n")
			t.root.Value = data
		} else {
			//logger.Printf("Put: parent!=nil & found. Overwriting\n")
			switch dir {
			case LEFT:
				parent.left.Value = data
			case RIGHT:
				parent.right.Value = data
			}
		}

	} else {
		if parent != nil {
			node.parent = parent
			newNode := node
			switch dir {
			case LEFT:
				parent.left = newNode
			case RIGHT:
				parent.right = newNode
			}
			//logger.Printf("Added %s to %s node of parent %s\n", newNode.String(), dir, parent.String())
			t.fixupPut(newNode)
		}
	}
	return nil
}

func isRed(n *NodeEntry) bool {
	key := reflect.ValueOf(n)
	if key.IsNil() {
		return false
	} else {
		return n.color == RED
	}
}

// fix possible violations of red-black-tree properties
// with combinations of:
// 1. recoloring
// 2. rotations
//
// Preconditions:
// P1) z is not nil
//
// @param z - the newly added Node to the tree.
func (t *Tree) fixupPut(z *NodeEntry) {
	//logger.Printf("\tfixup new node z %s\n", z.String())
loop:
	for {
		//logger.Printf("\tcurrent z %s\n", z.String())
		switch {
		case z.parent == nil:
			fallthrough
		case z.parent.color == BLACK:
			fallthrough
		default:
			// When the loop terminates, it does so because p[z] is black.
			//logger.Printf("\t\t=> bye\n")
			break loop
		case z.parent.color == RED:
			grandparent := z.parent.parent
			//logger.Printf("\t\tgrandparent is nil  %t addr:%d\n", grandparent == nil, unsafe.Pointer(t))
			if z.parent == grandparent.left {
				//logger.Printf("\t\t%s is the left child of %s\n", z.parent, grandparent)
				y := grandparent.right
				////logger.Printf("\t\ty (right) %s\n", y)
				if isRed(y) {
					// case 1 - y is RED
					//logger.Printf("\t\t(*) case 1\n")
					z.parent.color = BLACK
					y.color = BLACK
					grandparent.color = RED
					z = grandparent

				} else {
					if z == z.parent.right {
						// case 2
						//logger.Printf("\t\t(*) case 2\n")
						z = z.parent
						t.RotateLeft(z)
					}

					// case 3
					//logger.Printf("\t\t(*) case 3\n")
					z.parent.color = BLACK
					grandparent.color = RED
					t.RotateRight(grandparent)
				}
			} else {
				//logger.Printf("\t\t%s is the right child of %s\n", z.parent, grandparent)
				y := grandparent.left
				//logger.Printf("\t\ty (left) %s\n", y)
				if isRed(y) {
					// case 1 - y is RED
					//logger.Printf("\t\t..(*) case 1\n")
					z.parent.color = BLACK
					y.color = BLACK
					grandparent.color = RED
					z = grandparent

				} else {
					//logger.Printf("\t\t## %s\n", z.parent.left)
					if z == z.parent.left {
						// case 2
						//logger.Printf("\t\t..(*) case 2\n")
						z = z.parent
						t.RotateRight(z)
					}

					// case 3
					//logger.Printf("\t\t..(*) case 3\n")
					z.parent.color = BLACK
					grandparent.color = RED
					t.RotateLeft(grandparent)
				}
			}
		}
	}
	t.root.color = BLACK
}

// Size returns the number of items in the tree.
func (t *Tree) Size() uint64 {
	visitor := &countingVisitor{}
	t.Walk(visitor)
	return visitor.Count
}

// Has checks for existence of a item identified by supplied key.
func (t *Tree) Has(key interface{}) bool {
	if err := mustBeValidKey(key); err != nil {
		//logger.Printf("Has was prematurely aborted: %s\n", err.Error())
		return false
	}
	found, _, _ := t.internalLookup(nil, t.root, key, NODIR)
	return found
}

func (t *Tree) transplant(u *NodeEntry, v *NodeEntry) {
	if u.parent == nil {
		t.root = v
	} else if u == u.parent.left {
		u.parent.left = v
	} else {
		u.parent.right = v
	}
	if v != nil && u != nil {
		v.parent = u.parent
	}
}

// Delete removes the item identified by the supplied key.
// Delete is a noop if the supplied key doesn't exist.
func (t *Tree) Delete(key []byte) *NodeEntry {
	if !t.Has(key) {
		//logger.Printf("Delete: bail as no node exists for key %d\n", key)
		return nil
	}
	_, z := t.getNode(key)
	y := z
	yOriginalColor := y.color
	var x *NodeEntry

	if z.left == nil {
		// one child (RIGHT)
		//logger.Printf("\t\tDelete: case (a)\n")
		x = z.right
		//logger.Printf("\t\t\t--- x is right of z")
		t.transplant(z, z.right)

	} else if z.right == nil {
		// one child (LEFT)
		//logger.Printf("\t\tDelete: case (b)\n")
		x = z.left
		//logger.Printf("\t\t\t--- x is left of z")
		t.transplant(z, z.left)

	} else {
		// two children
		//logger.Printf("\t\tDelete: case (c) & (d)\n")
		y = t.getMinimum(z.right)
		//logger.Printf("\t\t\tminimum of z.right is %s (color=%s)\n", y, y.color)
		yOriginalColor = y.color
		x = y.right
		//logger.Printf("\t\t\t--- x is right of minimum")

		if y.parent == z {
			if x != nil {
				x.parent = y
			}
		} else {
			t.transplant(y, y.right)
			y.right = z.right
			y.right.parent = y
		}
		t.transplant(z, y)
		y.left = z.left
		y.left.parent = y
		y.color = z.color
	}
	if yOriginalColor == BLACK {
		t.fixupDelete(x)
	}
	return z
}

func (t *Tree) fixupDelete(x *NodeEntry) {
	//logger.Printf("\t\t\tfixupDelete of node %s\n", x)
	if x == nil {
		return
	}
loop:
	for {
		switch {
		case x == t.root:
			//logger.Printf("\t\t\t=> bye .. is root\n")
			break loop
		case x.color == RED:
			//logger.Printf("\t\t\t=> bye .. RED\n")
			break loop
		case x == x.parent.right:
			//logger.Printf("\t\tBRANCH: x is right child of parent\n")
			w := x.parent.left // is nillable
			if isRed(w) {
				// Convert case 1 into case 2, 3, or 4
				//logger.Printf("\t\t\tR> case 1\n")
				w.color = BLACK
				x.parent.color = RED
				t.RotateRight(x.parent)
				w = x.parent.left
			}
			if w != nil {
				switch {
				case !isRed(w.left) && !isRed(w.right):
					// case 2 - both children of w are BLACK
					//logger.Printf("\t\t\tR> case 2\n")
					w.color = RED
					x = x.parent // recurse up tree
				case isRed(w.right) && !isRed(w.left):
					// case 3 - right child RED & left child BLACK
					// convert to case 4
					//logger.Printf("\t\t\tR> case 3\n")
					w.right.color = BLACK
					w.color = RED
					t.RotateLeft(w)
					w = x.parent.left
				}
				if isRed(w.left) {
					// case 4 - left child is RED
					//logger.Printf("\t\t\tR> case 4\n")
					w.color = x.parent.color
					x.parent.color = BLACK
					w.left.color = BLACK
					t.RotateRight(x.parent)
					x = t.root
				}
			}
		case x == x.parent.left:
			//logger.Printf("\t\tBRANCH: x is left child of parent\n")
			w := x.parent.right // is nillable
			if isRed(w) {
				// Convert case 1 into case 2, 3, or 4
				//logger.Printf("\t\t\tL> case 1\n")
				w.color = BLACK
				x.parent.color = RED
				t.RotateLeft(x.parent)
				w = x.parent.right
			}
			if w != nil {
				switch {
				case !isRed(w.left) && !isRed(w.right):
					// case 2 - both children of w are BLACK
					//logger.Printf("\t\t\tL> case 2\n")
					w.color = RED
					x = x.parent // recurse up tree
				case isRed(w.left) && !isRed(w.right):
					// case 3 - left child RED & right child BLACK
					// convert to case 4
					//logger.Printf("\t\t\tL> case 3\n")
					w.left.color = BLACK
					w.color = RED
					t.RotateRight(w)
					w = x.parent.right
				}
				if isRed(w.right) {
					// case 4 - right child is RED
					//logger.Printf("\t\t\tL> case 4\n")
					w.color = x.parent.color
					x.parent.color = BLACK
					w.right.color = BLACK
					t.RotateLeft(x.parent)
					x = t.root
				}
			}
		}
	}
	x.color = BLACK
}

// Walk accepts a Visitor
func (t *Tree) Walk(visitor Visitor) {
	visitor.Visit(t.root)
}

func (t *Tree) GetRoot() *NodeEntry {
	return t.root
}

// countingVisitor counts the number
// of nodes in the tree.
type countingVisitor struct {
	Count uint64
}

func (v *countingVisitor) Visit(node *NodeEntry) {
	if node == nil {
		return
	}

	v.Visit(node.left)
	v.Count = v.Count + 1
	v.Visit(node.right)
}

// InorderVisitor walks the tree in inorder fashion.
// This visitor maintains internal state; thus do not
// reuse after the completion of a walk.
type InorderVisitor struct {
	buffer bytes.Buffer
}

func (v *InorderVisitor) Eq(other *InorderVisitor) bool {
	if other == nil {
		return false
	}
	return v.String() == other.String()
}

func (v *InorderVisitor) trim(s string) string {
	return strings.TrimRight(strings.TrimRight(s, "ed"), "lack")
}

func (v *InorderVisitor) String() string {
	return v.buffer.String()
}

func (v *InorderVisitor) Visit(node *NodeEntry) {
	if node == nil {
		v.buffer.Write([]byte("."))
		return
	}
	v.buffer.Write([]byte("("))
	v.Visit(node.left)
	v.buffer.Write([]byte(fmt.Sprintf("%s", node.Key))) // @TODO
	//v.buffer.Write([]byte(fmt.Sprintf("%d{%s}", node.Key, v.trim(node.color.String()))))
	v.Visit(node.right)
	v.buffer.Write([]byte(")"))
}

type HookVisitor struct {
	Hook func(node *NodeEntry)
}

func (v *HookVisitor) Visit(node *NodeEntry) {
	if node == nil {
		return
	}
	v.Hook(node)
	v.Visit(node.left)
	v.Visit(node.right)
}

var (
	ErrorKeyIsNil      = errors.New("The literal nil not allowed as keys")
	ErrorKeyDisallowed = errors.New("Disallowed key typsssssssse")
)

// Allowed key types are: Boolean, Integer, Floating point, Complex, String values
// And structs containing these.
// @TODO Should pointer type be allowed ?
func mustBeValidKey(key interface{}) error {
	if key == nil {
		return ErrorKeyIsNil
	}

	/*keyValue := reflect.ValueOf(key)
	switch keyValue.Kind() {
	case reflect.Chan:
		fallthrough
	case reflect.Func:
		fallthrough
	case reflect.Interface:
		fallthrough
	case reflect.Map:
		fallthrough
	case reflect.Ptr:
		return ErrorKeyDisallowed
	default:
		return nil
	}*/
	return nil
}
