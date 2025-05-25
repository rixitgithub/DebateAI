package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserDebateResult struct {
	UserID       primitive.ObjectID `bson:"userId" json:"userId"`
	RatingChange int                `bson:"ratingChange" json:"ratingChange"`
}

type UserDebate struct {
	ID             primitive.ObjectID  `bson:"_id,omitempty" json:"id,omitempty"`
	Type           string              `bson:"type" json:"type"`
	Title          string              `bson:"title" json:"title"`
	User1ID        primitive.ObjectID  `bson:"user1Id" json:"user1Id"`
	User2ID        *primitive.ObjectID `bson:"user2Id,omitempty" json:"user2Id,omitempty"`
	BotID          *primitive.ObjectID `bson:"botId,omitempty" json:"botId,omitempty"`
	Winner         *DebateResult       `bson:"winner,omitempty" json:"winner,omitempty"`
	Loser          *DebateResult       `bson:"loser,omitempty" json:"loser,omitempty"`
	PhaseDetailsID primitive.ObjectID  `bson:"phaseDetailsId" json:"phaseDetailsId"`
	Timestamp      time.Time           `bson:"timestamp" json:"timestamp"`
}
