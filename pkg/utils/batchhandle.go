// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package utils

import (
	"sync"

	"k8s.io/client-go/util/integer"
)

var maxBatchSize = 500

// BatchHandle batch the resource handles.
func BatchHandle(handleCnt int, handle func(int)) {
	batchSize := integer.IntMin(handleCnt, maxBatchSize)
	createWait := sync.WaitGroup{}
	for startIndex := 0; startIndex < handleCnt; {
		createWait.Add(batchSize)
		for i := startIndex; i < startIndex+batchSize; i++ {
			go func(ix int) {
				defer createWait.Done()
				handle(ix)
			}(i)
		}

		createWait.Wait()

		startIndex += batchSize
		batchSize = integer.IntMin(batchSize, handleCnt-startIndex)
	}
}
