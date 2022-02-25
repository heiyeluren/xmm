/*
   XMM 简单示例 - 02
   XMM Simple Example - 02

   目标：使用XMM构建一个单链表程序
   Objective: To build a single linked table program using XMM
   
   说明：示例复杂场景中使用XMM内存库
   Description: Example complex scenario using the XMM memory library
*/
package main

import (
	//xmm "xmm/src"
	xmm "github.com/heiyeluren/xmm"
	"fmt"
	"unsafe"
)

//定义一个链表的节点结构
// Define the node structure of a chain table
type XListNode struct {
	Val int
	Next *XListNode
}

//单链表主结构
// Single linked table main structure
type XList struct {
	Head *XListNode
	//Tail *XListNode
}


//初始化链表
//Initialize the chain table
func (l *XList) Init( mm xmm.XMemory ) {
	Node,err := mm.Alloc(unsafe.Sizeof(XListNode{}))
	if err != nil {
		panic("Alloc fail")
	}
	//head := &XListNode{Val:list[0]}
	head := (*XListNode)(Node)
	l.Head = head
	fmt.Println("Init Xlist done")
}

//链表增加节点
//Link table adds nodes
func (l *XList) Append(i int, mm xmm.XMemory) {
	h := l.Head
	for h.Next != nil {
		h = h.Next
	}
	p, err := mm.Alloc(unsafe.Sizeof(XListNode{}))
	if err != nil {
		panic("Alloc fail")
	}
	Node := (*XListNode)(p)
	Node.Val = i
	Node.Next = nil
	h.Next = Node

	fmt.Println("Append item:", Node.Val)
}

//遍历所有链表节点并打印
// iterate through all the chain table nodes and print
func (l *XList) Show() {
	h := l.Head
	//fmt.Println(h.Val)
	for h.Next != nil {
		h = h.Next
		fmt.Println("Show item:", h.Val)
	}
}

//释放整个链表结构
//Releasing the entire linked table structure
func (l *XList) Destroy(mm xmm.XMemory) {
	cnt := 0
	h := l.Head
	//fmt.Println(h.Val)

	//统计需要释放总数
	// Count the total number of releases required
	for h.Next != nil {
		h = h.Next
		cnt++
		//fmt.Println(h.Val)
	}
	//fmt.Println("item count:", cnt)

	//循环释放所有内存
	// Loop to free all memory
	for i := 0; i <= cnt; i++ {
		h := l.Head
		pre := l.Head
		for h.Next != nil {
			pre = h
			h = h.Next
		}
		fmt.Println("Free item:", h.Val)
		mm.Free(uintptr(unsafe.Pointer(h)))
		pre.Next = nil
	}
}

//主函数
// Main functions
func main() {

	//初始化XMM对象
	// Initialise the XMM object
	f := &xmm.Factory{}
	
	//从操作系统申请一个内存块，如果内存使用达到60%，就进行异步自动扩容，每次异步扩容256MB内存（固定值）
	// Request a block of memory from the OS and perform asynchronous auto-expansion if memory usage reaches 60%, 256MB of memory per asynchronous expansion (fixed value)
	mm, err := f.CreateMemory(0.6)
	if err != nil {
		panic("CreateMemory fail ")
	}

	//要生成的数字列表
	// List of numbers to be generated
	list := []int{ 2, 4, 3}

	fmt.Println("\n===== XMM X(eXtensible) Memory Manager example 02 - LinkedList ======\n")


	//初始化链表
	//Initialize LinkedList
	l := &XList{}
	l.Init(mm)
	fmt.Println("")

	//把元素压入链表
	// Pressing elements into a LinkedList
	for i := 0; i < len(list); i++ {
		l.Append(list[i], mm)
	}
	fmt.Println("")

	// Iterate through all the linked table data
	l.Show()
	fmt.Println("")

	//Free all LinkedList memory
	l.Destroy(mm)

	//结束
	//End
	fmt.Println("\n===== Example test success ======\n")
}


