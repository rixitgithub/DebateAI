package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User defines a user entity
type User struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Email       string             `bson:"email" json:"email"`
	DisplayName string             `bson:"displayName" json:"displayName"`
	Bio         string             `bson:"bio" json:"bio"`
	EloRating   int                `bson:"eloRating" json:"eloRating"`
	AvatarURL   string             `bson:"avatarUrl,omitempty" json:"avatarUrl,omitempty"`
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
}
