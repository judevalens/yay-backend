package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"firebase.google.com/go/auth"
	"go.uber.org/zap"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"yaybackEnd/helpers"
	"yaybackEnd/model"
)



const (
	TwitterOauthCallBack = "https://127.0.0.1/twitterCallback/"
)

type AuthManager struct {
	authClient *auth.Client
	httpClient http.Client
	ctx        context.Context
	logger     *zap.Logger
	AuthManagerRepository
	AuthSearchService
}

func NewAuthManager(authClient *auth.Client, httpClient http.Client, ctx context.Context, repository AuthManagerRepository, searchService AuthSearchService) *AuthManager {
	newAuthManager := new(AuthManager)

	logger, _ := zap.NewDevelopment()

	logger.Sugar()

	newAuthManager.ctx = ctx
	newAuthManager.httpClient = httpClient
	newAuthManager.authClient = authClient
	newAuthManager.AuthManagerRepository = repository
	newAuthManager.AuthSearchService = searchService
	return newAuthManager

}

func (authenticator *AuthManager) AuthenticateUser(spotifyAccountData, twitterAccountData map[string]interface{}) (map[string]interface{}, *model.User, error) {

	userEmail := spotifyAccountData["user_email"].(string)

	userRecord, userRecordErr := authenticator.authClient.GetUserByEmail(authenticator.ctx, userEmail)

	if userRecord == nil {
		userRecord, userRecordErr = authenticator.createNewUser(userEmail)
	}

	if userRecordErr != nil {
		// TODO need to handle error
		log.Fatal(userRecordErr)
	}

	user := model.NewUser(userRecord.UID, spotifyAccountData, twitterAccountData)

	log.Printf("spotify account data \n %v", spotifyAccountData)
	userIsAdded := authenticator.AddUser(*user)
	if userIsAdded != nil {
		return nil, nil, errors.New("failed register user")
	}
	_ = authenticator.IndexUser(map[string]string{
		"spotify_name":         user.SpotifyAccount["display_name"].(string),
		"twitter_name":         user.TwitterAccount["screen_name"].(string),
		"user_id":              user.GetUserUUID(),
		"objectID":             user.GetUserUUID(),
		"user_spotify_profile": user.SpotifyAccount["profile_picture"].(string),
	})
	customToken, customTokenErr := authenticator.authClient.CustomToken(authenticator.ctx, user.GetUserUUID())

	if customTokenErr != nil {
		return nil, nil, errors.New("failed to create custom token")

	}
	return map[string]interface{}{
		"status_code":  200,
		"custom_token": customToken,
	}, user, nil

}

func (authenticator *AuthManager) createNewUser(userSpotifyEmail string) (*auth.UserRecord, error) {
	var userToCreate = new(auth.UserToCreate)
	userToCreate.Email(userSpotifyEmail).EmailVerified(true)
	return authenticator.authClient.CreateUser(context.Background(), userToCreate)
}

func (authenticator *AuthManager) LoginWithSpotify(spotifyRecCode string) (map[string]interface{}, error) {
	tokenReqRes, tokenReqErr := authenticator.RequestSpotifyAccessToken(spotifyRecCode)
	accessToken := tokenReqRes["access_token"].(string)
	userInfo, _ := authenticator.RequestSpotifyUserInfo(accessToken)

	if tokenReqErr != nil {
		log.Printf("access token request err : %v", tokenReqErr)
		return nil, tokenReqErr
	}

	var userPictureUrl string

	if userInfo["images"] != nil {
		pictures := userInfo["images"].([]interface{})
		if len(pictures) > 0 {
			userPictureUrl = pictures[0].(map[string]interface{})["url"].(string)
		}
	}

	return map[string]interface{}{
		"status_code":      200,
		"access_token":     tokenReqRes["access_token"].(string),
		"refresh_token":    tokenReqRes["refresh_token"].(string),
		"expires_in":       tokenReqRes["expires_in"].(float64),
		"display_name":     userInfo["display_name"].(string),
		"token_time_stamp": tokenReqRes["token_time_stamp"],
		"user_email":       userInfo["email"].(string),
		//"picture": (userInfo["images"].(map[string]interface{}))["url"].(string),
		"profile":         userInfo["href"].(string),
		"profile_picture": userPictureUrl,
	}, nil
}

