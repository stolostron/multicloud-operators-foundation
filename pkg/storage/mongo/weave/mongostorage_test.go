package weave

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/countopt"
	"github.com/mongodb/mongo-go-driver/mongo/deleteopt"
	"github.com/mongodb/mongo-go-driver/mongo/findopt"
	"github.com/mongodb/mongo-go-driver/mongo/insertopt"
	mcmmongo "github.com/open-cluster-management/multicloud-operators-foundation/pkg/storage/mongo"
	. "github.com/smartystreets/goconvey/convey"
)

func Test_NewMongoStorage(t *testing.T) {
	options := &mcmmongo.Options{
		MongoUser:         "admin",
		MongoPassword:     "abc",
		MongoHost:         "",
		MongoDatabaseName: "",
		MongoSSLCa:        "a.ca",
		MongoSSLCert:      "a.cert",
		MongoSSLKey:       "a.key",
		MongoReplicaSet:   "mcm",
		MongoCollection:   "",
	}
	NewMongoStorage(options, "mcm")

	options = &mcmmongo.Options{
		MongoUser:         "admin",
		MongoPassword:     "abc",
		MongoHost:         "",
		MongoDatabaseName: "",
		MongoSSLCa:        "",
		MongoSSLCert:      "a.cert",
		MongoSSLKey:       "a.key",
		MongoReplicaSet:   "mcm",
		MongoCollection:   "",
	}
	NewMongoStorage(options, "")

	options = &mcmmongo.Options{
		MongoUser:         "",
		MongoPassword:     "",
		MongoHost:         "",
		MongoDatabaseName: "",
		MongoSSLCa:        "",
		MongoSSLCert:      "a.cert",
		MongoSSLKey:       "a.key",
		MongoReplicaSet:   "mcm",
		MongoCollection:   "",
	}
	NewMongoStorage(options, "")
}

func newRealData1() string {
	return base64.StdEncoding.EncodeToString([]byte("abc"))
}

func newRealData2(t *testing.T) string {
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
	return str
}

func newRealData3(t *testing.T) string {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	weaveData := []MongoResource{
		{
			Cluster: "cluster1",
		},
		{
			Cluster: "cluster2",
		},
	}
	compressWeaveData, err := json.Marshal(weaveData)
	if err != nil {
		t.Errorf("failed to marshal data")
	}
	if _, err := gz.Write(compressWeaveData); err != nil {
		t.Errorf("failed to gzip data")
	}
	if err := gz.Flush(); err != nil {
		t.Errorf("failed to flush gz")
	}
	if err := gz.Close(); err != nil {
		t.Errorf("failed to close gz")
	}
	str := base64.StdEncoding.EncodeToString(b.Bytes())
	return str
}

func Test_ExtractUnencodeDecompressUnmarshal(t *testing.T) {
	testCases := []struct {
		name string
		data string
		rst  bool
	}{
		{
			name: "test0",
			data: "abc",
			rst:  false,
		},
		{
			name: "test1",
			data: newRealData1(),
			rst:  false,
		},
		{
			name: "test2",
			data: newRealData2(t),
			rst:  false,
		},
		{
			name: "test3",
			data: newRealData3(t),
			rst:  true,
		},
	}

	for _, testcase := range testCases {
		_, err := ExtractUnencodeDecompressUnmarshal(testcase.data)
		if err != nil && testcase.rst {
			t.Errorf("%s case failed", testcase.name)
		}
		if err == nil && !testcase.rst {
			t.Errorf("%s case failed", testcase.name)
		}
	}
}

func Test_GetMongoResourceByUID(t *testing.T) {
	col := &mongo.Collection{}
	defer monkey.UnpatchAll()
	monkey.PatchInstanceMethod(reflect.TypeOf(col), "FindOne", func(_ *mongo.Collection, ctx context.Context,
		filter interface{}, opts ...findopt.One) *mongo.DocumentResult {
		return &mongo.DocumentResult{}
	})

	mongoStorage := MongoStorage{Collection: col}
	if _, err := mongoStorage.GetMongoResourceByUID("123"); err == nil {
		t.Errorf("GetMongoResourceByUID case fails")
	}
}

