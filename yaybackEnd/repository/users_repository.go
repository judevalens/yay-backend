package repository

import (
	"cloud.google.com/go/firestore"
	"context"
	"log"
	"yaybackEnd/model"
)

type UserFireStoreRepository struct {
	db  *firestore.Client
	ctx context.Context
}

func NewUserFireStoreRepository(db *firestore.Client, ctx context.Context) *UserFireStoreRepository {
	newUserFireStoreRepository := new(UserFireStoreRepository)
	newUserFireStoreRepository.db = db
	newUserFireStoreRepository.ctx = ctx
	return newUserFireStoreRepository
}

func (u UserFireStoreRepository) GetUserBySpotifyID(spotifyID string) *model.User {
	panic("implement me")
}

func (u UserFireStoreRepository) GetUserByTwitterID(twitterID string) *model.User {
	panic("implement me")
}

func (u UserFireStoreRepository) GetUserSpotifyAccessToken(uuid string) string {
	user, userErr := u.GetUserByUUID(uuid)

	if userErr != nil {
		// TODO must handle error
		log.Fatal(userErr)
	}
	return user.GetSpotifyAccount()["access_token"].(string)
}

func (u UserFireStoreRepository) GetUserByUUID(uuid string) (*model.User, error) {
	log.Printf("uuid : %v", uuid)
	usersCol := u.db.Collection("users").Doc(uuid)
	userDocSnapShot, userDocSnapShotErr := usersCol.Get(u.ctx)

	if userDocSnapShotErr != nil {
		return nil, userDocSnapShotErr
	}

	if !userDocSnapShot.Exists() {
		return nil, userDocSnapShotErr
	}

	UserData := userDocSnapShot.Data()
	spotifyData := UserData["spotify_account"].(map[string]interface{})
	twitterData := UserData["twitter_account"].(map[string]interface{})

	return model.NewUser(uuid, spotifyData, twitterData), nil

}

func (u UserFireStoreRepository) GetUserTwitterOauth(uuid string) (string, string, error) {
	user, userErr := u.GetUserByUUID(uuid)

	if userErr != nil {
		// TODO must handle error
		log.Fatal(userErr)
	}

	oauthToken, oauthSecret := user.GetUserTwitterOauth()

	return oauthToken, oauthSecret, nil

}

func (u UserFireStoreRepository) AddUser(user model.User) error {
	_, writeError := u.db.Collection("users").Doc(user.GetUserUUID()).Set(context.Background(), map[string]interface{}{
		"id": user.GetUserUUID(),
		"spotify_account": map[string]interface{}{
			"access_token":     user.SpotifyAccount["access_token"],
			"refresh_token":    user.SpotifyAccount["refresh_token"],
			"token_time_stamp": int64(user.SpotifyAccount["token_time_stamp"].(float64)),
		},
		"twitter_account": user.TwitterAccount,
	}, firestore.MergeAll)

	return writeError
}

func (u UserFireStoreRepository) UpdateSpotifyOauthInfo(user model.User, accessToken string, accessTokenTimeStamp int64) error {
	_, updateErr := u.db.Collection("users").Doc(user.GetUserUUID()).Set(context.Background(), map[string]interface{}{

		"spotify_account": map[string]interface{}{
			"access_token":     accessToken,
			"token_time_stamp": accessTokenTimeStamp,
		}},firestore.MergeAll)

	return updateErr

}
