package app

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
	auth2 "yaybackEnd/auth"
	"yaybackEnd/helpers"
	"yaybackEnd/model"
)

type RelationManager struct {
	httpClient http.Client
	*AuthManager
	RelationManagerRepository
}

func NewRelationManager(httpClient http.Client,authManager *AuthManager, repository RelationManagerRepository) *RelationManager{
	newRelationManager := new(RelationManager)
	newRelationManager.httpClient = httpClient
	newRelationManager.AuthManager = authManager
	newRelationManager.RelationManagerRepository = repository
	return  newRelationManager
}

func (r *RelationManager) getFollowedArtistOnSpotify(user *model.User) []*model.Artist {
	followedArtistUrl := "https://api.spotify.com/v1/me/following?type=artist"

	followedArtistReq, _ := http.NewRequest("GET", followedArtistUrl, nil)

	accessToken, accessTokenErr := r.AuthManager.GetAccessToken(user)

	if accessTokenErr != nil {
		log.Fatal(accessTokenErr)
	}

	followedArtistReq.Header.Set("Authorization", "Bearer "+accessToken)

	followedArtistRes, _ := r.httpClient.Do(followedArtistReq)

	followedArtistResBody, followedArtistReqErr := ioutil.ReadAll(followedArtistRes.Body)

	if followedArtistReqErr != nil {
		log.Fatal(followedArtistReqErr)
	}

	log.Printf("followed artist : %v", string(followedArtistResBody))

	var followedArtistJson map[string]interface{}

	followedArtistJsonMarshalErr := json.Unmarshal(followedArtistResBody, &followedArtistJson)

	if followedArtistJsonMarshalErr != nil {
		// TODO must handle error
		log.Fatal(followedArtistReqErr)
	}

	var artistList []*model.Artist

	artistsJSON := followedArtistJson["artists"].(map[string]interface{})["items"].([]map[string]interface{})

	for _, artistAccountData := range artistsJSON {
		artistID := artistAccountData["id"].(string)
		artist := r.RelationManagerRepository.GetArtistBySpotifyID(artistID)

		if artist == nil {
			addArtistErr := r.addArtist(artistAccountData, user)
			if addArtistErr != nil {
				// TODO MUST HANDLE ERROR
				log.Fatal(addArtistErr)
				///continue
				/// if artist is not on twitter we might just move onto the next artist in the list
			}
		}

		artistList = append(artistList, artist)
	}
	return artistList
}
func (r *RelationManager) UpdateFollowedArtistList(user *model.User) {
	followedArtistSpotify := r.getFollowedArtistOnSpotify(user)
	// add the followed artist
	for i, _ := range followedArtistSpotify {
		artist := followedArtistSpotify[i]

		isFollowing := r.RelationManagerRepository.IsFollowingArtist(user, artist)

		if isFollowing {
			continue
		}

		r.followArtist(user, artist)
	}

	// TODO must remove the artists that user has unfollowed via the Spotify app
}

func (r *RelationManager) requestArtistSpotifyAccount(artistSpotifyID string) {
}
func (r *RelationManager) requestArtistTwitterAccount(artistSpotifyName string, user *model.User) (map[string]interface{}, error) {
	twitterUserOauthToken, twitterUserOauthTokenSecret, _ := r.AuthManager.GetUserTwitterOauth(user.GetUserUUID())

	twitterArtistSearchUrl := "https://api.twitter.com/1.1/users/search.json"

	params := url.Values{}

	params.Set("q", artistSpotifyName)
	params.Set("count", "1")

	oauthParams := url.Values{}

	oauthParams.Add("oauth_consumer_key", auth2.TwitterApiKey)
	oauthParams.Add("oauth_nonce", strconv.FormatInt(time.Now().Unix(), 10))
	oauthParams.Add("oauth_version", "1.0")
	oauthParams.Add("oauth_signature_method", "HMAC-SHA1")
	oauthParams.Add("oauth_version", "1.0")
	oauthParams.Add("oauth_token", twitterUserOauthToken)
	oauthParams.Add("oauth_timestamp", strconv.FormatInt(time.Now().Unix(), 10))

	_, oauthHeader := helpers.OauthSignature("GET", twitterArtistSearchUrl, TwitterApiKey, twitterUserOauthTokenSecret, params, oauthParams)

	artistReq, _ := http.NewRequest("GET", twitterArtistSearchUrl, nil)

	artistReq.URL.RawQuery = params.Encode()
	artistReq.Header.Add("Authorization", oauthHeader)

	searchedArtistResponse, searchedArtistErr := r.httpClient.Do(artistReq)

	if searchedArtistErr != nil {
		log.Fatal(searchedArtistErr.Error())
	}

	var searchedArtistBytes []byte
	var searchedArtistJson interface{}

	searchedArtistBytes, _ = ioutil.ReadAll(searchedArtistResponse.Body)

	json.Unmarshal(searchedArtistBytes, &searchedArtistJson)

	return searchedArtistJson.([]map[string]interface{})[0], nil
}

func (r *RelationManager) addArtist(artistSpotifyAccountData map[string]interface{}, user *model.User) error {
	artistSpotifyID := artistSpotifyAccountData["id"].(string)
	artistTwitterAccountData, artistTwitterAccountDataErr := r.requestArtistTwitterAccount(artistSpotifyID, user)

	if artistTwitterAccountDataErr != nil {
		//TODO must handle error
		log.Fatal(artistTwitterAccountDataErr)
	}

	artistAccountData := map[string]interface{}{
		"id":              artistSpotifyID,
		"spotify_account": artistSpotifyAccountData,
		"twitter_account": artistTwitterAccountData,
		"followers":       []interface{}{},
	}
	return r.RelationManagerRepository.AddArtist(artistAccountData)
}

func (r *RelationManager) followArtist(user *model.User, artist *model.Artist) bool {
	if r.IsFollowingArtist(user, artist) {
		log.Printf("artistExist")
		return true
	}

	r.RelationManagerRepository.FollowArtist(user, artist)

	return true
}

type RelationManagerRepository interface {
	GetFollowedArtist(user *model.User) []*model.Artist
	IsFollowingArtist(user *model.User, artist *model.Artist) bool
	FollowArtist(user *model.User, artist *model.Artist) error
	GetArtistBySpotifyID(spotifyID string) *model.Artist
	GetArtistByTwitterID(spotifyID string) *model.Artist
	AddArtist(data map[string]interface{}) error
}