func (authenticator *AuthManager) RequestSpotifyAccessToken(code string) (map[string]interface{}, error) {
	tokenReqURL := "https://accounts.spotify.com/api/token"
	print(code)
	tokenReqBody := url.Values{}
	tokenReqBody.Add("code", code)
	tokenReqBody.Add("grant_type", "authorization_code")
	tokenReqBody.Add("redirect_uri", SpotifyAccessTokenRedirectUri)
	tokenReqBody.Add("client_id", SpotifyClientId)
	tokenReqBody.Add("client_secret", SpotifyClientSecret)

	tokenReq, _ := http.NewRequest("POST", tokenReqURL, strings.NewReader(tokenReqBody.Encode()))
	tokenReq.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	tokenReq.Header.Add("Content-Length", strconv.Itoa(len(tokenReqBody.Encode())))
	log.Printf("body :\n%v", tokenReqBody.Encode())

	tokenReqResponse, _ := authenticator.httpClient.Do(tokenReq)

	resBody, _ := ioutil.ReadAll(tokenReqResponse.Body)

	log.Printf("token response , status code %v,  body : %v", tokenReqResponse.StatusCode, string(resBody))
	if tokenReqResponse.StatusCode == 200 {
		var tokenResponse interface{}
		json.Unmarshal(resBody, &tokenResponse)
		tokenResponseMap := tokenResponse.(map[string]interface{})
		tokenResponseMap["token_time_stamp"] = time.Now().Unix()
		return tokenResponseMap, nil
	} else {
		return nil, errors.New("failed")
	}
}

func (authenticator *AuthManager) GetAccessTokenMap(user *model.User) (map[string]interface{}, error) {

	tokenTimeStamp, isInt := user.GetSpotifyAccount()["token_time_stamp"].(int64)

	if !isInt {
		tokenTimeStamp = int64(user.GetSpotifyAccount()["token_time_stamp"].(float64))
	}

	lastFetchedTime := time.Unix(tokenTimeStamp, 0)

	elapsedTime := time.Now().Sub(lastFetchedTime)

	maxTime := time.Duration(float64(time.Minute.Nanoseconds()*60) * 0.9)
	if elapsedTime.Seconds() >= maxTime.Seconds() {
		refreshedAccessTokenMap, refreshedTokenErr := authenticator.RequestSpotifyRefreshedAccessToken(user.GetSpotifyAccount()["refresh_token"].(string))

		if refreshedTokenErr != nil {
			log.Fatal(refreshedTokenErr)
			return nil, refreshedTokenErr

		}

		log.Printf("token_time_stamp %v", refreshedAccessTokenMap["token_time_stamp"].(int64))

		updateErr := authenticator.UpdateSpotifyOauthInfo(*user, refreshedAccessTokenMap["access_token"].(string), refreshedAccessTokenMap["token_time_stamp"].(int64))

		if updateErr != nil {
			log.Fatal(updateErr)
			return nil, refreshedTokenErr
		}

		return refreshedAccessTokenMap, nil
	} else {
		return user.GetSpotifyAccount(), nil
	}
}

func (authenticator *AuthManager) GetAccessToken(user *model.User) (string, error) {
	accessTokenMap, accessTokenMapErr := authenticator.GetAccessTokenMap(user)
	if accessTokenMapErr != nil {
		return "", accessTokenMapErr
	}
	return accessTokenMap["access_token"].(string), nil
}

