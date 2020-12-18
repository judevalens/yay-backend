package app

import (
	"cloud.google.com/go/firestore"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"yaybackEnd/artistManager"
	"yaybackEnd/model"
)

type RelationManager struct {
	httpClient *http.Client
		*AuthManager
}


func(r *RelationManager) getFollowedArtistOnSpotify(user *model.User)[]model.Artist{
	followedArtistUrl := "https://api.spotify.com/v1/me/following?type=artist"

	followedArtistReq, _ := http.NewRequest("GET", followedArtistUrl, nil)

	accessToken, accessTokenErr := r.AuthManager.GetAccessToken(user)

	if accessTokenErr != nil{
		log.Fatal(accessTokenErr)
	}

	followedArtistReq.Header.Set("Authorization", "Bearer "+ accessToken)

	followedArtistRes, _ := r.httpClient.Do(followedArtistReq)

	followedArtistResBody, followedArtistReqErr := ioutil.ReadAll(followedArtistRes.Body)

	if followedArtistReqErr != nil {
		log.Fatal(followedArtistReqErr)
	}

	log.Printf("followed artist : %v", string(followedArtistResBody))

	var followedArtistJson map[string]interface{}

	_ = json.Unmarshal(followedArtistResBody, &followedArtistJson)

	var artistList []model.Artist

	for _, artistAccountData := range followedArtistJson {
		artistList = append(artistList,model.NewArtist(artistAccountData.(map[string]interface{})))
	}


	return artistList
}


func (r *RelationManager) updateFollowedArtistList(user *model.User) {
	followedArtistSpotify := r.getFollowedArtistOnSpotify(user)

	userDoc := r.fireStoreDB.Collection("users").Doc(userID)

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

type RelationManagerRepositoryI interface {
	GetFollowedArtist(user model.User)[]model.Artist
	IsFollowingArtist(user model.User,artist model.Artist)bool
}