package mongo

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/deleteopt"
	"github.com/mongodb/mongo-go-driver/mongo/findopt"
	"github.com/mongodb/mongo-go-driver/mongo/insertopt"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	. "github.com/smartystreets/goconvey/convey"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/storage"
)

func Test_NewMongoStorage(t *testing.T) {
	Convey("test NewMongoStorage", t, func() {
		options := []Options{
			{
				MongoUser:         "",
				MongoPassword:     "",
				MongoHost:         "",
				MongoDatabaseName: "",
				MongoSSLCa:        "",
				MongoSSLCert:      "",
				MongoSSLKey:       "",
				MongoReplicaSet:   "",
				MongoCollection:   "",
			},
			{
				MongoUser:         "",
				MongoPassword:     "",
				MongoHost:         "",
				MongoDatabaseName: "",
				MongoSSLCa:        "a.ca",
				MongoSSLCert:      "a.cert",
				MongoSSLKey:       "a.key",
				MongoReplicaSet:   "",
				MongoCollection:   "",
			},
			{
				MongoUser:         "admin",
				MongoPassword:     "admin",
				MongoHost:         "",
				MongoDatabaseName: "",
				MongoSSLCa:        "a.ca",
				MongoSSLCert:      "a.cert",
				MongoSSLKey:       "a.key",
				MongoReplicaSet:   "",
				MongoCollection:   "",
			},
			{
				MongoUser:         "admin",
				MongoPassword:     "admin",
				MongoHost:         "",
				MongoDatabaseName: "",
				MongoSSLCa:        "a.ca",
				MongoSSLCert:      "a.cert",
				MongoSSLKey:       "a.key",
				MongoReplicaSet:   "mcm",
				MongoCollection:   "",
			},
		}

		for _, option := range options {
			_, err := NewMongoStorage(&option, schema.GroupKind{})
			So(err, ShouldBeError)
		}
	})
}

func Test_store_Create(t *testing.T) {
	Convey("test store Create", t, func() {
		Convey("test store create case1", func() {
			col := &mongo.Collection{}
			m := &store{
				collection: col,
			}
			err := m.Create(context.TODO(), "", nil, nil, 1)
			So(err, ShouldBeNil)
		})
		Convey("test store create case2", func() {
			col := &mongo.Collection{}
			m := &store{
				collection: col,
			}
			obj := &mcm.ResourceView{}
			err := m.Create(context.TODO(), "/a/b/c", obj, nil, 1)
			So(err, ShouldBeError)
		})
		Convey("test store create case3", func() {
			defer monkey.UnpatchAll()
			col := &mongo.Collection{}
			m := &store{
				collection: col,
			}
			monkey.PatchInstanceMethod(reflect.TypeOf(m), "Get", func(_ *store, ctx context.Context, key string,
				resourceVersion string, objPtr runtime.Object, ignoreNotFound bool) error {
				return nil
			})
			obj := &mcm.ResourceView{}
			err := m.Create(context.TODO(), "default/key", obj, nil, 1)
			So(err, ShouldBeError)
		})
		Convey("test store create case4", func() {
			defer monkey.UnpatchAll()
			col := &mongo.Collection{}
			monkey.PatchInstanceMethod(reflect.TypeOf(col), "InsertOne", func(_ *mongo.Collection, ctx context.Context, document interface{},
				opts ...insertopt.One) (*mongo.InsertOneResult, error) {
				return nil, errors.New("fail to InsertOne")
			})
			m := &store{
				collection: col,
			}
			monkey.PatchInstanceMethod(reflect.TypeOf(m), "Get", func(_ *store, ctx context.Context, key string,
				resourceVersion string, objPtr runtime.Object, ignoreNotFound bool) error {
				return errors.New(mongoDocumentNotFoundErr)
			})
			obj := &mcm.ResourceView{}
			obj.SetLabels(map[string]string{"app": "app"})
			err := m.Create(context.TODO(), "default/key", obj, nil, 1)
			So(err, ShouldBeError)
		})
	})
}

func Test_store_Get(t *testing.T) {
	Convey("test store Get", t, func() {
		Convey("test store Get case1", func() {
			col := &mongo.Collection{}
			m := &store{
				collection: col,
			}
			err := m.Get(context.TODO(), "a/b/c", "", nil, false)
			So(err, ShouldBeError)
		})
		Convey("test store Get case2", func() {
			defer monkey.UnpatchAll()
			col := &mongo.Collection{}
			monkey.PatchInstanceMethod(reflect.TypeOf(col), "FindOne", func(_ *mongo.Collection, ctx context.Context,
				filter interface{}, opts ...findopt.One) *mongo.DocumentResult {
				return &mongo.DocumentResult{}
			})
			m := &store{
				collection: col,
			}
			err := m.Get(context.TODO(), "default/key", "", nil, false)
			So(err, ShouldBeError)
		})
	})
}

