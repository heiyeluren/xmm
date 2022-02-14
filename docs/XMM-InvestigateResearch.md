# XMM 参考调研 - TCMalloc&Go 内存管理调研

- [XMM 参考调研 - TCMalloc&Go 内存管理调研](#xmm-参考调研---tcmallocgo-内存管理调研)
  - [1、调研背景](#1调研背景)
  - [2、TcMalloc 工作机制](#2tcmalloc-工作机制)
    - [数据模型](#数据模型)
  - [3、Go 内存分配机制](#3go-内存分配机制)
    - [3.1、数据模型](#31数据模型)
    - [3.2、内存初始化](#32内存初始化)
    - [3.2、对象申请内存](#32对象申请内存)
    - [相关参考文档](#相关参考文档)

## 1、调研背景

为了解决 Golang 的大内存 GC 问题，需要深入了解 Golang 的内存分配原理，Golang 的内存分配器思想来源于 TCMalloc，他继承了 TCMalloc 的高性能、高内存利用率等优点。实际上与 TCMalloc 有区别的，同时融入了自己的定制化内容。现在我们先了解下 TCMalloc 的实现原理。

说明：TcMalloc - Multi-threaded memory allocate（Goolge 开发的内存分配器）[github.com/google/tcmalloc](https://github.com/google/tcmalloc)

<br />

## 2、TcMalloc 工作机制

<img src=https://raw.githubusercontent.com/heiyeluren/XMM/main/docs/img/ir01.png width=50% />

<br />

在多线程环境下，TCMalloc 可以极大减少锁资源的争夺。针对 small object，TCMalloc 几乎就是 lock free 的。针对 large object，TCMalloc 采用高效的细粒度的自旋锁。同时内存也做了更好管理更加精细化，较少了内存空洞。

<br />

### 数据模型

- Page：page 是一个以 4KB 对齐的内存区域。
- Span：用来管理了连续的 page，将连续的 page 划分到同一个 span 中。
- ThreadCache：线程私有的内存，小于 32k 的对象在该区域分配。
- ClassSpan：根据 class 等级，从 CentralHeap 申请内存，并分级。交给该线程私有内存管理。
- CentralHeap：堆空间。
- 空闲 span 列表： 记录空闲 Span 的结构，用于申请内存和释放内存使用。
- SpanMap：维护了地址到 span 的映射关系。通过地址查询 span 地址，用于 span 合并。

<br />
<br />

## 3、Go 内存分配机制

![这是图片](https://raw.githubusercontent.com/heiyeluren/XMM/main/docs/img/ir02.png)

<br />

### 3.1、数据模型

- heapArena: 保留整个虚拟地址空间。
- arena：用来分配实际的系统内存 arenas(调用系统函数 mmap 申请)
- mspan：是 mheap 上管理的一连串的页，作为 span 的管理者。
- mheap：分配的堆，在页大小为 8KB 的粒度上进行管理，并分配为不同的 span。
- mcentral：通过不同等级 mcentral 搜集了给定大小等级的所有 span，作为 mcache 的预划分。
- mcache：为 per-P 的缓存，协程运行时私有 local heap，内部有不同 class 的 span，作为不同长度的对象保存。

<br />

### 3.2、内存初始化

![这是图片](https://raw.githubusercontent.com/heiyeluren/XMM/main/docs/img/ir03.png)

<br />


### 3.2、对象申请内存

![这是图片](https://raw.githubusercontent.com/heiyeluren/XMM/main/docs/img/ir04.png)

<br />

### 相关参考文档

1. [详解 Go 语言的内存模型及堆的分配管理](https://zhuanlan.zhihu.com/p/76802887)
2. [Go 语言内存管理](https://www.jianshu.com/p/7405b4e11ee2)
