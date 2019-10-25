// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package utils

import (
	"fmt"
	"io/ioutil"
	"os"

	"k8s.io/klog"
	"k8s.io/client-go/util/certificate"
)

// WriteKeyCertToFile write key/cert to a certain cert path
func WriteKeyCertToFile(certDir string, key, cert []byte) (string, error) {
	if _, err := os.Stat(certDir); os.IsNotExist(err) {
		direrr := os.MkdirAll(certDir, 0755)
		if direrr != nil {
			return "", direrr
		}
	}

	store, err := certificate.NewFileStore("mongo", certDir, "", "", "")
	if err != nil {
		return "", fmt.Errorf("unable to build mongo cert store")
	}

	if _, err := store.Update(cert, key); err != nil {
		return "", err
	}

	return store.CurrentPath(), nil
}

//GeneratePemFile generate a pem file that include key and cert
func GeneratePemFile(dir, certFile, keyFile string) (string, error) {
	cert, err := readAll(certFile)
	if err != nil {
		klog.Error("Could not read cert file: ", certFile)
		return "", err
	}
	key, err := readAll(keyFile)
	if err != nil {
		klog.Error("Could not read key file: ", keyFile)
		return "", err
	}
	curFilePath, err := WriteKeyCertToFile(dir, key, cert)
	if err != nil {
		klog.Error("Could not generate pem file: ", err)
	}
	return curFilePath, nil
}

// ReadAll read a file
func readAll(filePth string) ([]byte, error) {
	f, err := os.Open(filePth)
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(f)
}
