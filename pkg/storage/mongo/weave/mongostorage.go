// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

// <('_')>

package weave

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/clientopt"
	"k8s.io/klog"

	mcmmongo "github.com/open-cluster-management/multicloud-operators-foundation/pkg/storage/mongo"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
)

type MongoStorage struct {
	Collection *mongo.Collection
}

const (
	defaultCollectionName = "resources"
)

// Url includes port e.g. 127.0.0.1:27017
func NewMongoStorage(options *mcmmongo.Options, collectionName string) (*MongoStorage, error) {
	var mongoStorage = &MongoStorage{nil}

	if collectionName == "" {
		collectionName = defaultCollectionName
	} else {
		klog.Warning("Using non-default mongo collection name")
	}

	clientOptions := []clientopt.Option{}
	uri := "mongodb://" + options.MongoHost + "/admin"
	if options.MongoSSLCa != "" && options.MongoSSLCert != "" && options.MongoSSLKey != "" {
		curFile, err := utils.GeneratePemFile("/tmp", options.MongoSSLCert, options.MongoSSLKey)
		if err != nil {
			return nil, err
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
		klog.Error("Could not create MongoDB Client: ", err)
		return mongoStorage, err
	}

	err = client.Connect(context.Background())
	if err != nil {
		return nil, err
	}

	mongoStorage.Collection = client.Database(options.MongoDatabaseName).Collection(collectionName)
	//we need the above 3 in case mongo crashes and we need to recreate DB + collection for topology
	klog.Info("Connected to mongo at ", options.MongoHost)

	// This goroutine periodically checks the mongo connection, and refreshes it if it has some problem.
	// Due to an idiosyncrasy of the library, we must also reinitialize the Client object itself.
	go func() {
		for {
			_, err := mongoStorage.Collection.CountDocuments(context.Background(), map[string]string{}) // Check the connection by trying to count docs
			if err != nil {
				discErr := client.Disconnect(context.Background()) // Disconnect old connection
				if discErr != nil {
					klog.Error("Error disconnecting old MongoDB connection: ", discErr.Error())
				}
				client, err = mongo.NewClientWithOptions(uri, clientOptions...)
				if err != nil {
					klog.Error("Could not create MongoDB Client: ", err)
				}
				discErr = client.Connect(context.Background()) // Create new MongoDB connection
				if discErr != nil {
					klog.Error("Error reconnecting to MongoDB: ", discErr.Error())
				}
				mongoStorage.Collection = client.Database(options.MongoDatabaseName).Collection(collectionName)
			}
			time.Sleep(time.Second * 30)
		}
	}()

	return mongoStorage, nil
}

func gUnzip(data []byte) (resData []byte, err error) {
	b := bytes.NewBuffer(data)

	var reader io.Reader
	reader, err = gzip.NewReader(b) // oddly, we do not need to specify the compression level - gzip will figure it out, I guess.
	if err != nil {
		return nil, err
	}
	var resB bytes.Buffer
	_, err = resB.ReadFrom(reader)
	if err != nil {
		return nil, err
	}
	resData = resB.Bytes()
	return resData, nil
}

// Grabs the data out of the observer object and puts it into MongoResource type
// TODO Make this a method of Observer type
func ExtractUnencodeDecompressUnmarshal(data string) ([]MongoResource, error) {
	var decodedCompressedWeaveData []byte
	var weaveData []MongoResource
	decodedCompressedWeaveData, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}
	uncompressedWeaveData, err := gUnzip(decodedCompressedWeaveData)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(uncompressedWeaveData, &weaveData)
	if err != nil {
		return weaveData, err
	}
	return weaveData, nil
}

