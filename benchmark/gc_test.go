package benchmark

import (
	"fmt"
	"github.com/heiyeluren/xmm"
	"net/http"

	//"net/http"
	_ "net/http/pprof"
	"testing"
)

var M = 1024 * 1024

type User struct {
	Name string
	Age  int
}

func BenchmarkGc_Go(b *testing.B) {
	//初始化分配800m
	data := make([]byte, 800*M)
	data2 := make([]*User, 10000)
	for i := 0; i < b.N; i++ {
		//模拟小对象创建
		data2 = append(data2, &User{
			Name: "zhansan",
			Age:  18,
		})
		//模拟大对象创建
		if i > 10000 && i%10000 == 0 {
			data = append(data, make([]byte, M/12)...)
		}
	}
	fmt.Sprintf("len:%d", len(data))
	/*var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("\n---------------HeapAlloc:%d GCCPUFraction:%f PauseTotalNs:%d NumGC:%d \n", m.HeapAlloc, m.GCCPUFraction, m.PauseTotalNs, m.NumGC)*/
}

//xmm无手动释放内存，针对该xmm分配内存为进程的生命周期相同。
func BenchmarkGc_Xmm(b *testing.B) {
	f := &xmm.Factory{}

	// Request a memory block from the operating system
	//If memory usage reaches 60%, asynchronous automatic expansion is performed, each time 256MB of memory is expanded asynchronously (fixed value), the value of 0.6 can be configured independently
	//从操作系统申请一个内存块
	//如果内存使用达到60%，就进行异步自动扩容，每次异步扩容256MB内存（固定值），0.6这个数值可以自主配置
	mm, err := f.CreateMemory(0.8)
	if err != nil {
		panic("CreateMemory fail ")
	}
	//初始化分配800m
	ca := uintptr(800 * M)
	p, err := mm.AllocSlice(1, ca, ca)
	if err != nil {
		panic("Alloc fail ")
	}
	data := *(*[]byte)(p)
	data2 := make([]*User, 10000)
	for i := 0; i < b.N; i++ {
		//模拟小对象创建
		data2 = append(data2, &User{Name: "zhansan", Age: 18})
		//模拟大对象创建
		if i > 10000 && i%10000 == 0 {
			cac := uintptr(M / 12)
			p, err := mm.AllocSlice(1, cac, cac)
			if err != nil {
				panic("Alloc fail ")
			}
			data1 := *(*[]byte)(p)
			data = append(data, data1...)
		}
	}
	fmt.Sprintf("len:%d", len(data))
	/*var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("---------------HeapAlloc:%d GCCPUFraction:%f PauseTotalNs:%d NumGC:%d \n", m.HeapAlloc, m.GCCPUFraction, m.PauseTotalNs, m.NumGC)*/
}

func BenchmarkGcLargeMemory_Go(b *testing.B) {
	var data []byte
	//初始化分配800m
	data2 := make([]*User, 10000)
	for i := 0; i < b.N; i++ {
		//模拟小对象创建
		data2 = append(data2, &User{
			Name: "zhansan",
			Age:  18,
		})
		//模拟大对象创建
		if i%3 == 0 {
			data = append(data, make([]byte, M/12)...)
		} else {
			dat := make([]byte, M/12)
			fmt.Sprintf("%d", len(dat))
		}
	}
	fmt.Sprintf("len:%d", len(data))
	/*var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("\n---------------HeapAlloc:%d GCCPUFraction:%f PauseTotalNs:%d NumGC:%d \n", m.HeapAlloc, m.GCCPUFraction, m.PauseTotalNs, m.NumGC)*/
}

func BenchmarkGcLargeMemory_Xmm(b *testing.B) {
	go func() {
		http.ListenAndServe("0.0.0.0:17012", nil)
	}()
	f := &xmm.Factory{}

	// Request a memory block from the operating system
	//If memory usage reaches 60%, asynchronous automatic expansion is performed, each time 256MB of memory is expanded asynchronously (fixed value), the value of 0.6 can be configured independently
	//从操作系统申请一个内存块
	//如果内存使用达到60%，就进行异步自动扩容，每次异步扩容256MB内存（固定值），0.6这个数值可以自主配置
	mm, err := f.CreateMemory(0.8)
	if err != nil {
		panic("CreateMemory fail ")
	}
	var data []byte
	data2 := make([]*User, 10000)
	for i := 0; i < b.N; i++ {
		//模拟小对象创建
		data2 = append(data2, &User{
			Name: "zhansan",
			Age:  18,
		})
		cac := uintptr(M / 12)
		p, err := mm.AllocSlice(1, cac, cac)
		if err != nil {
			panic("Alloc fail ")
		}
		dat := *(*[]byte)(p)
		if i%3 == 0 {
			data = append(data, dat...)
		} else {
			fmt.Sprintf("%d", len(dat))
			mm.Free(uintptr(p))
		}
	}
	fmt.Sprintf("len:%d", len(data))
	/*var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("---------------HeapAlloc:%d GCCPUFraction:%f PauseTotalNs:%d NumGC:%d \n", m.HeapAlloc, m.GCCPUFraction, m.PauseTotalNs, m.NumGC)*/
}
