package benchmark

import (
	"fmt"
	"github.com/heiyeluren/xmm"
	"testing"
	"unsafe"
)

func BenchmarkAlloc_Go(b *testing.B) {
	data := make([]*User, 10000)
	name := "zhansan"
	for i := 0; i < b.N; i++ {
		data = append(data, &User{
			Name: name,
			Age:  18,
		})
	}
}

func BenchmarkAlloc_Xmm(b *testing.B) {
	data := make([]*User, 10000)
	//Initialising XMM objects
	//初始化XMM对象
	f := &xmm.Factory{}

	// Request a memory block from the operating system
	//If memory usage reaches 60%, asynchronous automatic expansion is performed, each time 256MB of memory is expanded asynchronously (fixed value), the value of 0.6 can be configured independently
	//从操作系统申请一个内存块
	//如果内存使用达到60%，就进行异步自动扩容，每次异步扩容256MB内存（固定值），0.6这个数值可以自主配置
	mm, err := f.CreateMemory(0.8)
	if err != nil {
		panic("CreateMemory fail ")
	}
	name, err := mm.From("heiyeluren")
	if err != nil {
		panic("Alloc fail ")
	}
	size := unsafe.Sizeof(User{})
	for i := 0; i < b.N; i++ {
		p, err := mm.Alloc(size)
		if err != nil {
			panic("Alloc fail ")
		}
		u := (*User)(p)
		u.Age = 18

		// To manipulate string types, XMM provides the From() interface to get a pointer directly and the string will be stored in XMM
		//操作字符串类型，XMM提供了From()接口，可以直接获得一个指针，字符串会存储在XMM中
		u.Name = name
		data = append(data, u)
	}
}

func BenchmarkGoAlloc_GC(b *testing.B) {
	var data *User
	name := "zhansan"
	for i := 0; i < b.N; i++ {
		data = &User{
			Name: name,
			Age:  18,
		}
	}
	fmt.Sprintf("%+v", data)
}

func BenchmarkXmmAlloc_GC(b *testing.B) {
	//Initialising XMM objects
	//初始化XMM对象
	f := &xmm.Factory{}
	var data *User

	// Request a memory block from the operating system
	//If memory usage reaches 60%, asynchronous automatic expansion is performed, each time 256MB of memory is expanded asynchronously (fixed value), the value of 0.6 can be configured independently
	//从操作系统申请一个内存块
	//如果内存使用达到60%，就进行异步自动扩容，每次异步扩容256MB内存（固定值），0.6这个数值可以自主配置
	mm, err := f.CreateMemory(0.8)
	if err != nil {
		panic("CreateMemory fail ")
	}
	// To manipulate string types, XMM provides the From() interface to get a pointer directly and the string will be stored in XMM
	//操作字符串类型，XMM提供了From()接口，可以直接获得一个指针，字符串会存储在XMM中
	name, err := mm.From("heiyeluren")
	if err != nil {
		panic("Alloc fail ")
	}
	size := unsafe.Sizeof(User{})
	for i := 0; i < b.N; i++ {
		p, err := mm.Alloc(size)
		if err != nil {
			panic("Alloc fail ")
		}
		u := (*User)(p)
		u.Age = 18
		u.Name = name
		data = u
	}
	fmt.Sprintf("%+v", data)
}
