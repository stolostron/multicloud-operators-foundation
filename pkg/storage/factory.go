// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
// <('_')>

package storage

import (
	"fmt"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/storage/mongo"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/storage"
)

func NewMCMStorage(options *StorageOptions, kind schema.GroupKind) (storage.Interface, error) {
	if options.StorageType == MongoStorageType {
		return mongo.NewMongoStorage(options.Mongo, kind)
	}

	return nil, fmt.Errorf("storage type is not supported")
}
