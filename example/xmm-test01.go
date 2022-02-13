/*
   XMM 简单示例 - 01

   目标：如何在结构体中使用XMM
   说明：示例如何结构体类场景如何使用XMM内存库
*/
package main

import (
	//xmm "xmm/src"
	xmm "github.com/heiyeluren/xmm/src"
	"fmt"
	"unsafe"
)

func main() {

	//定义一个类型(结构体)
	type User struct {
		Id     uint
		Name   string
		Age    uint
		Email  string
		Salary float32
	}

	//初始化XMM对象
	f := &xmm.Factory{}
	//从操作系统申请一个内存块
	//如果内存使用达到60%，就进行异步自动扩容，每次异步扩容256MB内存（固定值），0.6这个数值可以自主配置
	mm, err := f.CreateMemory(0.6)
	if err != nil {
		panic("CreateMemory fail ")
	}

	//自己从内存块中申请一小段自己想用的内存
	size := unsafe.Sizeof(User{})
	p, err := mm.Alloc(size)
	if err != nil {
		panic("Alloc fail ")
	}

	//使用该内存块，进行结构体元素赋值
	user := (*User)(p)
	user.Id		= 1
	user.Age	= 18
	user.Name	= "heiyeluren"
	user.Email	= "heiyeluren@example.com"

	//输出变量，打印整个结构体等
	fmt.Println("\n===== XMM X(eXtensible) Memory Manager example 01 ======\n")

	fmt.Println("\n-- Memory data status --\n")
	fmt.Println("User ptr addr: \t", p)
	fmt.Println("User data: \t",     user)

	//释放内存块（实际是做mark标记操作）
	mm.Free(uintptr(p))

	//Free()后再看看变量值，只是针对这个内存块进行mark标记动作，并未彻底从内存中释放（XMM设计机制，降低实际gc回收空闲时间）
	//XMM内部会有触发gc的机制，主要是内存容量，参数TotalGCFactor=0.0004，目前如果要配置，需要自己修改这个常量，一般不用管它，Free()操作中有万分之4的概率会命中触发gc~
	//GC触发策略：待释放内存  > 总内存 * 万分之4 会触发gc动作
	fmt.Println("\n-- Memory data status after XMM.Free() --\n")
	fmt.Println("memory ptr addr:\t", p)
	fmt.Println("User data:\t\t",      user)

	//结束
	fmt.Println("\n===== Example test success ======\n")
}

