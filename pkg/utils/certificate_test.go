// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package utils

import (
	"fmt"
	"os"
	"testing"

	certutil "k8s.io/client-go/util/cert"
)

func TestWriteKeyCertToFile(t *testing.T) {

	certDir := "/tmp/tmp-cert"
	defer os.RemoveAll(certDir)

	_, err := WriteKeyCertToFile(certDir, []byte("aaa"), []byte("aaa"))
	if err == nil {
		t.Errorf("Write key/cert file shoud fail")
	}

	cert, key, err := certutil.GenerateSelfSignedCertKey("test.com", nil, nil)
	if err != nil {
		t.Errorf("Failed to generate key: %v", err)
	}

	path, err := WriteKeyCertToFile(certDir, key, cert)
	if err != nil {
		t.Errorf("Faile to write key/cert: %v", err)
	}

	fmt.Printf("%v", path)
}
