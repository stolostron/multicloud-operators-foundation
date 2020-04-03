// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package utils

import "testing"

func Test_StringToMap(t *testing.T) {
	testCases := []struct {
		name string
		str  string
		rst  map[string]string
	}{
		{
			name: "case1",
			str:  "",
			rst:  nil,
		},
		{
			name: "case2",
			str:  "app=opt,zone=east-1",
			rst: map[string]string{
				"app":  "opt",
				"zone": "east-1",
			},
		},
	}

	for _, testCase := range testCases {
		rst := StringToMap(testCase.str)
		if len(rst) != len(testCase.rst) {
			t.Errorf("test case %s fail", testCase.name)
		}
	}
}