func Test_store_List(t *testing.T) {
	Convey("test store List", t, func() {
		Convey("Test store List case1", func() {
			col := &mongo.Collection{}
			m := &store{
				collection: col,
			}
			err := m.List(context.TODO(), "", "", storage.SelectionPredicate{}, nil)
			So(err, ShouldBeError)
		})
		Convey("Test store List case2", func() {
			defer monkey.UnpatchAll()
			col := &mongo.Collection{}
			monkey.PatchInstanceMethod(reflect.TypeOf(col), "Find", func(_ *mongo.Collection, ctx context.Context, filter interface{},
				opts ...findopt.Find) (mongo.Cursor, error) {
				return nil, errors.New("failed to Find")
			})
			m := &store{
				collection: col,
			}
			p := storage.SelectionPredicate{
				Label: labels.Everything(),
			}
			listObj := &mcm.ResourceViewList{}
			err := m.List(context.TODO(), "", "", p, listObj)
			So(err, ShouldBeError)
		})
	})
}

func Test_store_GuaranteedUpdate(t *testing.T) {
	Convey("Test store GuaranteedUpdate", t, func() {
		Convey("Test store GuaranteedUpdate case1", func() {
			col := &mongo.Collection{}
			m := &store{
				collection: col,
			}
			err := m.GuaranteedUpdate(context.TODO(), "a/b/c", nil, false, nil, nil)
			So(err, ShouldBeError)
		})
		Convey("Test store GuaranteedUpdate case2", func() {
			defer monkey.UnpatchAll()
			col := &mongo.Collection{}
			monkey.PatchInstanceMethod(reflect.TypeOf(col), "FindOneAndDelete", func(_ *mongo.Collection,
				ctx context.Context, filter interface{},
				opts ...findopt.DeleteOne) *mongo.DocumentResult {
				return &mongo.DocumentResult{}
			})
			m := &store{
				collection: col,
			}
			err := m.GuaranteedUpdate(context.TODO(), "default/key", nil, false, nil, nil)
			So(err, ShouldBeError)
		})
	})
}

func Test_store_Delete(t *testing.T) {
	Convey("Test store Delete", t, func() {
		Convey("Test store Delete case1", func() {
			col := &mongo.Collection{}
			m := &store{
				collection: col,
			}
			err := m.Delete(context.TODO(), "a/b/c", nil, nil, nil)
			So(err, ShouldBeError)
		})
		Convey("Test store Delete case2", func() {
			defer monkey.UnpatchAll()
			col := &mongo.Collection{}
			m := &store{
				collection: col,
			}
			monkey.PatchInstanceMethod(reflect.TypeOf(col), "DeleteOne", func(_ *mongo.Collection,
				ctx context.Context, filter interface{},
				opts ...deleteopt.Delete) (*mongo.DeleteResult, error) {
				return &mongo.DeleteResult{}, errors.New("fail to DeleteOne")
			})
			err := m.Delete(context.TODO(), "default/key", nil, nil, nil)
			So(err, ShouldBeError)
		})
	})
}

func Test_store_Methods(t *testing.T) {
	Convey("Test store other methods", t, func() {
		col := &mongo.Collection{}
		m := &store{
			collection: col,
		}
		_, err := m.Watch(context.TODO(), "", "", storage.SelectionPredicate{})
		So(err, ShouldBeNil)
		_, err = m.WatchList(context.TODO(), "", "", storage.SelectionPredicate{})
		So(err, ShouldBeNil)
		v := m.Versioner()
		So(v, ShouldBeNil)
		err = m.GetToList(context.TODO(), "", "", storage.SelectionPredicate{}, nil)
		So(err, ShouldBeNil)
		_, err = m.Count("")
		So(err, ShouldBeNil)
	})
}

func Test_decode_encode_LabelKey(t *testing.T) {
	Convey("Test decode encode Label Key", t, func() {
		labels := map[string]string{
			"app": "resourceview",
		}
		encodeLabels := encodeLableKey(map[string]string{})
		So(len(encodeLabels), ShouldEqual, 0)
		encodeLabels = encodeLableKey(labels)
		So(len(encodeLabels), ShouldEqual, len(labels))

		decodeLabels := decodeLableKey(map[string]string{})
		So(len(decodeLabels), ShouldEqual, 0)
		decodeLabels = decodeLableKey(encodeLabels)
		So(len(decodeLabels), ShouldEqual, len(labels))
	})
}

func Test_encodeString(t *testing.T) {
	Convey("Test encodeString", t, func() {
		rst := encodeString("")
		So(len(rst), ShouldEqual, 0)
		rst = encodeString("abc")
		So(len(rst), ShouldNotEqual, 0)
	})
}
