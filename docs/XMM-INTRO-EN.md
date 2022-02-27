<img src=https://raw.githubusercontent.com/heiyeluren/XMM/main/docs/img/xmm-logo02.png width=50% />

# XMM (eXtensible) Memory Manager - High performance Go memory manager

- [XMM (eXtensible) Memory Manager - fully-autonomous-third-party-go-memory-allocation-manager]
    - What is [XMM?] (#xmm-what-is-it)
        - [XMM has the following main features]
    - [Why design XMM?]
        - [Why design an autonomous memory manager?]
        - [Why not use built-in data structures like map/slice?]
        - [Why not use other open source memory pools?]
        - [What is the final design conclusion of XMM?]
        - [What are the goals of the XMM design?]
    - [XMM Quick Start]
        - [☆ XMM Use Cases ☆]
    - [XMM Implementation Principles]
        - [XMM Technology Exchange]

<br />

## What is XMM?

XMM - X(eXtensible) Memory Manager (high performance third party Go memory allocation manager)

XMM is a third-party memory management library implemented in the Go language environment, which does not rely on any
memory management capabilities of Go itself, and is a purely independent Go memory management library; it can handle the
allocation/release/management of large and small memory in various scenarios, and can help with the construction of any
complex data structures (chains/arrays/trees/hashes, etc.), and can escape Go's built-in GC mechanism well and perfectly
to ensure the ultra-high performance of programs, which is the infrastructure for building high-performance programs.


<br />

### XMM Key Features

1. XMM is a third-party memory management library implemented in the Go language environment, which does not rely on any
   memory management capabilities of Go itself, and is implemented in 6000 lines of pure Go code.

2. XMM can handle the allocation/release/management of memory in various scenarios, and can help build complex data
   structures such as chained tables/arrays/trees/hash tables, etc. XMM allows you to use system memory as easily and
   conveniently as C/C++, without worrying about performance.

3. XMM is a good and perfect way to escape Go's built-in GC mechanism to ensure high performance and is the
   infrastructure for building high performance programs; however, unlike sync. XMM is more like a memory allocator such
   as TcMalloc. 4.

4. XMM is process-safe and has very high allocation performance, currently reaching 350w alloc/s on a normal Linux
   server, which means that it can perform 3.5 million memory allocation operations per second without lagging, making
   it ideal for scenarios where you want to manage memory autonomously and with high performance.

5. The XMM memory library has a simple interface and is compatible with Go 1.8 and above, so it is easy to get started (
   go 1.12+ is recommended) and can be used to reconstruct all the high-performance data structures you want on top of
   XMM, such as map/slice and so on. (The examples section can be used as a reference for some data structure
   implementations)

<br />
<br />

## Why develop XMM?

<br />

### Why design a third party memory manager?

In order to cope with a variety of memory management scenarios, it may be necessary to have some scenarios where memory
is used autonomously in addition to the built-in data structures, such as building complex high performance data
structures, minimising Go's GC mechanism in the case of large memory footprint, or a very large number of small memory
blocks, and ensuring stable service performance without jitter due to GC.

<br />

### Why not use Go's built-in data structures like map/slice?

Golang itself, for the sake of performance and memory control, is completely closed to the public, and has its own GC
mechanism, so it is more troublesome to manage memory independently; the GC mechanism that comes with Go has been
iterated over many versions, and its performance is already very good so far, but in a large-scale fragmented memory
block, GC will still have some loss, and in extreme high-performance scenarios, GC will In extreme high performance
scenarios, GC can cause the entire background application service to underperform (or occasionally stall). So in a
nutshell, Go itself is affected by GC in terms of performance of pointers and so on, which always leads to poor
performance of the service.

<br />

### Why not use other open source memory pools?

1. in addition to Go's own memory module, research understands that most of the existing third-party object pools/memory
   pools/byte pools and other scenarios that require a certain piece of autonomous memory operations are basically in
   the form of Map/sync.

2. Pool is suitable for storing various types of data, but has a high probability of GC; sync.Pool is suitable for
   storing reused temporary objects, and can also be used for various data bodies, which can reduce GC appropriately (it
   cannot avoid GC); Bytes[] is suitable for storing byte data, and can only store byte data, which can avoid GC
   scanning as much as possible through certain processing; (compare
   with [Go language's byte pool based on channel-based implementation of concurrency-safe byte pooling](https://zhuanlan.zhihu.com/p/265790840))

3. existing open source libraries include:
   mcache [gopkg/mcache.go](https://github.com/bytedance/gopkg/blob/main/lang/mcache/mcache.go), which relies on
   sync.Pool, for example, and [Bytes[]], which uses Bytes[]
   Pool [bpool minio/bpool.go](https://github.com/minio/minio/blob/master/internal/bpool/bpool.go), for example,
   and [bpool minio/bpool.go](https://github.com/minio/minio/blob/master/internal/bpool/bpool.go) for MinIO.

4. Conclusion: XMM is completely different from their implementation mechanism, XMM is closer to Go's built-in memory
   allocation mechanism principle

<br />

### What is the final design conclusion of the XMM?

The XMM module has been implemented from scratch in order to escape Golang's GC mechanism and to have a fully autonomous
memory management operation, without any jitter caused by Go's own GC mechanism in the case of thousands of small
objects.
<br />
<br />

### What are the XMM design goals?

To ensure high performance, the XMM was designed with three core objectives in mind.

1. a single machine (6-core KVM or physical machine) memory allocation performance of 350w+ alloc/s; (memory allocation
   speed per second).

2. support for calling the user to manually force a block of memory to be free, or support for XMM itself to
   automatically GC some memory banks that are not manually free; (autonomous implementation of GC function)

3. no memory leaks, and memory management is not crude, but granular, fully comparable to the industry's mainstream
   memory management allocators.

<br />
<br />

## XMM Quick Start

### ☆ [XMM Use Case](https://github.com/heiyeluren/XMM/blob/main/docs/XMM-Usage-EN.md) ☆

<br />

Description: A quick preview of the XMM test program to download and use

1. [Using XMM - Getting Started](https://github.com/heiyeluren/XMM/blob/main/example/xmm-test00.go)
2. [XMM Usage - Structs](https://github.com/heiyeluren/XMM/blob/main/example/xmm-test01.go)
3. [XMM Usage - Linked Tables](https://github.com/heiyeluren/XMM/blob/main/example/xmm-test02.go)
4. [XMM Usage - Hash Tables](https://github.com/heiyeluren/XMM/blob/main/example/xmm-test03.go)

<br /> <br />

## Introduction to the principle of XMM implementation

1. [Core Design and Implementation Process of XMM](https://github.com/heiyeluren/XMM/blob/main/docs/XMM-DesignImplementation.md)
1. [XMM design and implementation technology research reference](https://github.com/heiyeluren/XMM/blob/main/docs/XMM-InvestigateResearch.md)

<br /> <br />

### XMM Technology Exchange Community

XMM is currently an early version, the overall performance is relatively good, and is also currently used in another
self-developed XMap module, of course, there are also some problems and bugs, welcome everyone to create together, you
can submit issues and PR, etc..

You can also send emails to the author for communication, and if you are convenient to use WeChat, you can add the
author's WeChat.

<br />

#### Author's email: heiyeluren@gmail.com / heiyeluren@qq.com

#### Author's WeChat: (swipe to add)

<br />
<img src=https://raw.githubusercontent.com/heiyeluren/XMM/main/docs/img/xmm-wx.png width=40% />

<br /><br />
<br />
