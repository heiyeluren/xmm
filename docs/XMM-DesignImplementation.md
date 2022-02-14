
# XMM 的核心设计与实现流程

- [XMM 的核心设计与实现流程](#xmm-的核心设计与实现流程)
  - [设计思考与要求](#设计思考与要求)
  - [1、模块设计图](#1模块设计图)
  - [2、数据结构](#2数据结构)
  - [3、流程图](#3流程图)
    - [3.1、启动分配 Start](#31启动分配-start)
    - [3.2、申请内存 Alloc](#32申请内存-alloc)
    - [3.3、申请 span 流程 Alloc span](#33申请-span-流程-alloc-span)

## 设计思考与要求

XMM - X(eXtensible) Memory Manager（完全自主研发的第三方 Go 内存分配管理器）
XMM 是一个在 Go 语言环境中完全自主实现的第三方内存管理库，不依赖于 Go 本身的任何内存管理能力，纯自主实现的 Go 内存管理库；能够应对各种场景下大小内存的 分配/释放/管理 等工作，能够帮助适用于任何复杂数据结构的构建（链表/数组/树/hash 等场景），能够良好完美的逃逸掉 Go 内置的 GC 机制，保证程序的超高性能，是构建高性能程序基础设施。

XMM 主要特点：

1. XMM 是完全自主研发的内存管理分配器（类似于 TCMalloc/Jemalloc/Ptmalloc 等），大部分场景可以不依赖于 Go 自带的内存管理器，目前在 Golang 方向无完全一样的同类产品。
2. XMM 设计中，能够逃逸掉 Go 内置的 GC 机制，所以程序不会因为 GC 导致应用程序性能卡顿，大幅提高程序运行性能。
3. XMM 分配性能绝佳，目前在比较一般硬件设备的 Linux 系统中，可以达到 350w+ alloc/s（单机每秒可以进行超过 350 万次内存分配操作），丝毫不会让你的程序卡顿。
4. XMM 内存库使用接口简单，兼容性强，能够兼容 Go 1.8 以上版本，容易上手（推荐 go 1.12+ 版本更好）

<br />

为了达成以上的目标，进行了很多内存分配器的调研学习，通过 Golang malloc/tcmalloc 的学习，发现 Golang 有高性能对象分配方式，但是需要对大对象 GC 买单：超大对象的 GC 会带来长时间的 STP。面对我们大数据量的 LocalCache 显然不是那么友好，不能满足我们需求，所以，我们需要设计一个不参与 GC 的高性能内存分配器。
（更多实现细节建议阅读源码）

<br />
<br />

## 1、模块设计图

![这是图片](https://raw.githubusercontent.com/heiyeluren/XMM/main/docs/img/di01.png)

<br />

## 2、数据结构

```go
// 核心堆结构
type xHeap struct {
  lock sync.mutex

  freeChunks *FreeChunkTree  // 红黑树
  rawMemorys rawMemory

  addrMap []*[]*rawMemory  //addr -> page -> rawMemory 关系
  allchunk []*chunk
}
​
​// Span 池
type spanPool struct{
  spans [classSize]*span
  heap *xHeap
}
​
// 写无锁 atomic、扩容必须得全局锁
type span struct{
  lock sync.mutex

  classIndex uint  // class 的索引
  classSize uint   // classSpan 的长度

  startAddr uintptr
  npages    uintptr
  freeIndex uintptr
  fact float32 // 扩容负载因子
}
​
// 连续 page 的管理
type chunk struct {
  startAddr uintptr
  npages    uintptr
}
​
// 用来管理 mmap 申请的内存，用于实际存放地址的元数据
type rawMemory struct {
  addr uintptr
  data []byte
  down bool
  next *rawMemory
  chunks [pagesPerArena]*chunk
}

```

<br />
<br />

## 3、流程图

<br />

### 3.1、启动分配 Start

![这是图片](https://raw.githubusercontent.com/heiyeluren/XMM/main/docs/img/di02.png)

<br />

### 3.2、申请内存 Alloc

![这是图片](https://raw.githubusercontent.com/heiyeluren/XMM/main/docs/img/di03.png)

<br />

### 3.3、申请 span 流程 Alloc span

![这是图片](https://raw.githubusercontent.com/heiyeluren/XMM/main/docs/img/di04.png)

<br />