func (authenticator *AuthManager) RequestSpotifyRefreshedAccessToken(refreshToken string) (map[string]interface{}, error) {

	tokenReqURL := "https://accounts.spotify.com/api/token"

	tokenReqBody := url.Values{}
	tokenReqBody.Add("refresh_token", refreshToken)
	tokenReqBody.Add("grant_type", "refresh_token")
	tokenReqBody.Add("client_id", SpotifyClientId)
	tokenReqBody.Add("client_secret", SpotifyClientSecret)

	tokenReq, _ := http.NewRequest("POST", tokenReqURL, strings.NewReader(tokenReqBody.Encode()))

	tokenReq.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	tokenReq.Header.Add("Content-Length", strconv.Itoa(len(tokenReqBody.Encode())))

	authorizationCode := SpotifyClientId + ":" + SpotifyClientSecret
	authorizationCodeB64 := "Basic " + base64.StdEncoding.EncodeToString([]byte(authorizationCode))
	tokenReq.Header.Add("Authorization", authorizationCodeB64)

	//log.Printf("body :\n%v", tokenReqBody.Encode())

	tokenReqResponse, _ := authenticator.httpClient.Do(tokenReq)

	resBody, _ := ioutil.ReadAll(tokenReqResponse.Body)

	log.Printf("token response , status code %v,  body : %v", tokenReqResponse.StatusCode, string(resBody))
	if tokenReqResponse.StatusCode == 200 {
		var tokenResponse interface{}
		_ = json.Unmarshal(resBody, &tokenResponse)
		tokenResponseMap := tokenResponse.(map[string]interface{})
		tokenResponseMap["token_time_stamp"] = time.Now().Unix()
		return tokenResponseMap, nil
	} else {
		return nil, errors.New("failed")
	}
}

func (authenticator *AuthManager) RequestSpotifyUserInfo(accessToken string) (map[string]interface{}, error) {
	spotifyUserProfileUrl := "https://api.spotify.com/v1/me"
	authorizationHeader := "Bearer " + accessToken

	userProfileReq, _ := http.NewRequest("GET", spotifyUserProfileUrl, nil)
	userProfileReq.Header.Add("Authorization", authorizationHeader)

	userprofileRes, userProfileReqError := authenticator.httpClient.Do(userProfileReq)

	if userProfileReqError != nil || userprofileRes.StatusCode != 200 {
		return nil, userProfileReqError
		log.Fatal("request failed")
	}

	userProfileReqByte, _ := ioutil.ReadAll(userprofileRes.Body)

	var userProfileMap interface{}

	json.Unmarshal(userProfileReqByte, &userProfileMap)
	userProfileMap = userProfileMap.(map[string]interface{})

	return userProfileMap.(map[string]interface{}), nil
}

func (authenticator *AuthManager) RequestTwitterRequestToken() (map[string]string, error) {

	twitterRequestTokenURL := "https://api.twitter.com/oauth/request_token"

	tokenRequestParams := url.Values{}

	tokenRequestParams.Add("oauth_callback", TwitterOauthCallBack)

	tokenRequest, _ := http.NewRequest("POST", twitterRequestTokenURL, strings.NewReader(tokenRequestParams.Encode()))

	tokenRequest.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	oauthParams := url.Values{}

	oauthParams.Add("oauth_consumer_key", TwitterApiKey)
	oauthParams.Add("oauth_nonce", strconv.FormatInt(time.Now().Unix(), 10))
	oauthParams.Add("oauth_version", "1.0")
	oauthParams.Add("oauth_signature_method", "HMAC-SHA1")
	oauthParams.Add("oauth_timestamp", strconv.FormatInt(time.Now().Unix(), 10))

	signature, oauthHeader := helpers.OauthSignature("POST", twitterRequestTokenURL, TwitterSecretKey, "", tokenRequestParams, oauthParams)

	tokenRequest.Header.Add("Authorization", oauthHeader)
	requestedToken, _ := authenticator.httpClient.Do(tokenRequest)

	log.Printf("signature : %s ,\nheader : %s", signature, oauthHeader)

	requestedTokenRes, _ := ioutil.ReadAll(requestedToken.Body)

	log.Printf("%v", requestedToken.StatusCode)
	log.Printf("%v", requestedToken.Status)
	log.Printf("%v byte read ; %v", string(requestedTokenRes), requestedToken.ContentLength)

	if requestedToken.StatusCode == 200 {
		parsedResponse, _ := url.ParseQuery(string(requestedTokenRes))

		return map[string]string{
			"status_code":              strconv.Itoa(requestedToken.StatusCode),
			"oauth_token":              parsedResponse.Get("oauth_token"),
			"oauth_token_secret":       parsedResponse.Get("oauth_token_secret"),
			"oauth_callback_confirmed": parsedResponse.Get("oauth_callback_confirmed"),
		}, nil

	} else {
		return map[string]string{
			"error":       "could not get request token",
			"status_code": strconv.Itoa(requestedToken.StatusCode),
		}, errors.New("could not get request token")
	}
}

