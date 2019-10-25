// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
// <('_')>

package storage

import (
	"bytes"
	"compress/gzip"
	"io"

	"github.com/spf13/pflag"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/storage/mongo"
)

// MongoStorageType define the mongo storage
const MongoStorageType = "mongo"

// StorageOptions defines the flag for mongo
type StorageOptions struct {
	StorageType string
	Mongo       *mongo.MongoOptions
}

// AddFlags adds flags for ServerRunOptions fields to be specified via FlagSet.
func (m *StorageOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(
		&m.StorageType,
		"mcm-storage-type",
		m.StorageType,
		"storage to store mcm specific data",
	)

	m.Mongo.AddFlags(fs)
}

// NewStorageOptions create storage options
func NewStorageOptions() *StorageOptions {
	options := &StorageOptions{
		StorageType: MongoStorageType,
		Mongo:       mongo.NewMongoOptions(),
	}
	return options
}

// RetriveDataFromResult retrieve storage data
func RetriveDataFromResult(data []byte, compressed bool) ([]byte, error) {
	if !compressed {
		return data, nil
	}

	buf := bytes.NewBuffer(data)
	var reader io.Reader
	reader, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}
	var resultBuffer bytes.Buffer
	_, err = resultBuffer.ReadFrom(reader)
	if err != nil {
		return nil, err
	}

	return resultBuffer.Bytes(), nil
}