func constructBsonRelationshipsArray(datum MongoResource, clusterDocID objectid.ObjectID) []*bson.Value {

	values := []*bson.Value{}
	if len(datum.Labels) > 0 {
		for _, rel := range datum.Relationships {
			bsonVal := bson.VC.DocumentFromElements(
				bson.EC.String("name", rel.Name),
				bson.EC.ObjectID("cluster", clusterDocID),
				bson.EC.String("namespace", rel.Namespace),
				bson.EC.String("topology", rel.Topology),
				bson.EC.String("type", rel.Type),
				bson.EC.String("uid", rel.UID),
				bson.EC.ArrayFromElements("labels", constructBsonLabels(datum)...),
			)
			values = append(values, bsonVal)
		}
		return values
	}
	return nil
}
func constructBsonLabels(datum MongoResource) []*bson.Value {
	values := []*bson.Value{}
	if len(datum.Labels) > 0 {
		for _, rel := range datum.Labels {
			bsonVal := bson.VC.DocumentFromElements(
				bson.EC.String("name", rel.Name),
				bson.EC.String("value", rel.Value),
			)
			values = append(values, bsonVal)
		}
		return values
	}
	return nil
}
func constructBsonDocuments(allData []MongoResource, clusterDocID objectid.ObjectID) []interface{} {

	allDocuments := []interface{}{}
	for _, datum := range allData {

		singleDoc := bson.NewDocument(
			bson.EC.String("topology", datum.Topology),
			bson.EC.ArrayFromElements("labels", constructBsonLabels(datum)...),
			bson.EC.String("name", datum.Name),
			bson.EC.String("type", datum.Type),
			bson.EC.String("uid", datum.UID),
			bson.EC.ArrayFromElements("relationships", constructBsonRelationshipsArray(datum, clusterDocID)...), // TODO Pass cluster ObjectID
		)

		if datum.Type != "cluster" {
			singleDoc.Append(bson.EC.ObjectID("cluster", clusterDocID),
				bson.EC.Int64("epoch", datum.Epoch),
				bson.EC.String("namespace", datum.Namespace))
		}

		allDocuments = append(allDocuments, singleDoc)

	}

	return allDocuments

}

func (m MongoStorage) GetMongoResourceByUID(uid string) (MongoResource, error) {
	var ret MongoResource
	result := m.Collection.FindOne(context.Background(), map[string]string{"uid": uid})
	err := result.Decode(&ret)
	if err != nil {
		return ret, err
	}
	return ret, nil
}

func stringSliceEquals(s1 []string, s2 []string) bool {
	if len(s1) != len(s2) {
		return false
	}

	for i := range s1 {
		if s1[i] != s2[i] {
			return false
		}
	}

	return true
}

func containsKey(keys bson.Keys, key string, prefix []string) bool {
	for _, k := range keys {
		if k.Name == key && stringSliceEquals(k.Prefix, prefix) {
			return true
		}
	}

	return false
}
func (m MongoStorage) createClusterDoc(clusterName string) MongoResource {
	klog.V(5).Info("Creating cluster", clusterName)

	mr := MongoResource{}
	mr.Cluster = clusterName
	mr.Epoch = time.Now().Unix()
	mr.Labels = []MongoLabel{}
	mr.Name = clusterName
	mr.Relationships = []MongoRelationship{}
	mr.Topology = "cluster"
	mr.Type = "cluster"
	mr.UID = clusterName
	return mr
}

