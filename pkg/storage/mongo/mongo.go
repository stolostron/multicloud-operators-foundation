// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
// <('_')>

// A storage implementation for Kubernetes resources using mongodb.

package mongo

import (
	"context"
	"encoding/base64"
	"reflect"
	"time"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/client-go/tools/cache"

	"k8s.io/klog"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/clientopt"
)

type store struct {
	collection *mongo.Collection
}

const (
	mongoDocumentNotFoundErr = "mongo: no documents in result"
)

// NewMongoStorage create a mongon storage
func NewMongoStorage(options *Options, kind schema.GroupKind) (storage.Interface, error) {
	var err error
	clientOptions := []clientopt.Option{}
	uri := "mongodb://" + options.MongoHost + "/admin"
	if options.MongoSSLCa != "" && options.MongoSSLCert != "" && options.MongoSSLKey != "" {
		curFile, cerErr := utils.GeneratePemFile("/tmp", options.MongoSSLCert, options.MongoSSLKey)
		if cerErr != nil {
			return nil, cerErr
		}
		clientOption := clientopt.SSL(&clientopt.SSLOpt{
			ClientCertificateKeyFile: curFile,
			Enabled:                  true,
			CaFile:                   options.MongoSSLCa,
		})
		clientOptions = append(clientOptions, clientOption)
	}

	if options.MongoUser != "" && options.MongoPassword != "" {
		clientOption := clientopt.Auth(clientopt.Credential{
			Username:      options.MongoUser,
			Password:      options.MongoPassword,
			AuthMechanism: "SCRAM-SHA-1",
		})
		clientOptions = append(clientOptions, clientOption)
	}

	if options.MongoReplicaSet != "" {
		clientOption := clientopt.ReplicaSet(options.MongoReplicaSet)
		clientOptions = append(clientOptions, clientOption)
	}

	client, err := mongo.NewClientWithOptions(uri, clientOptions...)
	if err != nil {
		klog.Error("Could not connect to mongodb: ", err)
		return nil, err
	}

	err = client.Connect(context.TODO())
	if err != nil {
		klog.Error("Could not connect to mongodb: ", err)
		return nil, err
	}
	db := client.Database(options.MongoDatabaseName)
	klog.Info("Connected to mongo at ", options.MongoHost)
	newStore := &store{collection: db.Collection(kind.Kind)}

	go checkAndRefreshConnection(options, kind, uri, client, clientOptions, newStore)

	return newStore, nil
}

// This function periodically checks the mongo connection, and refreshes it if it has some problem.
// Due to an idiosyncrasy of the library, we must also reinitialize the Client object itself.
func checkAndRefreshConnection(
	options *Options,
	kind schema.GroupKind,
	uri string,
	client *mongo.Client,
	clientOptions []clientopt.Option, newStore *store) {
	for {
		// Check the connection by trying to count docs
		_, err := newStore.collection.CountDocuments(context.TODO(), map[string]string{})
		if err != nil {
			discErr := client.Disconnect(context.Background()) // Disconnect old connection
			if discErr != nil {
				klog.Error("Error disconnecting old MongoDB connection: ", discErr.Error())
			}
			client, err = mongo.NewClientWithOptions(uri, clientOptions...)
			if err != nil {
				klog.Error("Could not create MongoDB Client: ", err)
			}
			discErr = client.Connect(context.TODO()) // Create new MongoDB connection
			if discErr != nil {
				klog.Error("Error reconnecting to MongoDB: ", discErr.Error())
			}
			newStore.collection = client.Database(options.MongoDatabaseName).Collection(kind.Kind)
		}
		time.Sleep(time.Second * 30)
	}
}

// Versioner associated with this interface.
// NOTE: not implemented
func (m *store) Versioner() storage.Versioner {
	return nil
}

// Create adds a new object at a key unless it already exists. 'ttl' is time-to-live
// in seconds (0 means forever). If no error is returned and out is not nil, out will be
// set to the read value from database.
func (m *store) Create(ctx context.Context, key string, obj, out runtime.Object, ttl uint64) error {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return nil
	}
	// Need to set namespace here since we get internal object from apiserver
	namespace, _, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	accessor.SetNamespace(namespace)

	err = m.Get(context.TODO(), key, "", obj, false)
	if err == nil {
		return errors.NewAlreadyExists(v1alpha1.Resource("obj"), key)
	}
	if err.Error() == mongoDocumentNotFoundErr {
		labels := accessor.GetLabels()
		if labels != nil {
			encodedLables := encodeLableKey(labels)
			accessor.SetLabels(encodedLables)
		}
		bsonData, err := bson.Marshal(obj)
		if err != nil {
			return err
		}
		_, err = m.collection.InsertOne(ctx, bsonData)
		if err != nil {
			return err
		}
		return nil
	}
	return err
}

// Watch begins watching the specified key. Events are decoded into API objects,
// and any items selected by 'p' are sent down to returned watch.Interface.
// resourceVersion may be used to specify what version to begin watching,
// which should be the current resourceVersion, and no longer rv+1
// (e.g. reconnecting without missing any updates).
// If resource version is "0", this interface will get current object at given key
// and send it in an "ADDED" event, before watch starts.
// NOTE: not implemented
func (m *store) Watch(
	ctx context.Context, key string, resourceVersion string, p storage.SelectionPredicate) (watch.Interface, error) {
	return nil, nil
}

// WatchList begins watching the specified key's items. Items are decoded into API
// objects and any item selected by 'p' are sent down to returned watch.Interface.
// resourceVersion may be used to specify what version to begin watching,
// which should be the current resourceVersion, and no longer rv+1
// (e.g. reconnecting without missing any updates).
// If resource version is "0", this interface will list current objects directory defined by key
// and send them in "ADDED" events, before watch starts.
// NOTE: note implemented
func (m *store) WatchList(
	ctx context.Context, key string, resourceVersion string, p storage.SelectionPredicate) (watch.Interface, error) {
	return nil, nil
}

