// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package version

import "testing"

func TestGet(t *testing.T) {
	info := Get()
	if info.BuildDate == "" {
		t.Errorf("fail to get version")
	}
}
