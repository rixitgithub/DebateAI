package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RatingHistory struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	UserID    primitive.ObjectID `bson:"userId" json:"userId"`
	DebateID  primitive.ObjectID `bson:"debateId" json:"debateId"`
	OldRating int                `bson:"oldRating" json:"oldRating"`
	NewRating int                `bson:"newRating" json:"newRating"`
	Reason    string             `bson:"reason" json:"reason"`
	Timestamp time.Time          `bson:"timestamp" json:"timestamp"`
}