// Get unmarshals json found at key into objPtr. On a not found error, will either
// return a zero object of the requested type, or an error, depending on ignoreNotFound.
// Treats empty responses and nil response nodes exactly like a not found error.
// The returned contents may be delayed, but it is guaranteed that they will
// be have at least 'resourceVersion'.
func (m *store) Get(
	ctx context.Context, key string, resourceVersion string, objPtr runtime.Object, ignoreNotFound bool) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	filterDoc := bson.NewDocument(
		bson.EC.String("objectmeta.namespace", namespace),
		bson.EC.String("objectmeta.name", name),
	)
	result := m.collection.FindOne(context.TODO(), filterDoc)
	err = result.Decode(objPtr)
	if err != nil {
		if err.Error() == mongoDocumentNotFoundErr && ignoreNotFound {
			return nil
		}
		return err
	}
	return nil
}

// GetToList unmarshals json found at key and opaque it into *List api object
// (an object that satisfies the runtime.IsList definition).
// The returned contents may be delayed, but it is guaranteed that they will
// be have at least 'resourceVersion'.
// NOTE: not implemented
func (m *store) GetToList(
	ctx context.Context, key string, resourceVersion string, p storage.SelectionPredicate, listObj runtime.Object) error {
	return nil
}

// List unmarshalls jsons found at directory defined by key and opaque them
// into *List api object (an object that satisfies runtime.IsList definition).
// The returned contents may be delayed, but it is guaranteed that they will
// be have at least 'resourceVersion'.
func (m *store) List(
	ctx context.Context, key string, resourceVersion string, p storage.SelectionPredicate, listObj runtime.Object) error {
	var err error
	listPtr, err := meta.GetItemsPtr(listObj)
	if err != nil {
		return err
	}
	v, err := conversion.EnforcePtr(listPtr)
	if err != nil || v.Kind() != reflect.Slice {
		panic("need ptr to slice")
	}

	//transter selecot to map
	labelStr := p.Label.String()
	m1 := utils.StringToMap(labelStr)
	filterDoc := bson.Document{}
	for k, v := range m1 {
		el := bson.EC.String("objectmeta.labels."+encodeString(k), v)
		filterDoc.Append(el)
	}
	cursor, err := m.collection.Find(context.Background(), &filterDoc)
	defer func() {
		err = cursor.Close(context.Background())
	}()
	if err != nil {
		return err
	}

	for cursor.Next(context.Background()) {
		obj := reflect.New(v.Type().Elem()).Interface().(runtime.Object)
		err := cursor.Decode(obj)
		if err != nil {
			return err
		}
		accessor, err := meta.Accessor(obj)
		if err != nil {
			continue
		}
		labels := accessor.GetLabels()
		if labels != nil {
			decodedLables := decodeLableKey(labels)
			accessor.SetLabels(decodedLables)
		}

		v.Set(reflect.Append(v, reflect.ValueOf(obj).Elem()))
	}
	return err
}

// GuaranteedUpdate keeps calling 'tryUpdate()' to update key 'key' (of type 'ptrToType')
// retrying the update until success if there is index conflict.
func (m *store) GuaranteedUpdate(
	ctx context.Context, key string, ptrToType runtime.Object, ignoreNotFound bool,
	precondtions *storage.Preconditions, tryUpdate storage.UpdateFunc, suggestion ...runtime.Object) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	filterDoc := bson.NewDocument(
		bson.EC.String("objectmeta.namespace", namespace),
		bson.EC.String("objectmeta.name", name),
	)
	result := m.collection.FindOneAndDelete(context.TODO(), filterDoc)
	err = result.Decode(ptrToType)
	if err != nil {
		if err.Error() != mongoDocumentNotFoundErr || !ignoreNotFound {
			return err
		}
	}

	updatedObject, _, _ := tryUpdate(ptrToType, storage.ResponseMeta{})

	return m.Create(context.TODO(), key, updatedObject, nil, 0)
}

// Delete removes the specified key and returns the value that existed at that spot.
// If key didn't exist, it will return NotFound storage error.
func (m *store) Delete(ctx context.Context, key string, out runtime.Object, preconditions *storage.Preconditions) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	filterDoc := bson.NewDocument(
		bson.EC.String("objectmeta.namespace", namespace),
		bson.EC.String("objectmeta.name", name),
	)
	_, err = m.collection.DeleteOne(context.TODO(), filterDoc)
	if err != nil {
		return err
	}
	return nil
}

// Count returns number of different entries under the key (generally being path prefix).
// NOTE: not implemented
func (m *store) Count(key string) (int64, error) {
	return 0, nil
}

// encodeLableKey encode label to a base64 formart
func encodeLableKey(labels map[string]string) map[string]string {
	if len(labels) == 0 {
		return map[string]string{}
	}
	remap := map[string]string{}
	for k, v := range labels {
		newK := base64.StdEncoding.EncodeToString([]byte(k))
		remap[newK] = v
	}
	return remap
}

// decoeLableKey encode label to a base64 formart
func decodeLableKey(labels map[string]string) map[string]string {
	if len(labels) == 0 {
		return map[string]string{}
	}
	remap := map[string]string{}
	for k, v := range labels {
		decodedKey, err := base64.StdEncoding.DecodeString(k)
		if err != nil {
			continue
		}
		remap[string(decodedKey)] = v
	}
	return remap
}

func encodeString(str string) string {
	if len(str) == 0 {
		return ""
	}
	newstr := base64.StdEncoding.EncodeToString([]byte(str))
	return newstr
}
