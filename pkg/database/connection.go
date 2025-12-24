package database

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type DB struct {
	Client *mongo.Client
	DB     *mongo.Database
	Ctx    context.Context
}

func Connect(uri string, dbName string) *DB {
	// Set client options
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(uri).SetServerAPIOptions(serverAPI)

	// Creates a new client and connects to the server
	client, err := mongo.Connect(opts)
	if err != nil {
		panic(err)
	}

	db := client.Database(dbName)

	return &DB{
		Client: client,
		DB:     db,
		Ctx:    context.Background(),
	}
}

func (db *DB) Disconnect() {
	if err := db.Client.Disconnect(db.Ctx); err != nil {
		panic(err)
	}
}

func (db *DB) GetOne(collection string, kind string, name string, result interface{}) error {
	filter := bson.M{"meta.kind": kind, "meta.name": name}
	err := db.DB.Collection(collection).FindOne(db.Ctx, filter).Decode(result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return ErrNotFound
		}
		return err
	}

	return nil
}

func (db *DB) Create(collection string, document interface{}) error {
	_, err := db.DB.Collection(collection).InsertOne(db.Ctx, document)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrDocumentExists
		}
		return err
	}

	return nil
}

func (db *DB) Update(collection string, kind string, name string, version int32, document interface{}) error {
	filter := bson.M{"meta.version": version, "meta.kind": kind, "meta.name": name}
	result, err := db.DB.Collection(collection).ReplaceOne(db.Ctx, filter, document)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return ErrOptimisticLock
	}

	return nil
}
