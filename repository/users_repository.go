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
		"id":              user.GetUserUUID(),
		"spotify_account": user.SpotifyAccount,
		"twitter_account": user.TwitterAccount,
	}, firestore.MergeAll)

	// SETTING THE USER PROFILE
	_, _ = u.db.Collection("user_profiles").Doc(user.GetUserUUID()).Set(u.ctx, map[string]interface{}{

		"spotify_user_name":  user.SpotifyAccount["display_name"],
		"profile_picture":  user.SpotifyAccount["profile_picture"],
		"twitter_user_name":  user.TwitterAccount["screen_name"],
		"twitter_id":  user.TwitterAccount["user_id"],
		"user_desc":  "",
	},firestore.MergeAll)

	return writeError
}

func (u UserFireStoreRepository) UpdateSpotifyOauthInfo(user model.User, accessToken string, accessTokenTimeStamp int64) error {
	_, updateErr := u.db.Collection("users").Doc(user.GetUserUUID()).Set(context.Background(), map[string]interface{}{

		"spotify_account": map[string]interface{}{
			"access_token":     accessToken,
			"token_time_stamp": accessTokenTimeStamp,
		}}, firestore.MergeAll)

	return updateErr

}

func (u UserFireStoreRepository) UpdateUserTops(user *model.User, userTops map[string]interface{}) error {
	var err error
	userProfileCol := u.db.Collection("user_profiles").Doc(user.GetUserUUID())
	userTopTrack := userProfileCol.Collection("tops").Doc("tracks")
	userTopArtist := userProfileCol.Collection("tops").Doc("artists")

	topTracks := userTops["tracks"].(map[string]interface{})
	topArtists := userTops["artists"].(map[string]interface{})

	_, err = userTopArtist.Set(u.ctx, topArtists)

	if err != nil {
		log.Printf("user top update err : \n%v", err)
	}

	_, err = userTopTrack.Set(u.ctx, topTracks)
	if err != nil {
		log.Printf("user top update err : \n%v", err)
	}

	return err
}
func (u UserFireStoreRepository) GetUserProfile(user *model.User)(map[string]interface{}, error){
	var userProfile =  make(map[string]interface{})
	userProfileDoc := u.db.Collection("user_profiles").Doc(user.GetUserUUID())

	topArtists, topArtistsErr := userProfileDoc.Collection("tops").Doc("artists").Get(u.ctx)

	topTracks, topTracksErr := userProfileDoc.Collection("tops").Doc("tracks").Get(u.ctx)

	if topTracksErr == nil{
		userProfile["topTracks"] = topTracks.Data()

	}

	if topArtistsErr == nil{
		userProfile["topArtists"] = topArtists.Data()
	}

	userProfileSnapShot, userProfileSnapShotErr := userProfileDoc.Get(u.ctx)

	if userProfileSnapShotErr == nil{
		userProfile["basic"] = userProfileSnapShot.Data()
	}


	return userProfile,nil

}