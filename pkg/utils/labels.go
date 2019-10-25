// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package utils

import "strings"

//string to map
func StringToMap(str string) map[string]string {
	if len(str) == 0 {
		return nil
	}
	returnMap := map[string]string{}
	split := strings.Split(str, ",")
	for _, sp := range split {
		tm := strings.Split(sp, "=")
		if len(tm) >= 2 {
			returnMap[tm[0]] = tm[1]
		}
	}
	return returnMap
}
