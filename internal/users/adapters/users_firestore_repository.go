package adapters

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/shotokan/firebase-training/internal/users/models"
)

type UserRepository struct {
	firestoreClient *firestore.Client
}

func NewUserFirestoreRepository(firestoreClient *firestore.Client) *UserRepository {
	return &UserRepository{
		firestoreClient: firestoreClient,
	}
}

func (repo UserRepository) AddUser(ctx context.Context, user models.User) error {
	collection := repo.userCollection()

	userDto := repo.marshalTraining(user)

	return repo.firestoreClient.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		return tx.Create(collection.Doc(userDto.ID), userDto)
	})
}

func (repo UserRepository) marshalTraining(user models.User) User {
	return User{
		ID:       user.ID.String(),
		Name:     user.Name,
		Email:    user.Email,
		Password: user.Password,
	}
}

func (repo UserRepository) userCollection() *firestore.CollectionRef {
	return repo.firestoreClient.Collection("users")
}
