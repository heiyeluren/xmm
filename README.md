# XMM (eXtensible) Memory Manager - 完全自主第三方Go内存分配管理器
<br />
<br />

### XMM是什么？

XMM - X(eXtensible) Memory Manager（完全自主研发的第三方Go内存分配管理器）

XMM是一个在Go语言环境中完全自主实现的第三方内存管理库，不依赖于Go本身的任何内存管理能力，纯自主实现的Go内存管理库；能够应对各种场景下大小内存的 分配/释放/管理 等工作，能够帮助适用于任何复杂数据结构的构建（链表/数组/树/hash等场景），能够良好完美的逃逸吊Go内置的GC机制，保证程序的超高性能，是构建高性能程序基础设施。

<br />

### XMM主要具备以下特点：
1.  XMM是完全自主研发的内存管理分配器（类似于 TCMalloc/Jemalloc/Ptmalloc 等），大部分场景可以不依赖于Go自带的内存管理器，目前在Golang方向无完全一样的同类产品。
2.  XMM设计中，能够逃逸掉Go内置的GC机制，所以程序不会因为GC导致应用程序性能卡顿，大幅提高程序运行性能。
3. XMM分配性能绝佳，目前在比较一般硬件设备的Linux系统中，可以达到 350w+ alloc/s（单机每秒可以进行超过350万次内存分配操作），丝毫不会让你的程序卡顿。
4. XMM内存库使用接口简单，兼容性强，能够兼容Go 1.8 以上版本，容易上手（推荐 go 1.12+版本更好）

<br />
<br />
<br />

## 为什么要设计XMM？
<br />

### 为什么要设计自主的内存管理器？

为了应对在多种内存管理的场景下的使用，可能需要有一些除了内置数据结构外的一些内存自主调度使用的场景，比如构建复杂的高性能的数据结构，在大规模内存占用，或者是非常多的小内存块占用场景下，能够尽量减少Go的GC机制，保障服务性能稳定不会应为GC而产生抖动。

<br />


### 为什么不使用内置的 map/slice 等数据结构？

Golang本身为了性能和内存可控，整个内存管理是完全封闭不对外的，并且有自主的Gc机制，需要自主内存管理比较麻烦；Go中自带的Gc机制经过很多个版本的迭代，到目前性能已经很不错，但是在大规模的碎片化内存块下面，GC还是会有一定损耗，在极端高性能场景下，GC会让整个后台应用服务性能上不去（或偶尔卡顿）。所以一句话，Go本身指针等还有性能会受到GC的影响，导致服务性能总是上不去。
<br />

### 为什么不使用其他开源的内存池？

1. 除Go本身的内存模块，调研了解现有大部分的第三方 对象池/内存池/字节池 等需要某块自主内存操作的场景中基本是Map/sync.Pool/Bytes[] 等方式。

2. Map数据结构适合保存各类型数据，但GC概率大； sync.Pool 这类保存复用临时对象，也可以各种数据机构，可适当减少GC（无法避免GC）； Bytes[] 方式来保存字节数据，并且只能保存字节数据，通过某些处理，尽量逃避GC扫描；（对比参考 [Go语言基于channel实现的并发安全的字节池](https://zhuanlan.zhihu.com/p/265790840) ）

3. 现有开源库包括：依赖于 sync.Pool 的比如字节的mcache [gopkg/mcache.go](https://github.com/bytedance/gopkg/blob/main/lang/mcache/mcache.go) ；采用Bytes[]方式的比如MinIO 的 [bpool minio/bpool.go](https://github.com/minio/minio/blob/master/internal/bpool/bpool.go) ，都可以学习参考。

4. 结论：XMM与他们实现机制完全不同，XMM更靠近Go内置内存分配机制原理

<br />

### XMM的最终设计结论是什么？

为了完全实现最终为了逃逸掉Golang的Gc机制，以及拥有完全自主可控的内存管理分配操作，在面对成千上万的小对象场景中，不会应为Go本身GC机制带来任何的抖动，所以自主从零开始实现了XMM模块，达到在Go程序中调用XMM模块可以达到完美的自主内存 申请/释放/管理 的功能，并可以完美逃逸掉Go的GC机制。
<br />
<br />

### XMM设计的目标是什么？
为了保证高性能，XMM设计之初，就定下了三个核心目标：

1. 单机（6核心KVM或物理机）内存分配性能达到 350w+ alloc/s；（每秒内存分配速度）；

2. 可以支持调用用户手工强制Free某个申请内存块，也可以支持XMM自身自动Gc某些未手工Free的内存库；（自主实现Gc功能）

3. 不会内存泄露，并且内存管理不是粗糙的，而颗粒度细致的，完全尽量可媲美行业主流的内存管理分配器。


<br />
<br />
<br />

## XMM如何使用？

<br /><br /><br />

### XMM使用示例一：
XMM的使用非常简单方便，我们直接通过代码来看。

示例一：看一个快速入门的例子，简单常用变量采用XMM进行存储

```go
/*
    XMM 示例00

    目标：如何简单快速使用XMM
    说明：就是示例如何快速简单使用XMM内存库
*/

package main

import (
	xmm "xmm/src"
	"fmt"
	"unsafe"
	//_ "github.com/spf13/cast"
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


```



## XMM问题反馈

XMM 目前是0.1版本，总体性能比较好，目前也在另外一个自研的XMap模块中使用，当然也少不了一些问题和bug，欢迎大家一起共创，或者直接提交PR等等。

欢迎加入XMM技术交流微信群，要加群，可以先添加如下微信让对方拉入群：


![image](https://raw.githubusercontent.com/heiyeluren/docs/master/imgs/koala_wx.png)



