
## XMM Benchmark

### 机器配置：

* Linux 3.10.0-957.21.3.el7.x86_64   		   8G    		   4核(MHz: 2194.842)

### 测试工具

* Golang test


### 测试程序

- [分配性能测试程序](https://github.com/heiyeluren/xmm/blob/main/benchmark/alloc_test.go)
- [GC性能测试程序](https://github.com/heiyeluren/xmm/blob/main/benchmark/gc_test.go)


<br />



## XMM性能测试结果数据

<br />

### 1. 对象内存分配(24B) -- 小对象内存

适用于Xmm分配的内存作为临时存放。例如：比较多的小对象分配。分别使用Xmm和golang来分配24B的结构体空间。`BenchmarkAlloc_`

|      | 响应时长  | Qps      |分配/秒|
| ---- | --------- | -------- | ------- |
| Xmm  | 60 ns/op  | 16666666 |  1666万次/秒 |
| Go   | 130 ns/op | 7692307  | 769万次/秒 |   


<br />


### 2. 对象内存分配(85K) -- 大对象内存

​    适用于Xmm做为大内存分配器,可以作为临时内存来使用。`BenchmarkGcLargeMemory_`

|      | 响应时长     | Qps  |
| ---- | ------------ | ---- |
| Xmm  | 139333 ns/op | 7177 |
| Go   | 210860 ns/op | 4742 |


<br />


### 3. 对象内存分配(24B和85K按照1:1000)

​    适用于Xmm做为永久内存管理，存放较大的内存块。24B利用golang自动分配内存管理，85K利用Xmm分配内存，例如：使用Xmm分配出一个localcache内存，作为业务缓存内存。`BenchmarkGc_`

|      | 响应时长   | Qps     | 分配/秒 |
| ---- | ---------- | ------- |--------|
| Xmm  | 762 ns/op  | 1312335 | 131万次/秒 |
| Go   | 8367 ns/op | 119517  | 11万次/秒 |


<br />


### 4. GC STW测试

​    Xmm致力于减少GC STW时间，分别使用Go和Xmm分配850M内存，然后通过构建一个线上的内存分配模型，持续运行五分钟，统计其GC暂停时间评测性能。例如：用来解决Golang构建大内存对服务gc的压力。`TestGcPauseTotal_`

|      | PauseTotalNs | NumGC |
| ---- | ------------ | ----- |
| Xmm  | 6894426      | 743   |
| Go   | 8023804      | 757   |


<br />



### 附录

```sh
# 1. 对象内存分配(24B)
go test -v -bench=BenchmarkAlloc_Go -benchtime=2s alloc_test.go
go test -v -bench=BenchmarkAlloc_Xmm -benchtime=2s alloc_test.go



# 2. 对象内存分配(85K)
go test -v -bench=BenchmarkGcLargeMemory_Xmm -benchtime=2s gc_test.go
go test -v -bench=BenchmarkGcLargeMemory_Go -benchtime=2s gc_test.go


# 3. 对象内存分配(24B和85K按照1:1000)
go test -v -bench=BenchmarkGc_Xmm -benchtime=2s gc_test.go
go test -v -bench=BenchmarkGc_Go -benchtime=2s gc_test.go



# 4. GC STW测试
go test -run=TestGcPauseTotal_Xmm gc_test.go
go test -run=TestGcPauseTotal_Go gc_test.go
```



