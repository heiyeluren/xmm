/*
    XMM 示例00

    目标：如何简单快速使用XMM
    说明：就是示例如何快速简单使用XMM内存库
*/

package main

import (
	//xmm "xmm/src"
	xmm "github.com/heiyeluren/xmm/src"
	"fmt"
	"unsafe"
)

func main() {

	//初始化XMM对象
	f := &xmm.Factory{}

	//从操作系统申请一个内存块
	//如果内存使用达到60%，就进行异步自动扩容，每次异步扩容256MB内存（固定值），0.6这个数值可以自主配置
	mm, err := f.CreateMemory(0.6)
	if err != nil {
		panic("CreateMemory fail ")
	}

	//操作int类型，申请内存后赋值
	var tmpNum int = 9527
	p, err := mm.Alloc(unsafe.Sizeof(tmpNum))
	if err != nil {
		panic("Alloc fail ")
	}
	Id := (*int)(p)
	//把设定好的数字赋值写入到XMM内存中
	*Id = tmpNum

	//操作字符串类型，XMM提供了From()接口，可以直接获得一个指针，字符串会存储在XMM中
	Name, err := mm.From("heiyeluren")
	if err != nil {
			panic("Alloc fail ")
	}

	//从XMM内存中输出变量和内存地址
	fmt.Println("\n===== XMM X(eXtensible) Memory Manager example 00 ======\n")
	fmt.Println("\n-- Memory data status --\n")
	fmt.Println(" Id  :", *Id,  "\t\t( Id ptr addr: ", Id, ")"   )
	fmt.Println(" Name:", Name, "\t( Name ptr addr: ", &Name, ")")

	fmt.Println("\n===== Example test success ======\n")

	//释放Id,Name内存块
	mm.Free(uintptr(p))
	mm.FreeString(Name)
}

