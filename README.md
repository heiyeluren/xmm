
<img src=https://raw.githubusercontent.com/heiyeluren/XMM/main/docs/img/xmm-logo02.png width=50% />
<p>
<a href="https://sourcegraph.com/github.com/heiyeluren/XMM?badge"><img src="https://sourcegraph.com/github.com/heiyeluren/XMM/-/badge.svg" alt="GoDoc"></a>
<a href="https://pkg.go.dev/github.com/heiyeluren/XMM"><img src="https://pkg.go.dev/badge/github.com/heiyeluren/XMM" alt="GoDoc"></a>
<a href="https://goreportcard.com/report/github.com/heiyeluren/XMM"><img src="https://goreportcard.com/badge/github.com/heiyeluren/XMM" alt="Go Report Card" /a>
</p>


## ➤ [XMM English Introduction Document](https://github.com/heiyeluren/XMM/blob/main/docs/XMM-INTRO-EN.md)


<br />
<br />


## ➤ XMM中文文档
## XMM (eXtensible) Memory Manager - 完全自主第三方 Go 内存分配管理器

- [XMM (eXtensible) Memory Manager - 完全自主第三方 Go 内存分配管理器](#xmm-extensible-memory-manager---完全自主第三方-go-内存分配管理器)
  - [XMM 是什么？](#xmm-是什么)
    - [XMM 主要具备以下特点](#xmm-主要具备以下特点)
  - [为什么要设计 XMM？](#为什么要设计-xmm)
    - [为什么要设计自主的内存管理器？](#为什么要设计自主的内存管理器)
    - [为什么不使用内置的 map/slice 等数据结构？](#为什么不使用内置的-mapslice-等数据结构)
    - [为什么不使用其他开源的内存池？](#为什么不使用其他开源的内存池)
    - [XMM 的最终设计结论是什么？](#xmm-的最终设计结论是什么)
    - [XMM 设计的目标是什么？](#xmm-设计的目标是什么)
  - [XMM 快速使用入门](#xmm-快速使用入门)
    - [☆ XMM 使用案例 ☆](#-xmm-使用案例-)
  - [XMM 实现原理介绍](#xmm-实现原理介绍)
    - [XMM 技术交流](#xmm-技术交流)




<br />

## XMM 是什么？

XMM - X(eXtensible) Memory Manager（完全自主研发的第三方 Go 内存分配管理器）

XMM 是一个在 Go 语言环境中完全自主实现的第三方内存管理库，不依赖于 Go 本身的任何内存管理能力，纯自主实现的 Go 内存管理库；能够应对各种场景下大小内存的 分配/释放/管理 等工作，能够帮助适用于任何复杂数据结构的构建（链表/数组/树/hash 等场景），能够良好完美的逃逸掉 Go 内置的 GC 机制，保证程序的超高性能，是构建高性能程序基础设施。

<br />

### XMM 主要具备以下特点

1. XMM 是一个在 Go 语言环境中完全自主实现的第三方内存管理库，不依赖于 Go 本身的任何内存管理能力，通过 6000 行纯 Go 代码自主实现的 Go 内存管理库，适合不用 Go 的 GC 想要自己管理内存的场景。

2. XMM 能够应对各种场景下大小内存的 分配/释放/管理 等工作，能够帮助适用于任何复杂数据结构的构建，比如链表/数组/树/哈希表等等场景；XMM 可以让你像 C/C++ 一样方便便捷使用系统内存，并且不用担心性能问题。

3. XMM 能够良好完美的逃逸掉 Go 内置的 GC 机制，保证程序的超高性能，是构建高性能程序基础设施；但与 sync.Pool 等实现机制完全不同，sync.Pool 等使用字节流实现来逃逸 GC，XMM 是纯使用 Linux 系统的 mmap 作为底层内存存储，XMM 更像 TcMalloc 等内存分配器。

4. XMM 协程安全，且分配性能超高，目前在普通 Linux 服务器上面可以达到 350w alloc/s，就是每秒可以进行 350 万次的内存分配操作不卡顿，非常适合想要自主管理内存且超高性能场景。

5. XMM 内存库使用接口简单，兼容性强，能够兼容 Go 1.8 以上版本，容易上手（推荐 go 1.12+ 版本更好），可以在 XMM 之上重构你所有想要的高性能数据结构，比如 map/slice 等等。（案例部分可以做一些数据结构实现的参考）

<br />
<br />

## 为什么要设计 XMM？

<br />

### 为什么要设计自主的内存管理器？

为了应对在多种内存管理的场景下的使用，可能需要有一些除了内置数据结构外的一些内存自主调度使用的场景，比如构建复杂的高性能的数据结构，在大规模内存占用，或者是非常多的小内存块占用场景下，能够尽量减少 Go 的 GC 机制，保障服务性能稳定不会因为 GC 而产生抖动。

<br />


### 为什么不使用内置的 map/slice 等数据结构？

Golang 本身为了性能和内存可控，整个内存管理是完全封闭不对外的，并且有自主的 GC 机制，需要自主内存管理比较麻烦；Go 中自带的 GC 机制经过很多个版本的迭代，到目前性能已经很不错，但是在大规模的碎片化内存块下面，GC 还是会有一定损耗，在极端高性能场景下，GC 会让整个后台应用服务性能上不去（或偶尔卡顿）。所以一句话，Go 本身指针等还有性能会受到 GC 的影响，导致服务性能总是上不去。
<br />

### 为什么不使用其他开源的内存池？

1. 除 Go 本身的内存模块，调研了解现有大部分的第三方 对象池/内存池/字节池 等需要某块自主内存操作的场景中基本是 Map/sync.Pool/Bytes[] 等方式。

2. Map 数据结构适合保存各类型数据，但 GC 概率大； sync.Pool 这类保存复用临时对象，也可以各种数据机构，可适当减少 GC（无法避免 GC）； Bytes[] 方式来保存字节数据，并且只能保存字节数据，通过某些处理，尽量逃避 GC 扫描；（对比参考 [Go 语言基于 channel 实现的并发安全的字节池](https://zhuanlan.zhihu.com/p/265790840) ）

3. 现有开源库包括：依赖于 sync.Pool 的比如字节的 mcache [gopkg/mcache.go](https://github.com/bytedance/gopkg/blob/main/lang/mcache/mcache.go)；采用 Bytes[] 方式的比如 MinIO 的的 [bpool minio/bpool.go](https://github.com/minio/minio/blob/master/internal/bpool/bpool.go) ，都可以学习参考。

4. 结论：XMM 与他们实现机制完全不同，XMM 更靠近 Go 内置内存分配机制原理

<br />

### XMM 的最终设计结论是什么？

为了完全实现最终为了逃逸掉 Golang 的 GC 机制，以及拥有完全自主可控的内存管理分配操作，在面对成千上万的小对象场景中，不会因为 Go 本身 GC 机制带来任何的抖动，所以自主从零开始实现了 XMM 模块，达到在 Go 程序中调用 XMM 模块可以达到完美的自主内存 申请/释放/管理 的功能，并可以完美逃逸掉 Go 的 GC 机制。
<br />
<br />

### XMM 设计的目标是什么？
为了保证高性能，XMM 设计之初，就定下了三个核心目标：

1. 单机（6 核心 KVM 或物理机）内存分配性能达到 350w+ alloc/s；（每秒内存分配速度）；

2. 可以支持调用用户手工强制Free某个申请内存块，也可以支持XMM自身自动GC某些未手工Free的内存库；（自主实现GC功能）

3. 不会内存泄露，并且内存管理不是粗糙的，而颗粒度细致的，完全尽量可媲美行业主流的内存管理分配器。


<br />
<br />

## XMM 快速使用入门

### ☆ [XMM 使用案例](https://github.com/heiyeluren/XMM/blob/main/docs/XMM-Usage.md) ☆

<br />

说明：XMM 测试程序快速预览下载使用

1. [XMM 使用 - 入门](https://github.com/heiyeluren/XMM/blob/main/example/xmm-test00.go)
2. [XMM 使用 - 结构体](https://github.com/heiyeluren/XMM/blob/main/example/xmm-test01.go)
3. [XMM 使用 - 链表](https://github.com/heiyeluren/XMM/blob/main/example/xmm-test02.go)
4. [XMM 使用 - 哈希表](https://github.com/heiyeluren/XMM/blob/main/example/xmm-test03.go)

<br /> <br />


## XMM 实现原理介绍

1. [XMM 的核心设计与实现流程](https://github.com/heiyeluren/XMM/blob/main/docs/XMM-DesignImplementation.md)
1. [XMM 设计实现技术调研参考](https://github.com/heiyeluren/XMM/blob/main/docs/XMM-InvestigateResearch.md)

<br />

### XMM 项目开发者

| 项目角色      | 项目成员 |
| ----------- | ----------- |
| 项目发起人/负责人      | 黑夜路人( @heiyeluren ) <br />老张 ( @Zhang-Jun-tao )       |
| 项目开发者   | 老张 ( @Zhang-Jun-tao ) <br />黑夜路人( @heiyeluren ) <br /> Viktor ( @guojun1992 )        |

<br /> <br />

### XMM 技术交流

XMM 目前是早期版本，总体性能比较好，目前也在另外一个自研的 XMap 模块中使用，当然也少不了一些问题和 bug，欢迎大家一起共创，或者直接提交 PR 等等。

欢迎加入 XMM 技术交流微信群，要加群，可以先添加如下微信让对方拉入群：<br />
（如无法看到图片，请手工添加微信： heiyeluren2017 ）

<img src=https://raw.githubusercontent.com/heiyeluren/xmm/main/docs/img/heiyeluren2017-wx.jpg width=40% />



<br />
<br />
