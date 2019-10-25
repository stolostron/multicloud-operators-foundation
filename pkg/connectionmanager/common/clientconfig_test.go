// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package common

import (
	"reflect"
	"testing"
)

func TestClientConfig(t *testing.T) {
	key, cert, _ := NewCertKey("test", "test")
	config := NewClientConfig("localhost", cert, key)
	ohost, okey, ocert, err := GetHostCerKeyFromClientConfig(config)
	if err != nil {
		t.Errorf("failed to parse client config")
	}

	if ohost != "localhost" {
		t.Errorf("host is not correct")
	}

	if !reflect.DeepEqual(key, okey) || !reflect.DeepEqual(cert, ocert) {
		t.Errorf("key and certs are not equal")
	}
}
