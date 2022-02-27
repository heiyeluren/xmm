/*
   XMM Example 00

   Goal: How to use XMM quickly and easily
   Description: An example of how to use the XMM memory library quickly and easily.

   XMM 示例00

   目标：如何简单快速使用XMM
   说明：就是示例如何快速简单使用XMM内存库
*/

package main

import (
	"fmt"
	"unsafe"

	"github.com/heiyeluren/xmm"
)

func main() {

	// Initialising XMM objects
	// 初始化XMM对象
	f := &xmm.Factory{}

	// Request a memory block from the operating system
	// If memory usage reaches 60%, asynchronous automatic expansion is performed, each time 256MB of memory is expanded asynchronously (fixed value), the value of 0.6 can be configured independently
	// 从操作系统申请一个内存块
	// 如果内存使用达到60%，就进行异步自动扩容，每次异步扩容256MB内存（固定值），0.6这个数值可以自主配置
	mm, err := f.CreateMemory(0.6)
	if err != nil {
		panic("CreateMemory fail ")
	}

	// manipulate int types, request memory and assign a value
	// 操作int类型，申请内存后赋值
	var tmpNum int = 9527
	p, err := mm.Alloc(unsafe.Sizeof(tmpNum))
	if err != nil {
		panic("Alloc fail ")
	}
	Id := (*int)(p)
	// Write the set number assignment to the XMM memory
	// 把设定好的数字赋值写入到XMM内存中
	*Id = tmpNum

	// To manipulate string types, XMM provides the From() interface to get a pointer directly and the string will be stored in XMM
	// 操作字符串类型，XMM提供了From()接口，可以直接获得一个指针，字符串会存储在XMM中
	Name, err := mm.From("heiyeluren")
	if err != nil {
		panic("Alloc fail ")
	}

	// Output variables and memory addresses from XMM memory
	// 从XMM内存中输出变量和内存地址
	fmt.Println("\n===== XMM X(eXtensible) Memory Manager example 00 ======\n")
	fmt.Println("\n-- Memory data status --\n")
	fmt.Println(" Id  :", *Id, "\t\t( Id ptr addr: ", Id, ")")
	fmt.Println(" Name:", Name, "\t( Name ptr addr: ", &Name, ")")

	fmt.Println("\n===== Example test success ======\n")

	// Free the Id,Name memory block
	// 释放Id,Name内存块
	mm.Free(uintptr(p))
	mm.FreeString(Name)
}