func Test_ExtractTransformLoadRemove(t *testing.T) {
	Convey("test ExtractTransformLoadRemove", t, func() {
		Convey("fail to ExtractUnencodeDecompressUnmarshal", func() {
			defer monkey.UnpatchAll()
			monkey.Patch(ExtractUnencodeDecompressUnmarshal, func(string) ([]MongoResource, error) {
				return []MongoResource{}, errors.New("fail to ExtractUnencodeDecompressUnmarshal")
			})
			col := &mongo.Collection{}
			mongoStorage := MongoStorage{Collection: col}
			if err := mongoStorage.ExtractTransformLoadRemove("123"); err == nil {
				t.Errorf("ExtractTransformLoadRemove case fails")
			}
		})
		Convey("fail to InsertAndRemoveOldDocs", func() {
			defer monkey.UnpatchAll()
			monkey.Patch(ExtractUnencodeDecompressUnmarshal, func(string) ([]MongoResource, error) {
				return []MongoResource{}, nil
			})
			col := &mongo.Collection{}
			mongoStorage := MongoStorage{Collection: col}
			monkey.PatchInstanceMethod(reflect.TypeOf(mongoStorage), "InsertAndRemoveOldDocs", func(_ MongoStorage,
				allData []MongoResource) error {
				return errors.New("fail to InsertAndRemoveOldDocs")
			})
			if err := mongoStorage.ExtractTransformLoadRemove("123"); err == nil {
				t.Errorf("InsertAndRemoveOldDocs case fails")
			}
		})
	})
}

func Test_InsertAndRemoveOldDocs(t *testing.T) {
	Convey("test InsertAndRemoveOldDocs", t, func() {
		Convey("fail CountDocuments", func() {
			col := &mongo.Collection{}
			defer monkey.UnpatchAll()
			monkey.PatchInstanceMethod(reflect.TypeOf(col), "CountDocuments", func(_ *mongo.Collection, ctx context.Context,
				filter interface{}, opts ...countopt.Count) (int64, error) {
				return 0, errors.New("fail to CountDocuments")
			})

			mongoStorage := MongoStorage{Collection: col}
			allData := []MongoResource{
				{Cluster: "cluster1"},
			}
			err := mongoStorage.InsertAndRemoveOldDocs(allData)
			So(err, ShouldBeError)
		})
		Convey("fail Decode", func() {
			col := &mongo.Collection{}
			defer monkey.UnpatchAll()
			monkey.PatchInstanceMethod(reflect.TypeOf(col), "CountDocuments", func(_ *mongo.Collection, ctx context.Context,
				filter interface{}, opts ...countopt.Count) (int64, error) {
				return 0, nil
			})
			monkey.PatchInstanceMethod(reflect.TypeOf(col), "FindOne", func(_ *mongo.Collection, ctx context.Context,
				filter interface{}, opts ...findopt.One) *mongo.DocumentResult {
				return &mongo.DocumentResult{}
			})
			monkey.PatchInstanceMethod(reflect.TypeOf(col), "InsertOne", func(_ *mongo.Collection, ctx context.Context, document interface{},
				opts ...insertopt.One) (*mongo.InsertOneResult, error) {
				return nil, errors.New("fail to InsertOne")
			})
			mongoStorage := MongoStorage{Collection: col}
			allData := []MongoResource{
				{Cluster: "cluster1"},
			}
			err := mongoStorage.InsertAndRemoveOldDocs(allData)
			So(err, ShouldBeError)
		})
		Convey("ok Decode", func() {
			col := &mongo.Collection{}
			defer monkey.UnpatchAll()
			monkey.PatchInstanceMethod(reflect.TypeOf(col), "CountDocuments", func(_ *mongo.Collection, ctx context.Context,
				filter interface{}, opts ...countopt.Count) (int64, error) {
				return 0, nil
			})
			monkey.PatchInstanceMethod(reflect.TypeOf(col), "FindOne", func(_ *mongo.Collection, ctx context.Context,
				filter interface{}, opts ...findopt.One) *mongo.DocumentResult {
				return &mongo.DocumentResult{}
			})
			monkey.PatchInstanceMethod(reflect.TypeOf(col), "InsertOne", func(_ *mongo.Collection, ctx context.Context, document interface{},
				opts ...insertopt.One) (*mongo.InsertOneResult, error) {
				return &mongo.InsertOneResult{InsertedID: objectid.ObjectID{}}, nil
			})
			monkey.PatchInstanceMethod(reflect.TypeOf(col), "FindOne", func(_ *mongo.Collection, ctx context.Context,
				filter interface{}, opts ...findopt.One) *mongo.DocumentResult {
				return &mongo.DocumentResult{}
			})
			mongoStorage := MongoStorage{Collection: col}
			allData := []MongoResource{
				{Cluster: "cluster1"},
			}
			err := mongoStorage.InsertAndRemoveOldDocs(allData)
			So(err, ShouldBeError)
		})
	})
}

