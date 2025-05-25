package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PhaseEntry struct {
	Type      string    `bson:"type" json:"type"`
	Speaker   string    `bson:"speaker" json:"speaker"`
	Content   string    `bson:"content" json:"content"`
	Timestamp time.Time `bson:"timestamp" json:"timestamp"`
}

type PhaseDetailsDocument struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	DebateID     primitive.ObjectID `bson:"debateId" json:"debateId"`
	PhaseDetails []PhaseEntry       `bson:"phaseDetails" json:"phaseDetails"`
}
