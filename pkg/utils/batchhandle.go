// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package utils

import (
	"fmt"
	"k8s.io/client-go/util/integer"
	"sync"
)

var maxBatchSize = 500

// BatchHandle batch the resource handles.
func BatchHandle(handleCnt int, handle func(int) error) error {
	batchSize := integer.IntMin(handleCnt, maxBatchSize)
	createWait := sync.WaitGroup{}
	errCh := make(chan error, handleCnt)
	for startIndex := 0; startIndex < handleCnt; {
		createWait.Add(batchSize)
		for i := startIndex; i < startIndex+batchSize; i++ {
			go func(ix int) {
				defer createWait.Done()
				if err := handle(ix); err != nil {
					errCh <- err
				}
			}(i)
		}

		createWait.Wait()

		startIndex = startIndex + batchSize
		batchSize = integer.IntMin(batchSize, handleCnt-startIndex)
	}

	close(errCh)
	errCount := len(errCh)
	errors := ""
	if errCount > 0 {
		for err := range errCh {
			errors += err.Error() + "\n"
		}
		return fmt.Errorf("batch handle occurs %v errors: \n%v", errCount, errors)
	}

	return nil
}
