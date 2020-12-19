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
	SpotifyAccessTokenRedirectUri = "http://com.example.yay/"
	SpotifyClientId               = "c32f7f7b46e14062ba2aea1b462415c9"
	SpotifyClientSecret           = "4bf8bb4cb9964ec8bb9d900bc9bc5fb3"
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
}

func NewAuthManager(authClient *auth.Client, httpClient http.Client, ctx context.Context, repository AuthManagerRepository) *AuthManager {
	newAuthManager := new(AuthManager)

	logger, _ := zap.NewDevelopment()

	logger.Sugar()

	newAuthManager.ctx = ctx
	newAuthManager.httpClient = httpClient
	newAuthManager.authClient = authClient
	newAuthManager.AuthManagerRepository = repository
	return newAuthManager

}

func (authenticator *AuthManager) AuthenticateUser(spotifyAccountData, twitterAccountData map[string]interface{}) (map[string]interface{}, error) {

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

	userIsAdded := authenticator.AddUser(*user)
	if userIsAdded != nil {
		return nil, errors.New("failed register user")
	}
	customToken, customTokenErr := authenticator.authClient.CustomToken(authenticator.ctx, user.GetUserUUID())

	if customTokenErr != nil {
		return nil, errors.New("failed to create custom token")

	}
	return map[string]interface{}{
		"status_code":  200,
		"custom_token": customToken,
	}, nil

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

	return map[string]interface{}{
		"status_code":      200,
		"access_token":     tokenReqRes["access_token"].(string),
		"refresh_token":    tokenReqRes["refresh_token"].(string),
		"expires_in":       tokenReqRes["expires_in"].(float64),
		"display_name":     userInfo["display_name"].(string),
		"token_time_stamp": tokenReqRes["token_time_stamp"],
		"user_email":       userInfo["email"].(string),
		//"picture": (userInfo["images"].(map[string]interface{}))["url"].(string),
		"profile": userInfo["href"].(string),
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

	lastFetchedTime := time.Unix(user.GetSpotifyAccount()["token_time_stamp"].(int64),0)

	elapsedTime := time.Now().Sub(lastFetchedTime)

	maxTime := time.Duration(float64(time.Minute.Nanoseconds() * 60)*0.9)
	if elapsedTime.Seconds() >= maxTime.Seconds() {
		refreshedAccessTokenMap, refreshedTokenErr := authenticator.RequestSpotifyRefreshedAccessToken(user.GetSpotifyAccount()["refresh_token"].(string))

		if refreshedTokenErr != nil {
			log.Fatal(refreshedTokenErr)
			return nil, refreshedTokenErr

		}

		log.Printf("token_time_stamp %v",refreshedAccessTokenMap["token_time_stamp"].(int64) )

		updateErr  := authenticator.UpdateSpotifyOauthInfo(*user, refreshedAccessTokenMap["access_token"].(string), refreshedAccessTokenMap["token_time_stamp"].(int64))

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
	accessTokenMap , accessTokenMapErr := authenticator.GetAccessTokenMap(user)
	if accessTokenMapErr != nil{
		return "", accessTokenMapErr
	}
	return accessTokenMap["access_token"].(string),nil
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

	oauthParams := url.Values{
		"oauth_consumer_key":     []string{helpers.TwitterSecretKey},
		"oauth_nonce":            []string{strconv.FormatInt(time.Now().Unix(), 10)},
		"oauth_version":          []string{"1.0"},
		"oauth_signature_method": []string{"HMAC-SHA1"},
		"oauth_timestamp":        []string{strconv.FormatInt(time.Now().Unix(), 10)},
	}

	signature, oauthHeader := helpers.OauthSignature("POST", twitterRequestTokenURL, helpers.TwitterSecretKey, "", tokenRequestParams, oauthParams)

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

type AuthManagerRepository interface {
	GetUserBySpotifyID(spotifyID string) *model.User
	GetUserByTwitterID(twitterID string) *model.User
	GetUserSpotifyAccessToken(twitterID string) string
	GetUserByUUID(uuid string) (*model.User, error)
	GetUserTwitterOauth(uuid string) (string, string, error)
	AddUser(user model.User) error
	UpdateSpotifyOauthInfo(user model.User,accessToken string, accessTokenTimeStamp int64)error
}
