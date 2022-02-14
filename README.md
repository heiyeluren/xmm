# XMM (eXtensible) Memory Manager - 完全自主第三方Go内存分配管理器
<br />
<br />

#### XMM是什么？

XMM - X(eXtensible) Memory Manager（完全自主研发的第三方Go内存分配管理器）

XMM是一个在Go语言环境中完全自主实现的第三方内存管理库，不依赖于Go本身的任何内存管理能力，纯自主实现的Go内存管理库；能够应对各种场景下大小内存的 分配/释放/管理 等工作，能够帮助适用于任何复杂数据结构的构建（链表/数组/树/hash等场景），能够良好完美的逃逸吊Go内置的GC机制，保证程序的超高性能，是构建高性能程序基础设施。

<br />

#### XMM主要具备以下特点：
1.  XMM是完全自主研发的内存管理分配器（类似于 TCMalloc/Jemalloc/Ptmalloc 等），大部分场景可以不依赖于Go自带的内存管理器，目前在Golang方向无完全一样的同类产品。
2.  XMM设计中，能够逃逸掉Go内置的GC机制，所以程序不会因为GC导致应用程序性能卡顿，大幅提高程序运行性能。
3. XMM分配性能绝佳，目前在比较一般硬件设备的Linux系统中，可以达到 350w+ alloc/s（单机每秒可以进行超过350万次内存分配操作），丝毫不会让你的程序卡顿。
4. XMM内存库使用接口简单，兼容性强，能够兼容Go 1.8 以上版本，容易上手（推荐 go 1.12+版本更好）

<br />
<br />
<br />

## 为什么要设计XMM？
<br />

#### 为什么要设计自主的内存管理器？

为了应对在多种内存管理的场景下的使用，可能需要有一些除了内置数据结构外的一些内存自主调度使用的场景，比如构建复杂的高性能的数据结构，在大规模内存占用，或者是非常多的小内存块占用场景下，能够尽量减少Go的GC机制，保障服务性能稳定不会应为GC而产生抖动。

<br />


#### 为什么不使用内置的 map/slice 等数据结构？

Golang本身为了性能和内存可控，整个内存管理是完全封闭不对外的，并且有自主的Gc机制，需要自主内存管理比较麻烦；Go中自带的Gc机制经过很多个版本的迭代，到目前性能已经很不错，但是在大规模的碎片化内存块下面，GC还是会有一定损耗，在极端高性能场景下，GC会让整个后台应用服务性能上不去（或偶尔卡顿）。所以一句话，Go本身指针等还有性能会受到GC的影响，导致服务性能总是上不去。
<br />

#### 为什么不使用其他开源的内存池？

1. 除Go本身的内存模块，调研了解现有大部分的第三方 对象池/内存池/字节池 等需要某块自主内存操作的场景中基本是Map/sync.Pool/Bytes[] 等方式。

