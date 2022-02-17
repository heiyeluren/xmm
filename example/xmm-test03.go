
/*
   XMM 简单示例 - 03

   目标：使用XMM构建一个哈希表程序
   说明：示例复杂场景中使用XMM内存库
   
   XMM Simple Example - 03

   Objective: To build a hash table program using XMM
   Description: Example complex scenario using the XMM memory library   
*/
package main

import (
	//xmm "xmm/src"
	xmm "github.com/heiyeluren/xmm"
	"strconv"
	"strings"
	"fmt"
	"unsafe"
	"encoding/json"
)

//定义哈希表桶数量
// Define the number of hash table buckets
const HashTableBucketMax = 1024

//定义哈希表存储实际KV节点存储数据
// Define hash tables to store actual KV node storage data
type XEntity struct {
	Key   string	//Key必须是字符串  Key, must be a string
	Value string	//Value是字符串，从interface转成json格式  Value value, a string, convert interface{} to json
}

//定义哈希表中单个桶结构（开拉链法）
// Define a single bucket structure in a hash table (open zip method)
type XBucket struct {
	Data *XEntity	//当前元素KV Current element KV
	Next *XBucket	//如果冲突情况开拉链法下一个节点的Next指针 Next pointer to the next node of the open zip method if a conflict situation arises
}

//哈希表入口主结构HashTable
// HashTable entry main structure HashTable
type XHashTable struct {
	Table []*XBucket	//哈希表所有桶存储池 Hash table all bucket storage pool
	Size uint64		//哈希表已存总元素数量 Total number of elements stored in the hash table
	mm xmm.XMemory		//XMM内存管理对象 XMM Memory Management Objects
}


//初始化哈希表
// Initialize the hash table
func (h *XHashTable) Init( mm xmm.XMemory ) {
	//设置需要申请多大的连续数组内存空间，如果是动态扩容情况建议设置为16，目前是按照常量桶大小
	// Set how much contiguous array memory space to request, 16 is recommended for dynamic expansion cases, currently it is based on the constant bucket size
	cap := HashTableBucketMax
	initCap := uintptr(cap)
	
	//申请按照设定总桶数量大小的内存块
	// Request a block of memory of the set total bucket size
	p, err := mm.AllocSlice(unsafe.Sizeof(&XBucket{}), initCap, initCap)
	if err != nil {
		panic("Alloc fail")
	}
	
	//把申请的内存库给哈希表总存储池，初始化数量和XMM内存对象
	// give the requested memory bank to the total hash table storage pool, initialize the number and XMM memory objects
	h.Table = *(*[]*XBucket)(p)
	h.Size = 0
	h.mm = mm

	fmt.Println("Init XHashTable done")
}

//释放整个哈希表
//Free the entire hash table
func (h *XHashTable) Destroy() (error) {
	return nil
}

//哈希表 Set数据
///hash tables Set data
func (h *XHashTable) Set(Key string, Value interface{}) (error)  {

	//--------------
	// 构造Entity 
	// Constructing Entity
	//--------------
	
	//Value进行序列化
	//Value for serialisation
	jdata, err := json.Marshal(Value)
	if err != nil {
		fmt.Println("Set() op Value [", Value, "] json encode fail")
		return err
	}
	sValue := string(jdata)

	//申请三个内存，操作字符串类型，XMM提供了From()接口，可以直接获得一个指针，字符串会存储在XMM中
	// request three memory, manipulate string types, XMM provides a From() interface to get a pointer directly, the string will be stored in XMM
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
	// Assembling Entity
	pEntity := (*XEntity)(p)
	pEntity.Key = pKey
	pEntity.Value = pVal
	
	//---------------
	// 挂接到Bucket
	// Hook up to Bucket
	//---------------
	bucketIdx := getBucketSlot(Key)
	//bucketSize := unsafe.Sizeof(&XBucket{})
	bucket := h.Table[bucketIdx]

	//构造bucket
	// Constructing a bucket
	pb, err := h.mm.Alloc(unsafe.Sizeof(XBucket{}))
	if err != nil {
		panic("Alloc fail ")
	}
	pBucket := (*XBucket)(pb)
	pBucket.Data = pEntity
	pBucket.Next = nil

	//如果槽没有被占用则压入数据后结束
	// end after pressing in data if the slot is not occupied
	if bucket == nil {
		h.Table[bucketIdx] = pBucket
		h.Size = h.Size + 1
		return nil
	}

	//使用开拉链法把KV放入到冲突的槽中
	// Use the unzipping method to place the KV in the conflicting slot
	var k string
	for bucket != nil {
		k = bucket.Data.Key
		//如果发现有重名key，则直接替换Value
		// If a duplicate key is found, the Value is replaced directly
		if strings.Compare(strings.ToLower(Key), strings.ToLower(k)) == 0 {
			//挂接新Value
			// mount the new Value
			pValNew, err := h.mm.From(sValue)
			if err != nil {
				panic("Alloc fail ")
			}
			bucket.Data.Value = pValNew
			return nil
		}
		//如果是最后一个拉链的节点，则把当前KV挂上去
		// If it is the last node to zip, hook up the current KV
		if bucket.Next == nil {
			bucket.Next = pBucket
			h.Size = h.Size + 1
			return nil
		}
		//没找到则继续循环
		// continue the cycle if not found
		bucket = bucket.Next
	}
	return nil
}