func Test_RemoveAllMatchingData(t *testing.T) {
	Convey("test RemoveAllMatchingData", t, func() {
		Convey("fail to DeleteMany", func() {
			defer monkey.UnpatchAll()
			col := &mongo.Collection{}
			monkey.PatchInstanceMethod(reflect.TypeOf(col), "DeleteMany", func(_ *mongo.Collection,
				ctx context.Context, filter interface{},
				opts ...deleteopt.Delete) (*mongo.DeleteResult, error) {
				return &mongo.DeleteResult{}, errors.New("fail to DeleteMany")
			})
			mongoStorage := MongoStorage{Collection: col}
			if _, err := mongoStorage.RemoveAllMatchingData(1, objectid.ObjectID{}, "topotype"); err == nil {
				t.Errorf("RemoveAllMatchingData case fails")
			}
		})
	})
}

func Test_RemoveIncorrectClusterIDDocs(t *testing.T) {
	Convey("test RemoveIncorrectClusterIDDocs", t, func() {
		Convey("fail to DeleteMany cluster", func() {
			defer monkey.UnpatchAll()
			col := &mongo.Collection{}
			monkey.PatchInstanceMethod(reflect.TypeOf(col), "DeleteMany", func(_ *mongo.Collection,
				ctx context.Context, filter interface{},
				opts ...deleteopt.Delete) (*mongo.DeleteResult, error) {
				return &mongo.DeleteResult{}, errors.New("fail to DeleteMany")
			})
			mongoStorage := MongoStorage{Collection: col}
			_, err := mongoStorage.RemoveIncorrectClusterIDDocs(objectid.ObjectID{})
			So(err, ShouldBeError)
		})
		Convey("fail to DeleteMany id", func() {
			defer monkey.UnpatchAll()
			col := &mongo.Collection{}
			monkey.PatchInstanceMethod(reflect.TypeOf(col), "DeleteMany", func(_ *mongo.Collection,
				ctx context.Context, filter interface{},
				opts ...deleteopt.Delete) (*mongo.DeleteResult, error) {
				return &mongo.DeleteResult{}, nil
			})

			mongoStorage := MongoStorage{Collection: col}
			_, err := mongoStorage.RemoveIncorrectClusterIDDocs(objectid.ObjectID{})
			So(err, ShouldBeNil)
		})
	})
}

func Test_Construct(t *testing.T) {
	Convey("test Construct", t, func() {
		datum := MongoResource{
			Labels: []MongoLabel{
				{
					Name:  "ype",
					Value: "abc",
				},
			},
			Relationships: []MongoRelationship{
				{
					Cluster: "cl1",
				},
			},
		}
		constructBsonRelationshipsArray(datum, objectid.ObjectID{})
		constructBsonDocuments([]MongoResource{datum}, objectid.ObjectID{})

		s1 := []string{"abc"}
		s2 := []string{"123", "abc"}
		s3 := []string{"123"}
		s4 := []string{"abc"}
		So(stringSliceEquals(s1, s2), ShouldBeFalse)
		So(stringSliceEquals(s1, s3), ShouldBeFalse)
		So(stringSliceEquals(s1, s4), ShouldBeTrue)
		So(containsKey(bson.Keys{bson.Key{Name: "abc", Prefix: s1}}, "abc", s2), ShouldBeFalse)
	})
}