2. Map数据结构适合保存各类型数据，但GC概率大； sync.Pool 这类保存复用临时对象，也可以各种数据机构，可适当减少GC（无法避免GC）； Bytes[] 方式来保存字节数据，并且只能保存字节数据，通过某些处理，尽量逃避GC扫描；（对比参考 [Go语言基于channel实现的并发安全的字节池](https://zhuanlan.zhihu.com/p/265790840) ）

3. 现有开源库包括：依赖于 sync.Pool 的比如字节的mcache [gopkg/mcache.go](https://github.com/bytedance/gopkg/blob/main/lang/mcache/mcache.go) ；采用Bytes[]方式的比如MinIO 的 [bpool minio/bpool.go](https://github.com/minio/minio/blob/master/internal/bpool/bpool.go) ，都可以学习参考。

4. 结论：XMM与他们实现机制完全不同，XMM更靠近Go内置内存分配机制原理

<br />

#### XMM的最终设计结论是什么？

为了完全实现最终为了逃逸掉Golang的Gc机制，以及拥有完全自主可控的内存管理分配操作，在面对成千上万的小对象场景中，不会应为Go本身GC机制带来任何的抖动，所以自主从零开始实现了XMM模块，达到在Go程序中调用XMM模块可以达到完美的自主内存 申请/释放/管理 的功能，并可以完美逃逸掉Go的GC机制。
<br />
<br />

#### XMM设计的目标是什么？
为了保证高性能，XMM设计之初，就定下了三个核心目标：

1. 单机（6核心KVM或物理机）内存分配性能达到 350w+ alloc/s；（每秒内存分配速度）；

2. 可以支持调用用户手工强制Free某个申请内存块，也可以支持XMM自身自动Gc某些未手工Free的内存库；（自主实现Gc功能）

3. 不会内存泄露，并且内存管理不是粗糙的，而颗粒度细致的，完全尽量可媲美行业主流的内存管理分配器。


<br />

##### XMM技术交流
欢迎加入XMM技术交流微信群，要加群，可以先添加如下微信让对方拉入群：

<img src=https://raw.githubusercontent.com/heiyeluren/docs/master/imgs/koala_wx.png width=20% />



<br />
<br />
<br />

## XMM如何使用？

<br />

说明：XMM测试程序快速预览下载使用
1. [XMM使用-入门](https://github.com/heiyeluren/XMM/blob/main/example/xmm-test00.go)
2. [XMM使用-结构体](https://github.com/heiyeluren/XMM/blob/main/example/xmm-test01.go)
3. [XMM使用-链表](https://github.com/heiyeluren/XMM/blob/main/example/xmm-test02.go)
4. [XMM使用-哈希表](https://github.com/heiyeluren/XMM/blob/main/example/xmm-test03.go)

<br />

<br />

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



```

<br />
<br />
<br />


#### 示例二：如何在结构体中使用XMM

说明：稍微复杂一点的应用，如何在结构体中使用XMM，进行申请和释放内存操作：


```go

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



```


<br />
<br />
<br />

### 示例三：使用XMM构建一个链表

说明：通过XMM构建一些复杂的数据结构应用，构建一个单链表的结构使用

<br />


```go
/*
   XMM 简单示例 - 02

   目标：使用XMM构建一个单链表程序
   说明：示例复杂场景中使用XMM内存库
*/
package main

import (
	//xmm "xmm/src"
	xmm "github.com/heiyeluren/xmm/src"
	"fmt"
	"unsafe"
)

//定义一个链表的节点结构
type XListNode struct {
	Val int
	Next *XListNode
}

//单链表主结构
type XList struct {
	Head *XListNode
	//Tail *XListNode
}


//初始化链表
func (l *XList) Init( mm xmm.XMemory ) {
	Node,err := mm.Alloc(unsafe.Sizeof(XListNode{}))
	if err != nil {
		panic("Alloc fail")
	}
	//head := &XListNode{Val:list[0]}
	head := (*XListNode)(Node)
	l.Head = head
	fmt.Println("Init Xlist done")
}

//链表增加节点
func (l *XList) Append(i int, mm xmm.XMemory) {
	h := l.Head
	for h.Next != nil {
		h = h.Next
	}
	p, err := mm.Alloc(unsafe.Sizeof(XListNode{}))
	if err != nil {
		panic("Alloc fail")
	}
	Node := (*XListNode)(p)
	Node.Val = i
	Node.Next = nil
	h.Next = Node

	fmt.Println("Append item:", Node.Val)
}

//遍历所有链表节点并打印
func (l *XList) Show() {
	h := l.Head
	//fmt.Println(h.Val)
	for h.Next != nil {
		h = h.Next
		fmt.Println("Show item:", h.Val)
	}
}

//释放整个链表结构
func (l *XList) Destroy(mm xmm.XMemory) {
	cnt := 0
	h := l.Head
	//fmt.Println(h.Val)

	//统计需要释放总数
	for h.Next != nil {
		h = h.Next
		cnt++
		//fmt.Println(h.Val)
	}
	//fmt.Println("item count:", cnt)

	//循环释放所有内存
	for i := 0; i <= cnt; i++ {
		h := l.Head
		pre := l.Head
		for h.Next != nil {
			pre = h
			h = h.Next
		}
		fmt.Println("Free item:", h.Val)
		mm.Free(uintptr(unsafe.Pointer(h)))
		pre.Next = nil
	}
}

//主函数
func main() {

	//初始化XMM对象
	f := &xmm.Factory{}
	//从操作系统申请一个内存块，如果内存使用达到60%，就进行异步自动扩容，每次异步扩容256MB内存（固定值）
	mm, err := f.CreateMemory(0.6)
	if err != nil {
		panic("CreateMemory fail ")
	}

	//要生成的数字列表
	list := []int{ 2, 4, 3}

	fmt.Println("\n===== XMM X(eXtensible) Memory Manager example 02 - LinkedList ======\n")


	//初始化链表
	l := &XList{}
	l.Init(mm)
	fmt.Println("")

	//把元素压入链表
	for i := 0; i < len(list); i++ {
		l.Append(list[i], mm)
	}
	fmt.Println("")

	//遍历所有链表数据
	l.Show()
	fmt.Println("")

	//释放所有链表内存
	l.Destroy(mm)

	//结束
	fmt.Println("\n===== Example test success ======\n")
}




```



<br />
<br />
<br />

### 示例四：使用XMM构建一个哈希表

说明：通过XMM构建一些复杂的数据结构应用，构建一个哈希表的数据结构使用（代码比较长，可以跳过阅读）

```go


/*
   XMM 简单示例 - 03

   目标：使用XMM构建一个哈希表程序
   说明：示例复杂场景中使用XMM内存库
*/
package main

import (
	//xmm "xmm/src"
	xmm "github.com/heiyeluren/xmm/src"
	"strconv"
	"strings"
	"fmt"
	"unsafe"
	"encoding/json"
)

//定义哈希表桶数量
const HashTableBucketMax = 1024

//定义哈希表存储实际KV节点存储数据
type XEntity struct {
	Key   string	//Key，必须是字符串
	Value string	//Value值，是字符串，把interface{}转json
}

//定义哈希表中单个桶结构（开拉链法）
type XBucket struct {
	Data *XEntity	//当前元素KV
	Next *XBucket	//如果冲突情况开拉链法下一个节点的Next指针
}

//哈希表入口主结构HashTable
type XHashTable struct {
	Table []*XBucket	//哈希表所有桶存储池
	Size uint64			//哈希表已存总元素数量
	mm xmm.XMemory		//XMM内存管理对象
}


//初始化哈希表
func (h *XHashTable) Init( mm xmm.XMemory ) {
	//设置需要申请多大的连续数组内存空间，如果是动态扩容情况建议设置为16，目前是按照常量桶大小
	cap := HashTableBucketMax
	initCap := uintptr(cap)
	//申请按照设定总桶数量大小的内存块
	p, err := mm.AllocSlice(unsafe.Sizeof(&XBucket{}), initCap, initCap)
	if err != nil {
		panic("Alloc fail")
	}
	//把申请的内存库给哈希表总存储池，初始化数量和XMM内存对象
	h.Table = *(*[]*XBucket)(p)
	h.Size = 0
	h.mm = mm

	fmt.Println("Init XHashTable done")
}

//释放整个哈希表
func (h *XHashTable) Destroy() (error) {
	return nil
}

//哈希表 Set数据
func (h *XHashTable) Set(Key string, Value interface{}) (error)  {

	//--------------
	// 构造Entity
	//--------------
	//Value进行序列化
	jdata, err := json.Marshal(Value)
	if err != nil {
		fmt.Println("Set() op Value [", Value, "] json encode fail")
		return err
	}
	sValue := string(jdata)

	//申请三个内存，操作字符串类型，XMM提供了From()接口，可以直接获得一个指针，字符串会存储在XMM中
	//size := unsafe.Sizeof(User{})
	p, err := h.mm.Alloc(unsafe.Sizeof(XEntity{}))
	if err != nil {
		panic("Alloc fail ")
	}
	pKey, err := h.mm.From(Key)
	if err != nil {
		panic("Alloc fail ")
	}
	pVal, err := h.mm.From(sValue)
	if err != nil {
		panic("Alloc fail ")
	}

	//拼装Entity
	pEntity := (*XEntity)(p)
	pEntity.Key = pKey
	pEntity.Value = pVal
	
	//---------------
	// 挂接到Bucket
	//---------------
	bucketIdx := getBucketSlot(Key)
	//bucketSize := unsafe.Sizeof(&XBucket{})
	bucket := h.Table[bucketIdx]

	//构造bucket
	pb, err := h.mm.Alloc(unsafe.Sizeof(XBucket{}))
	if err != nil {
		panic("Alloc fail ")
	}
	pBucket := (*XBucket)(pb)
	pBucket.Data = pEntity
	pBucket.Next = nil

	//如果槽没有被占用则压入数据后结束
	if bucket == nil {
		h.Table[bucketIdx] = pBucket
		h.Size = h.Size + 1
		return nil
	}

	//使用开拉链法把KV放入到冲突的槽中
	var k string
	for bucket != nil {
		k = bucket.Data.Key
		//如果发现有重名key，则直接替换Value
		if strings.Compare(strings.ToLower(Key), strings.ToLower(k)) == 0 {
			//释放原Value内存，挂接新Value
			//pv := bucket.Data.Value
			//mm.Free(pv)
			pValNew, err := h.mm.From(sValue)
			if err != nil {
				panic("Alloc fail ")
			}
			bucket.Data.Value = pValNew
			return nil
		}
		//如果是最后一个拉链的节点，则把当前KV挂上去
		if bucket.Next == nil {
			bucket.Next = pBucket
			h.Size = h.Size + 1
			return nil
		}
		//没找到则继续循环
		bucket = bucket.Next
	}
	return nil
}

//哈希表 Get数据
func (h *XHashTable) Get(Key string) (interface{}, error)  {
	var k string
	var val interface{}
	bucketIdx := getBucketSlot(Key)
	//bucketSize := unsafe.Sizeof(&XBucket{})
	bucket := h.Table[bucketIdx]
	for bucket != nil {
		k = bucket.Data.Key
		//如果查找到相同Key开始返回数据
		if strings.Compare(strings.ToLower(Key), strings.ToLower(k)) == 0 {
			//bTmp := []byte(k)
			err := json.Unmarshal([]byte(bucket.Data.Value), &val)
			if err != nil {
				//fmt.Println("Get() op Value [", Value, "] json decode fail")
				return nil, err
			}
			return val, nil
		}
		//没找到则继续向后查找
		if bucket.Next != nil {
			bucket = bucket.Next
			continue
		}
	}
	return nil, nil
}

//Delete数据
func (h *XHashTable) Remove(Key string)(error) {
	var k string
	bucketIdx := getBucketSlot(Key)
	bucket := h.Table[bucketIdx]

	//如果节点不存在直接返回
	if bucket == nil {
		return nil
	}

	//进行节点判断处理
	tmpBucketPre := bucket
	linkDepthSize := 0  	//存储当前开拉链第几层
	for bucket != nil {
		linkDepthSize = linkDepthSize + 1
		tmpBucketPre = bucket //把当前节点保存下来
		k = bucket.Data.Key
		//如果查找了相同的Key进行删除操作
		if strings.Compare(strings.ToLower(Key), strings.ToLower(k)) == 0 {
			//如果是深度第一层的拉链，把下一层拉链替换顶层拉链后返回
			if linkDepthSize == 1 {
				//如果是终点了直接当前桶置为nil
				if bucket.Next == nil {
					h.Table[bucketIdx] = nil
				} else { //如果还有其他下一级开拉链元素则替代本级元素
					h.Table[bucketIdx] = bucket.Next
				}
			} else { //如果查到的可以不是第一级元素，则进行前后替换
				tmpBucketPre.Next = bucket.Next
			}
			//释放内存
			//p := bucket.Data.Key
			//h.mm.Free(p)
			h.mm.FreeString(bucket.Data.Key)
			h.mm.FreeString(bucket.Data.Value)
			h.mm.Free(uintptr(unsafe.Pointer(bucket.Data)))
			//h.mm.Free(bucket)
			h.Size = h.Size - 1
			return nil
		}
		//如果还没找到，继续遍历把下一节点升级为当前节点
		if bucket.Next != nil {
			bucket = bucket.Next
			continue
		}
	}
	return nil
}

//获取目前总元素数量
func (h *XHashTable) getSize() (uint64) {
	return h.Size
}

//获取槽的计算位置
func getBucketSlot(key string) uint64 {
	hash := BKDRHash(key)
	return hash % HashTableBucketMax
}

//哈希函数（采用冲突率低性能高的 BKDR Hash算法，也可采用 MurmurHash）
func BKDRHash(key string) uint64 {
	var str []byte = []byte(key)  // string transfer format to []byte
	seed := uint64(131) // 31 131 1313 13131 131313 etc..
	hash := uint64(0)
	for i := 0; i < len(str); i++ {
		hash = (hash * seed) + uint64(str[i])
	}
	return hash ^ (hash>>16)&0x7FFFFFFF
}



//主函数
func main() {

	//初始化XMM对象
	f := &xmm.Factory{}
	//从操作系统申请一个内存块，如果内存使用达到60%，就进行异步自动扩容，每次异步扩容256MB内存（固定值）
	mm, err := f.CreateMemory(0.6)
	if err != nil {
		panic("CreateMemory fail ")
	}

	fmt.Println("\n===== XMM X(eXtensible) Memory Manager example 03 - HashTable ======\n")


	//初始化哈希表
	h := &XHashTable{}
	h.Init(mm)

	//简单数据类型压入哈希表
	fmt.Println("\n---- Simple data type hash Set/Get -----")

	//压入一批数据
	for i := 0; i < 5; i++ {
		fmt.Println("Hash Set: ", strconv.Itoa(i), strconv.Itoa(i*10))
		h.Set(strconv.Itoa(i), strconv.Itoa(i*10))
	}
	//读取数据
	for i := 0; i < 5; i++ {
		fmt.Print("Hash Get: ", i, " ")
		fmt.Println(h.Get(strconv.Itoa(i)))
	}
	fmt.Println("Hash Table Size: ", h.getSize())



	//存取复合型数据结构到哈希表
	fmt.Println("\n---- Mixed data type hash Set/Get -----")

	//构造测试数据
	testKV := make(map[string]interface{})
	testKV["map01"]   = map[string]string{"name":"heiyeluren", "email":"heiyeluren@example.com"}
	testKV["array01"] = [...]uint{9527, 2022, 8}
	testKV["slice01"] = make([]int, 3, 5)
	//fmt.Println(testKV)

	//压入数据到哈希表
	for k, v := range testKV {
		fmt.Print("Hash Set: ", k, " \n")
		h.Set(k, v)
	}
	//读取哈希表
	for k, _ := range testKV {
		fmt.Print("Hash Get: ", k, " ")
		fmt.Println(h.Get(k))
	}
	fmt.Println("Hash Table Size: ", h.getSize())

	//覆盖同样Key数据
	fmt.Println("\n---- Overwrite data hash Set/Get -----")
	for k, _ := range testKV {
		fmt.Print("Cover Hash Set: ", k, " \n")
		h.Set(k, "Overwrite data")
	}
	for k, _ := range testKV {
		fmt.Print("Cover Hash Get: ", k, " ")
		fmt.Println(h.Get(k))
	}
	fmt.Println("Hash Table Size: ", h.getSize())

	//删除Key
	fmt.Println("\n---- Delete data Remove op -----")

	k1 := "test01"
	v1 := "value01"
	fmt.Println("Hash Set: ", k1, " ", v1, " ", h.Set(k1, v1))
	fmt.Print("Hash Get: ", k1, " ")
	fmt.Println(h.Get(k1))

	fmt.Println("Hash Table Size: ", h.getSize())

	fmt.Print("Remove Key: ", k1)
	fmt.Println(h.Remove(k1))
	fmt.Print("Hash Get: ", k1, " ")
	fmt.Println(h.Get(k1))

	//读取老的key看看有没有受影响
	for k, _ := range testKV {
		fmt.Print("Hash Get: ", k, " ")
		fmt.Println(h.Get(k))
	}
	fmt.Println("Hash Table Size: ", h.getSize())


	//释放所有哈希表
	h.Destroy()

	//结束
	fmt.Println("\n===== Example test success ======\n")
}






```


<br />
<br />
<br />


## XMM问题反馈

XMM 目前是0.1版本，总体性能比较好，目前也在另外一个自研的XMap模块中使用，当然也少不了一些问题和bug，欢迎大家一起共创，或者直接提交PR等等。

欢迎加入XMM技术交流微信群，要加群，可以先添加如下微信让对方拉入群：


![image](https://raw.githubusercontent.com/heiyeluren/docs/master/imgs/koala_wx.png)



