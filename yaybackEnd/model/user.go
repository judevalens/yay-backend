package model

import (
	"cloud.google.com/go/firestore"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"yaybackEnd/artistManager"
	"yaybackEnd/model/FeedContent"
)
type User struct {
	uuid string
	SpotifyID string
	TwitterID string
	SpotifyAccount map[string]interface{}
	TwitterAccount map[string]interface{}
	UsersRepositoryI
}

func NewUser(uuid string, spotifyAccountData,twitterAccountData map[string]interface{})*User {
	newUser := new(User)
	newUser.uuid = uuid
	newUser.SpotifyAccount = spotifyAccountData
	newUser.TwitterAccount = twitterAccountData
	return  newUser
}

func NewUserByUUID(uuid string)*User {
	return nil
}

func(user *User) Authenticate()  {
}

func(user *User) updateFollowedArtistList()  {
	accessToken, _ := artistManager.authenticator.GetSpotifyRefreshedAccessToken(userID)
	followedArtistSpotify := (artistManager.getFollowedArtistOnSpotify(accessToken)["artists"]).(map[string]interface{})["items"].([]interface{})

	userDoc := artistManager.fireStoreDB.Collection("users").Doc(userID)

	userDocSnapShot, _ := userDoc.Get(artistManager.ctx)
	followedArtist, _ := userDocSnapShot.DataAt("followed_artist")

	if followedArtist == nil {
		userDoc.Set(artistManager.ctx, map[string]interface{}{
			"followed_artist": []interface{}{},
		}, firestore.MergeAll)
	}

	// add the followed artist
	for i, _ := range followedArtistSpotify {
		artist := followedArtistSpotify[i].(map[string]interface{})
		artistID := artist["id"].(string)

		artistQuerySnapShot, artistQueryErr := artistManager.fireStoreDB.Collection("users").Where("id", "==", userID).Where("followed_artist", "array-contains", artistID).Documents(artistManager.ctx).GetAll()

		if artistQueryErr != nil {
			log.Fatal(artistQueryErr)
		}

		if len(artistQuerySnapShot) > 0 {
			continue
		}

		artistManager.followArtist(artist, artist["name"].(string), userID)

		userDoc.Update(artistManager.ctx, []firestore.Update{
			{Path: "followed_artist",
				Value: firestore.ArrayUnion(artistID),
			},
		})
	}
}

func (user *User) followArtist(Artist *Artist)  {
}
func (user *User) GetSpotifyToken()  {
}


func (user *User) getSpotifyUserInfo(){
}

func (user *User) GetUserUUID() string {
	return user.uuid
}
func (user *User) addContentToFeed(content FeedContent.Content)  {
}

func (user *User) UpdateSpotifyOauthInfo(accessToken string, accessTokenTimeStamp int64){

}

func (user *User) GetAccessTokenTimeStamp() int64{
	return 0
}

func (user *User) GetSpotifyAccount() map[string]interface{}{
	return user.SpotifyAccount
}

func (user *User) GetUserTwitterOauth()(string,string){
	// TODO check if values exists
	return user.SpotifyAccount["oauth_token"].(string),user.SpotifyAccount["oauth_secret"].(string)
}


type UsersRepositoryI interface {
	GetID()string
}
