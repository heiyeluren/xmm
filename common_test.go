// Copyright (c) 2022 XMM project Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// XMM Project Site: https://github.com/heiyeluren
// XMM URL: https://github.com/heiyeluren/XMM
//

package xmm

import (
	"fmt"
	"sync"
	"testing"
)

func TestSpanList(t *testing.T) {
	waitCnt, loopCnt := 50, 50000
	list := mSpanList{}
	var wait sync.WaitGroup
	wait.Add(waitCnt)
	for i := 0; i < waitCnt; i++ {
		go func() {
			defer wait.Done()
			for j := 0; j < loopCnt; j++ {
				list.insert(&xSpan{classIndex: uint(j)})
			}
		}()
	}
	wait.Wait()
	cnt := 0
	cntMap := make(map[uint]int)
	list.foreach(func(span *xSpan) {
		cnt++
		cntMap[span.classIndex] = cntMap[span.classIndex] + 1
	})
	for u, i := range cntMap {
		if i != waitCnt {
			t.Fatal(u, "的数目不对 cnt:", i)
		}
		//fmt.Println(u, i)
	}
	if cnt != waitCnt*loopCnt {
		t.Fatal("总数不对")
	}
	fmt.Println(cnt)

}