// TODO need to figure out which params we need
func (authenticator *AuthManager) RequestTwitterAccessToken(oauthToken, oauthVerifier string) (map[string]interface{}, error) {
	log.Printf("getting access token")

	params := url.Values{}

	params.Add("oauth_token", oauthToken)
	params.Add("oauth_verifier", oauthVerifier)

	twitterAccessTokenURL := "https://api.twitter.com/oauth/access_token" + "?" + params.Encode()
	tokenRequest, _ := http.NewRequest("POST", twitterAccessTokenURL, nil)

	tokenRequest.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	requestedAccessToken, _ := authenticator.httpClient.Do(tokenRequest)
	log.Printf("status code for accessToken : %v", requestedAccessToken.StatusCode)

	if requestedAccessToken.StatusCode == 200 {
		requestedAccessTokenRes, _ := ioutil.ReadAll(requestedAccessToken.Body)

		parsedResponse, _ := url.ParseQuery(string(requestedAccessTokenRes))

		return map[string]interface{}{
			"status_code":        requestedAccessToken.StatusCode,
			"oauth_token":        parsedResponse.Get("oauth_token"),
			"oauth_token_secret": parsedResponse.Get("oauth_token_secret"),
			"user_id":            parsedResponse.Get("user_id"),
			"screen_name":        parsedResponse.Get("screen_name"),
		}, nil
	} else {
		return map[string]interface{}{
			"error":       "could not get access token",
			"status_code": requestedAccessToken.StatusCode,
		}, errors.New("could not get access token")
	}

}

func (authenticator *AuthManager) GetTwitterAccessToken(uuid string) (string, string, error) {
	return authenticator.GetUserTwitterOauth(uuid)
}

// SHOULD probably move the subsequent function into another file
func (authenticator *AuthManager) requestUserTop(user *model.User, topType string) (map[string]interface{}, error) {
	topUrl := "https://api.spotify.com/v1/me/top/" + topType

	topReq, _ := http.NewRequest("GET", topUrl, nil)

	params := url.Values{}
	params.Add("limit", "20")

	topReq.URL.RawQuery = params.Encode()

	accessToken, accessTokenErr := authenticator.GetAccessToken(user)

	if accessTokenErr != nil {
	}

	topReq.Header.Set("Authorization", "Bearer "+accessToken)

	topReqRes, _ := authenticator.httpClient.Do(topReq)

	topResByte, topResErr := ioutil.ReadAll(topReqRes.Body)

	if topResErr != nil {

	}
	var topRes map[string]interface{}

	topUnmarshalErr := json.Unmarshal(topResByte, &topRes)

	if topResErr != nil {
		log.Print(topUnmarshalErr)
	}

	log.Printf("status code  : %v, url : %v, user top, \n %v", topReqRes.Status, topReq.URL.String(), string(topResByte))

	return topRes, nil
}

func (authenticator *AuthManager) UpdateUserProfile(user *model.User) {
	var err error

	var userTops = make(map[string]interface{})
	userTops["tracks"], err = authenticator.requestUserTop(user, "tracks")
	userTops["artists"], err = authenticator.requestUserTop(user, "artists")

	if err != nil {
	}

	log.Printf("user tops, \n %v", userTops)

	_ = authenticator.UpdateUserTops(user, userTops)

}

func (authenticator *AuthManager) getUserProfile(user *model.User) (map[string]interface{}, error) {
	var err error

	userProfile, err := authenticator.GetUserProfile(user)

	if err != nil {

	}

	return userProfile, nil
}

type AuthManagerRepository interface {
	GetUserBySpotifyID(spotifyID string) *model.User
	GetUserByTwitterID(twitterID string) *model.User
	GetUserSpotifyAccessToken(twitterID string) string
	GetUserByUUID(uuid string) (*model.User, error)
	GetUserTwitterOauth(uuid string) (string, string, error)
	AddUser(user model.User) error
	UpdateSpotifyOauthInfo(user model.User, accessToken string, accessTokenTimeStamp int64) error
	UpdateUserTops(user *model.User, userTops map[string]interface{}) error
	GetUserProfile(user *model.User) (map[string]interface{}, error)
}

type AuthSearchService interface {
	IndexUser(user interface{}) error
	SearchUsers(query string) ([]map[string]interface{}, error)
}
