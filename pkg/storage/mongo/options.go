// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
// <('_')>

package mongo

import (
	"os"

	"github.com/spf13/pflag"
)

// Options defines the flag for mongo
type Options struct {
	MongoUser         string
	MongoPassword     string
	MongoHost         string
	MongoDatabaseName string
	MongoSSLCa        string
	MongoSSLCert      string
	MongoSSLKey       string
	MongoReplicaSet   string
	MongoCollection   string
}

// AddFlags adds flags for ServerRunOptions fields to be specified via FlagSet.
func (m *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&m.MongoCollection, "mongo-collection", m.MongoCollection, ""+
		"Collection name to use for mongodb, default is resources")
	fs.StringVar(&m.MongoUser, "mongo-root-user", m.MongoUser, "Root username for Mongo DB")
	fs.StringVar(&m.MongoPassword, "mongo-root-password", m.MongoPassword, ""+
		"Root password for Mongo DB",
	)
	fs.StringVar(&m.MongoHost, "mongo-host", m.MongoHost, "Host URL & Port for Mongo Endpoint")
	fs.StringVar(&m.MongoDatabaseName, "mongo-database", m.MongoDatabaseName, ""+
		"Database name to use for mongodb, default is mcm",
	)
	fs.StringVar(&m.MongoSSLCa, "mongo-ssl-ca", m.MongoSSLCa, "CA file to connect to mongo")
	fs.StringVar(&m.MongoSSLCert, "mongo-ssl-cert", m.MongoSSLCert, ""+
		"cert file to connect to mongo",
	)
	fs.StringVar(&m.MongoSSLKey, "mongo-ssl-key", m.MongoSSLKey, "key file to connect to mongo")
	fs.StringVar(&m.MongoReplicaSet, "mongo-replicaset", m.MongoReplicaSet, ""+
		"replica set of mongo",
	)
}

// NewMongoOptions create mongooptions
func NewMongoOptions() *Options {
	options := &Options{
		MongoUser:         os.Getenv("MONGO_ROOT_USERNAME"),
		MongoPassword:     os.Getenv("MONGO_ROOT_PASSWORD"),
		MongoSSLCa:        os.Getenv("MONGO_SSLCA"),
		MongoSSLCert:      os.Getenv("MONGO_SSLCERT"),
		MongoSSLKey:       os.Getenv("MONGO_SSLKEY"),
		MongoHost:         "",
		MongoDatabaseName: "mcm",
		MongoReplicaSet:   "",
		MongoCollection:   "",
	}
	return options
}
