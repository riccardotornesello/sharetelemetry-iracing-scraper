package database

import "time"

type Meta struct {
	Version   int32     `bson:"version"`
	CreatedAt time.Time `bson:"created_at"`

	Kind   string                 `bson:"kind,omitempty"`
	Name   string                 `bson:"name,omitempty"`
	Labels map[string]interface{} `bson:"labels,omitempty"`
}
