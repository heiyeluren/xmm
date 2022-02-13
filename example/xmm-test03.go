
/*
   XMM 简单示例 - 03

   目标：使用XMM构建一个哈希表程序
   说明：示例复杂场景中使用XMM内存库
*/
package main

import (
	xmm "xmm/src"
	"strconv"
	"strings"
	"fmt"
	"unsafe"
	"encoding/json"
	//"reflect"
	//_ "github.com/spf13/cast"
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
			//h.mm.Free(bucket.Data.Key)
			//h.mm.Free(bucket.Data.Value)
			//h.mm.Free(bucket.Data)
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


