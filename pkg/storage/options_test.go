// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package storage

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	"github.com/spf13/pflag"
)

func Test_Options(t *testing.T) {
	opt := NewStorageOptions()
	opt.AddFlags(pflag.CommandLine)
	if opt.StorageType != MongoStorageType {
		t.Errorf("failed to new storage options")
	}
}

func Test_RetrieveDataFromResult(t *testing.T) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write([]byte("YourDataHere")); err != nil {
		t.Errorf("failed to gzip data")
	}
	if err := gz.Flush(); err != nil {
		t.Errorf("failed to flush gz")
	}
	if err := gz.Close(); err != nil {
		t.Errorf("failed to close gz")
	}
	str := base64.StdEncoding.EncodeToString(b.Bytes())
	dataok, _ := base64.StdEncoding.DecodeString(str)
	testCases := []struct {
		name       string
		data       []byte
		compressed bool
		rst        error
	}{
		{
			name:       "ok_1",
			data:       []byte("abcd"),
			compressed: false,
			rst:        nil,
		},
		{
			name:       "ok_2",
			data:       dataok,
			compressed: true,
			rst:        nil,
		},
		{
			name:       "error_1",
			data:       []byte("abcd"),
			compressed: true,
			rst:        errors.New("failed to compress"),
		},
	}

	for _, testCase := range testCases {
		_, err := RetriveDataFromResult(testCase.data, testCase.compressed)
		if (err == nil && testCase.rst != nil) || (err != nil && testCase.rst == nil) {
			t.Errorf("case %s failed ", testCase.name)
		}
	}
}

func Test_NewMCMStorage(t *testing.T) {
	opt := NewStorageOptions()
	NewMCMStorage(opt, mcm.Kind("ResourceViewResult"))
}
