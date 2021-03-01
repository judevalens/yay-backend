package app

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
	"yaybackEnd/helpers"
	"yaybackEnd/model"
)

type RelationManager struct {
	httpClient http.Client
	*AuthManager
	RelationManagerRepository
	UserSearchService
}

func NewRelationManager(httpClient http.Client, authManager *AuthManager, repository RelationManagerRepository, userSearchService UserSearchService) *RelationManager {
	newRelationManager := new(RelationManager)
	newRelationManager.httpClient = httpClient
	newRelationManager.AuthManager = authManager
	newRelationManager.RelationManagerRepository = repository
	newRelationManager.UserSearchService = userSearchService
	return newRelationManager
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

	//log.Printf("followed artist : %v", string(followedArtistResBody))

	var followedArtistJson map[string]interface{}

	followedArtistJsonMarshalErr := json.Unmarshal(followedArtistResBody, &followedArtistJson)

	if followedArtistJsonMarshalErr != nil {
		// TODO must handle error
		log.Fatal(followedArtistReqErr)
	}

	var artistList []*model.Artist

	//log.Printf("%v", followedArtistJson["artists"].(map[string]interface{}))

	artistsJSON := followedArtistJson["artists"].(map[string]interface{})["items"].([]interface{})

	for _, artistAccountData := range artistsJSON {
		var artist *model.Artist
		var addArtistErr error
		artistID := artistAccountData.(map[string]interface{})["id"].(string)
		artist, _ = r.RelationManagerRepository.GetArtistBySpotifyID(artistID)

		if artist == nil {
			artist, addArtistErr = r.addArtist(artistAccountData.(map[string]interface{}), user)
			if addArtistErr != nil {
				// TODO MUST HANDLE ERROR
				log.Print("addArtistErr")
				log.Print(addArtistErr)
				continue
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

		followArtistError := r.followArtist(user, artist)

		if followArtistError != nil {
			//TODO must handle error
			log.Print(followArtistError)
		}
	}

	// TODO must remove the artists that user has unFollowed via the Spotify app
	// TODO must decide what to return
}

func (r *RelationManager) requestArtistSpotifyAccount(artistSpotifyID string) {
}
func (r *RelationManager) requestArtistTwitterAccount(artistSpotifyName string, user *model.User) (map[string]interface{}, error) {
	twitterUserOauthToken, twitterUserOauthTokenSecret, _ := r.AuthManager.GetUserTwitterOauth(user.GetUserUUID())

	twitterArtistSearchUrl := "https://api.twitter.com/1.1/users/search.json"

	params := url.Values{}

	params.Set("q", artistSpotifyName)
	params.Set("count", "1")
	params.Set("include_entities", "false")

	oauthParams := url.Values{}

	oauthParams.Add("oauth_consumer_key", TwitterApiKey)
	oauthParams.Add("oauth_nonce", strconv.FormatInt(time.Now().Unix(), 10))
	oauthParams.Add("oauth_version", "1.0")
	oauthParams.Add("oauth_signature_method", "HMAC-SHA1")
	oauthParams.Add("oauth_version", "1.0")
	oauthParams.Add("oauth_token", twitterUserOauthToken)
	oauthParams.Add("oauth_timestamp", strconv.FormatInt(time.Now().Unix(), 10))

	_, oauthHeader := helpers.OauthSignature("GET", twitterArtistSearchUrl, TwitterSecretKey, twitterUserOauthTokenSecret, params, oauthParams)

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
	//log.Printf("searched artist name : %v, \n%v", artistSpotifyName, string(searchedArtistBytes))

	searchedArtistUnmarshalErr := json.Unmarshal(searchedArtistBytes, &searchedArtistJson)

	if searchedArtistUnmarshalErr != nil {
		// TODO handle error
		log.Fatal(searchedArtistUnmarshalErr)
	}

	searchResult := searchedArtistJson.([]interface{})

	if len(searchResult) > 0 {
		return searchResult[0].(map[string]interface{}), nil
	} else {
		return nil, errors.New("no artist found")
	}

}

func (r *RelationManager) SearchUsers(query string) ([]map[string]interface{}, error) {
	log.Printf("searching for user : %v", query)
	return r.UserSearchService.SearchUsers(query)
}

func (r *RelationManager) addArtist(artistSpotifyAccountData map[string]interface{}, user *model.User) (*model.Artist, error) {
	artistSpotifyID := artistSpotifyAccountData["id"].(string)
	artistSpotifyName := artistSpotifyAccountData["name"].(string)
	artistTwitterAccountData, artistTwitterAccountDataErr := r.requestArtistTwitterAccount(artistSpotifyName, user)

	if artistTwitterAccountDataErr != nil {
		//TODO must handle error
		log.Print(artistTwitterAccountDataErr)
		return nil, artistTwitterAccountDataErr

	}

	artistAccountData := map[string]interface{}{
		"id":              artistSpotifyID,
		"spotify_account": artistSpotifyAccountData,
		"twitter_account": artistTwitterAccountData,
		"followers":       []interface{}{},
	}

	artist := model.NewArtist(artistAccountData)
	return artist, r.RelationManagerRepository.AddArtist(artistAccountData, artistSpotifyID)
}

func (r *RelationManager) followArtist(user *model.User, artist *model.Artist) error {
	if r.IsFollowingArtist(user, artist) {
		log.Printf("artistExist")
		return nil
	}
	return r.RelationManagerRepository.FollowArtist(user, artist)
}

func (r *RelationManager) FollowUser(userA, userB *model.User)  error {
	var err error

	//TODO this whole process should be a transaction

	err = r.RelationManagerRepository.FollowUser(userA, userB)

	if err != nil{
		return err
	}
	err = r.RelationManagerRepository.FollowUser(userB, userA)
	if err != nil{
		return err
	}
	return nil
}
func (r *RelationManager) IsFollowing(userA, userB *model.User) (bool, error) {


	return r.RelationManagerRepository.IsFollowingUser(userA, userB)
}

type RelationManagerRepository interface {
	GetFollowedArtist(user *model.User) []*model.Artist
	IsFollowingArtist(user *model.User, artist *model.Artist) bool
	FollowArtist(user *model.User, artist *model.Artist) error
	GetArtistBySpotifyID(spotifyID string) (*model.Artist, error)
	GetArtistByTwitterID(spotifyID string) *model.Artist
	AddArtist(data map[string]interface{}, spotifyID string) error
	IsFollowingUser(userA, userB *model.User) (bool, error)
	FollowUser(userA, userB *model.User) error
}

type UserSearchService interface {
	IndexUser(user interface{}) error
	SearchUsers(query string) ([]map[string]interface{}, error)
}
