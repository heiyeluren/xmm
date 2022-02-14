# XMM参考调研 - Tcmalloc&Go内存管理调研
- [XMM参考调研 - Tcmalloc&Go内存管理调研](#xmm参考调研---tcmallocgo内存管理调研)
    - [1、调研背景](#1调研背景)
    - [2、TcMalloc 工作机制](#2tcmalloc-工作机制)
        - [数据模型](#数据模型)
    - [3、Go内存分配机制](#3go内存分配机制)
      - [3.1、数据模型](#31数据模型)
      - [3.2、内存初始化](#32内存初始化)
      - [3.2、对象申请内存](#32对象申请内存)
      - [相关参考文档：](#相关参考文档)
<br />

### 1、调研背景

为了解决golang的大内存GC问题，需要深入了解Golang的内存分配原理，Golang的内存分配器思想来源于tcmalloc，他继承了TcMalloc的高性能、高内存利用率等优点。实际上与tcmalloc有区别的，同时融入了自己的定制化内容。现在我们先了解下TcMalloc的实现原理。

说明：TcMalloc - Multi-threaded memory allocate（Goolge开发的内存分配器）

<br />

### 2、TcMalloc 工作机制

<img src=https://raw.githubusercontent.com/heiyeluren/XMM/main/docs/img/ir01.png width=50% />

<br />

在多线程环境下，TCMalloc可以极大减少锁资源的争夺。针对small object，TCMalloc几乎就是lock free的。针对large object，TCMalloc采用高效的细粒度的自旋锁。同时内存也做了更好管理更加精细化,较少了内存空洞。

<br />

##### 数据模型
- Page：page是一个以4KB对齐的内存区域。
- Span：用来管理了连续的page，将连续的page划分到同一个span中。
- ThreadCache：线程私有的内存，小于32k的对象在该区域分配。
- ClassSpan：根据class等级，从CentralHeap申请内存，并分级。交给该线程私有内存管理。
- CentralHeap：堆空间。
- 空闲span列表： 记录空闲Span的结构，用于申请内存和释放内存使用。
- SpanMap：维护了地址到span的映射关系。通过地址查询span地址，用于span合并。

<br />
<br />

### 3、Go内存分配机制

![这是图片](https://raw.githubusercontent.com/heiyeluren/XMM/main/docs/img/ir02.png)

<br />

#### 3.1、数据模型
- heapArena: 保留整个虚拟地址空间。
- arena：用来分配实际的系统内存arenas(调用系统函数mmap申请)
- mspan：是 mheap 上管理的一连串的页，作为span的管理者。
- mheap：分配的堆，在页大小为 8KB 的粒度上进行管理，并分配为不同的span。
- mcentral：通过不同等级mcentral搜集了给定大小等级的所有 span，作为mcache的预划分。
- mcache：为 per-P 的缓存，协程运行时私有local heap，内部有不同class的span，作为不同长度的对象保存。

<br />

#### 3.2、内存初始化
![这是图片](https://raw.githubusercontent.com/heiyeluren/XMM/main/docs/img/ir03.png)

<br />


#### 3.2、对象申请内存
![这是图片](https://raw.githubusercontent.com/heiyeluren/XMM/main/docs/img/ir04.png)

<br />

#### 相关参考文档：

1. [详解Go语言的内存模型及堆的分配管理](https://zhuanlan.zhihu.com/p/76802887)
2. [Go语言内存管理](https://www.jianshu.com/p/7405b4e11ee2)

<br />