//哈希表 Get数据
// Hash Table Get Data
func (h *XHashTable) Get(Key string) (interface{}, error)  {
	var k string
	var val interface{}
	bucketIdx := getBucketSlot(Key)
	//bucketSize := unsafe.Sizeof(&XBucket{})
	bucket := h.Table[bucketIdx]
	for bucket != nil {
		k = bucket.Data.Key
		//如果查找到相同Key开始返回数据
		// If the same Key is found start returning data
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
		// continue to search backwards if not found
		if bucket.Next != nil {
			bucket = bucket.Next
			continue
		}
	}
	return nil, nil
}

//哈希表 Delete数据
// Hash Table Delete Data
func (h *XHashTable) Remove(Key string)(error) {
	var k string
	bucketIdx := getBucketSlot(Key)
	bucket := h.Table[bucketIdx]

	//如果节点不存在直接返回
	// return directly if node does not exist
	if bucket == nil {
		return nil
	}

	//进行节点判断处理
	// Perform node judgement processing
	tmpBucketPre := bucket
	linkDepthSize := 0  	//存储当前开拉链第几层 Store the current layer of open zips
	for bucket != nil {
		linkDepthSize = linkDepthSize + 1
		tmpBucketPre = bucket //把当前节点保存下来 Save the current node
		k = bucket.Data.Key
		
		//如果查找了相同的Key进行删除操作
		// If the same Key is found for the delete operation
		if strings.Compare(strings.ToLower(Key), strings.ToLower(k)) == 0 {
			
			//如果是深度第一层的拉链，把下一层拉链替换顶层拉链后返回
			// In the case of a deep first layer of zips, replace the next layer of zips with the top layer and return
			if linkDepthSize == 1 {
				//如果是终点了直接当前桶置为nil
				//If it's the end of the line set the current bucket to nil
				if bucket.Next == nil {
					h.Table[bucketIdx] = nil
				} else { //如果还有其他下一级开拉链元素则替代本级元素 If there are other open zip elements at the next level, they replace the element at this level
					h.Table[bucketIdx] = bucket.Next
				}
			} else { //如果查到的可以不是第一级元素，则进行前后替换 If the element found can be other than the first level, then a back and forth substitution is made
				tmpBucketPre.Next = bucket.Next
			}
			//释放内存
			//Free memory
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
		// If not found, continue traversing to upgrade the next node to the current one
		if bucket.Next != nil {
			bucket = bucket.Next
			continue
		}
	}
	return nil
}

//获取哈希表元素总数量
// Get the total number of hash table elements
func (h *XHashTable) getSize() (uint64) {
	return h.Size
}

//获取指定Key在哈希表中槽的位置
// Get the position of the specified Key in the hash table slot
func getBucketSlot(key string) uint64 {
	hash := BKDRHash(key)
	return hash % HashTableBucketMax
}

//哈希函数（采用冲突率低性能高的 BKDR Hash算法，也可采用 MurmurHash）
// Hash function (using BKDR Hash algorithm with low conflict rate and high performance, MurmurHash can also be used)
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
// Main functions
func main() {

	//初始化XMM对象
	// Initialise the XMM object
	f := &xmm.Factory{}
	
	//从操作系统申请一个内存块，如果内存使用达到60%，就进行异步自动扩容，每次异步扩容256MB内存（固定值）
	// Request a block of memory from the OS and perform asynchronous auto-expansion if memory usage reaches 60%, 256MB of memory per asynchronous expansion (fixed value)
	mm, err := f.CreateMemory(0.6)
	if err != nil {
		panic("CreateMemory fail ")
	}

	fmt.Println("\n===== XMM X(eXtensible) Memory Manager example 03 - HashTable ======\n")


	//初始化哈希表
	// Initialize the hash table
	h := &XHashTable{}
	h.Init(mm)

	//简单数据类型压入哈希表
	// Simple data types pressed into a hash table
	fmt.Println("\n---- Simple data type hash Set/Get -----")

	//压入一批数据
	// Press in a batch of data
	for i := 0; i < 5; i++ {
		fmt.Println("Hash Set: ", strconv.Itoa(i), strconv.Itoa(i*10))
		h.Set(strconv.Itoa(i), strconv.Itoa(i*10))
	}
	//读取数据
	// Read data
	for i := 0; i < 5; i++ {
		fmt.Print("Hash Get: ", i, " ")
		fmt.Println(h.Get(strconv.Itoa(i)))
	}
	fmt.Println("Hash Table Size: ", h.getSize())



	//存取复合型数据结构到哈希表
	// Accessing compound data structures to hash tables
	fmt.Println("\n---- Mixed data type hash Set/Get -----")

	//构造测试数据
	// Constructing test data
	testKV := make(map[string]interface{})
	testKV["map01"]   = map[string]string{"name":"heiyeluren", "email":"heiyeluren@example.com"}
	testKV["array01"] = [...]uint{9527, 2022, 8}
	testKV["slice01"] = make([]int, 3, 5)
	//fmt.Println(testKV)

	//压入数据到哈希表
	// Pressing data into a hash table
	for k, v := range testKV {
		fmt.Print("Hash Set: ", k, " \n")
		h.Set(k, v)
	}
	//读取哈希表
	// Read the hash table
	for k, _ := range testKV {
		fmt.Print("Hash Get: ", k, " ")
		fmt.Println(h.Get(k))
	}
	fmt.Println("Hash Table Size: ", h.getSize())

	//覆盖同样Key数据
	// Overwrite the same Key data
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
	// Delete Key
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
	// Read the old key to see if it is affected
	for k, _ := range testKV {
		fmt.Print("Hash Get: ", k, " ")
		fmt.Println(h.Get(k))
	}
	fmt.Println("Hash Table Size: ", h.getSize())


	//释放所有哈希表
	// Free all hash tables
	h.Destroy()

	//结束
	//End
	fmt.Println("\n===== Example test success ======\n")
}


