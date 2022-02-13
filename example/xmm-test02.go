/*
   XMM 简单示例 - 02

   目标：使用XMM构建一个单链表程序
   说明：示例复杂场景中使用XMM内存库
*/
package main

import (
	//xmm "xmm/src"
	xmm "github.com/heiyeluren/xmm/src"
	"fmt"
	"unsafe"
)

//定义一个链表的节点结构
type XListNode struct {
	Val int
	Next *XListNode
}

//单链表主结构
type XList struct {
	Head *XListNode
	//Tail *XListNode
}


//初始化链表
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
func (l *XList) Show() {
	h := l.Head
	//fmt.Println(h.Val)
	for h.Next != nil {
		h = h.Next
		fmt.Println("Show item:", h.Val)
	}
}

//释放整个链表结构
func (l *XList) Destroy(mm xmm.XMemory) {
	cnt := 0
	h := l.Head
	//fmt.Println(h.Val)

	//统计需要释放总数
	for h.Next != nil {
		h = h.Next
		cnt++
		//fmt.Println(h.Val)
	}
	//fmt.Println("item count:", cnt)

	//循环释放所有内存
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
func main() {

	//初始化XMM对象
	f := &xmm.Factory{}
	//从操作系统申请一个内存块，如果内存使用达到60%，就进行异步自动扩容，每次异步扩容256MB内存（固定值）
	mm, err := f.CreateMemory(0.6)
	if err != nil {
		panic("CreateMemory fail ")
	}

	//要生成的数字列表
	list := []int{ 2, 4, 3}

	fmt.Println("\n===== XMM X(eXtensible) Memory Manager example 02 - LinkedList ======\n")


	//初始化链表
	l := &XList{}
	l.Init(mm)
	fmt.Println("")

	//把元素压入链表
	for i := 0; i < len(list); i++ {
		l.Append(list[i], mm)
	}
	fmt.Println("")

	//遍历所有链表数据
	l.Show()
	fmt.Println("")

	//释放所有链表内存
	l.Destroy(mm)

	//结束
	fmt.Println("\n===== Example test success ======\n")
}