func (m MongoStorage) InsertAndRemoveOldDocs(allData []MongoResource) error {
	resource := allData[0]
	var foundClusterDoc bool
	var createdClusterDoc MongoResource
	var insertID objectid.ObjectID
	_, err := m.Collection.CountDocuments(context.Background(), map[string]string{}) //not great, but verify if collection even exists
	if err != nil {
		return err
	}

	clusterName, epoch := resource.Cluster, resource.Epoch //replace with updated epoch
	klog.V(5).Info("Inserting ", len(allData), " docs into mongodb for cluster: ", clusterName)
	clusterDoc := bson.NewDocument()

	clusterDocResult := m.Collection.FindOne(context.Background(), map[string]string{"type": "cluster", "uid": clusterName})
	err = clusterDocResult.Decode(clusterDoc)
	if err != nil {
		if err.Error() == "mongo: no documents in result" { // string checking error, don't tell my mom :)
			foundClusterDoc = false
		} else {
			klog.Error("Error decoding cluster document found for cluster: ", resource.Cluster)
			return err
		}
	} else {
		foundClusterDoc = true
	}

	var clusterDocID objectid.ObjectID
	if foundClusterDoc {
		//we found an existing cluster doc so use it
		keys, err := clusterDoc.Keys(false)
		if err != nil || !containsKey(keys, "_id", nil) {
			klog.Error("Error reading cluster document for cluster: ", resource.Cluster)
			return err
		}
		clusterDocID = clusterDoc.Lookup("_id").ObjectID()
	} else {
		//we couldn't find a doc so create and insert a doc and get the id
		createdClusterDoc = m.createClusterDoc(clusterName)
		insertResult, err := m.Collection.InsertOne(context.Background(), createdClusterDoc)
		if err != nil {
			klog.Error("Had an issue inserting new cluster doc on mongo start")
			return err
		}
		insertID = insertResult.InsertedID.(objectid.ObjectID)
		findResult := m.Collection.FindOne(context.Background(), map[string]string{"type": "cluster", "uid": clusterName})
		clusterDoc := bson.NewDocument()
		err = findResult.Decode(clusterDoc)
		if err != nil {
			klog.Error("Error decoding cluster document found for cluster: ", clusterName)
			return err
		}
		clusterDocID = insertID
		foundClusterDoc = true
	}

	allDocuments := constructBsonDocuments(allData, clusterDocID)

	// TODO First order of business after GA is to fix this garbage
	if resource.Type != "cluster" && !foundClusterDoc {
		// Someone tried to put in data before the cluster doc. Can't do that
		klog.Error("No cluster doc for cluster: ", clusterName)
		return err
	} else if resource.Type == "cluster" && !foundClusterDoc {
		// This is the first run. Insert the cluster doc
		klog.V(5).Info("Intializing mongoDB for new cluster: ", resource.UID)
		_, err := m.Collection.InsertMany(context.Background(), allDocuments)
		if err != nil {
			return err
		}
	} else if resource.Type == "cluster" && foundClusterDoc {
		// Delete old docs because this doc indicates the klusterlet restarted
		if _, err := m.RemoveIncorrectClusterIDDocs(clusterDocID); err != nil {
			return err
		}

		// Insert new cluster doc
		_, err := m.Collection.InsertMany(context.Background(), allDocuments)
		if err != nil {
			return err
		}
	} else {
		// Insert docs with clusterDocId equal to the _id of the cluster doc we found
		_, err := m.Collection.InsertMany(context.Background(), allDocuments)
		if err != nil {
			return err
		}
		// Delete old docs with dated epoch but the same cluster field
		if _, err := m.RemoveAllMatchingData(epoch, clusterDocID, resource.Type); err != nil {
			return err
		}
	}
	return nil

}

//Extracts mongo resources from ClusterStatusTopology object
//Inserts all the new mongo resources
//Deletes all the old ones
func (m MongoStorage) ExtractTransformLoadRemove(data string) error {
	allData, err := ExtractUnencodeDecompressUnmarshal(data)
	if err != nil {
		return err
	}
	err = m.InsertAndRemoveOldDocs(allData)
	if err != nil {
		return err
	}
	return nil
}

// Deletes anything with cluster field equal to given clusterDocId which does not have epoch field equal to given epoch.
func (m MongoStorage) RemoveAllMatchingData(epoch int64, clusterDocID objectid.ObjectID, topotype string) (bool, error) {
	res, err := m.Collection.DeleteMany(context.Background(), bson.NewDocument(
		bson.EC.ObjectID("cluster", clusterDocID),
		bson.EC.Interface("epoch", map[string]int64{"$ne": epoch}),
		bson.EC.Interface("type", map[string]string{"$eq": topotype})))
	klog.V(5).Info("Removing old data", res.DeletedCount)
	if err != nil {
		return false, err
	}
	return true, nil
}

// Removes anything with a cluster field pointing to the given objectID
func (m MongoStorage) RemoveIncorrectClusterIDDocs(clusterDocID objectid.ObjectID) (bool, error) {

	// delete the old docs pointing to old cluster doc
	res, err := m.Collection.DeleteMany(context.Background(), bson.NewDocument(
		bson.EC.ObjectID("cluster", clusterDocID)))
	klog.V(5).Info(res.DeletedCount)
	if err != nil {
		return false, err
	}

	// delete the old cluster doc itself
	res, err = m.Collection.DeleteMany(context.Background(), bson.NewDocument(
		bson.EC.ObjectID("_id", clusterDocID)))
	klog.V(5).Info(&res.DeletedCount)

	if err != nil {
		return false, err
	}

	return true, nil
}
