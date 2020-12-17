package auth

import (
	"cloud.google.com/go/firestore"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"firebase.google.com/go/auth"
	"firebase.google.com/go/db"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const EmailParam = "userEmail"
const RedirectUri = "http://com.example.yay/"
const SpotifyClientId = "c32f7f7b46e14062ba2aea1b462415c9"
const SpotifyClientSecret = "4bf8bb4cb9964ec8bb9d900bc9bc5fb3"

const TwitterApiKey = "9OTPANWh0sX2TmBqSz3IG1WQa"
const TwitterSecretKey = "6Jil7rakvEFzhslXxxCuM8Szf8ZC6qUQmNK6dbMSjCNNK6qwuD"
const TwitterBearerToken = "AAAAAAAAAAAAAAAAAAAAANPdKQEAAAAAXkg3AHAmFBW8OpZlV1LNy5nDHbg%3D7Spe4lgo4cFqabdV12XHKPn28H4k9esKo9znQkXDeZoNSBOGWz"
const TwitterAccessToken = "2919369773-YRHJGm5S5IqXqN6S61k9jU5f5oIqDXazARzEYFo"
const TwitterAccessSecret = "B7P71NG4Bk5oVfVlYL5wpU8VShwRDoddYEKGz1FX3KReE"

type Authenticator struct {
	authClient *auth.Client
	db         *db.Client
	fireStoreDB *firestore.Client
	router     *mux.Router
	httpClient http.Client
	ctx        context.Context
}

type LoginBody struct {
	UserEmail string
	Code      string
}

func NewAuthenticator(_authClient *auth.Client, _db *db.Client,_fireStoreDB *firestore.Client, _ctx context.Context, router *mux.Router) *Authenticator {
	var authenticator = new(Authenticator)
	authenticator.authClient = _authClient
	authenticator.router = router
	authenticator.httpClient = http.Client{}
	authenticator.db = _db
	authenticator.fireStoreDB = _fireStoreDB
	authenticator.ctx = _ctx
	authenticator.setRoutes()

	return authenticator

}

func (authenticator *Authenticator) setRoutes() {
	authenticator.router.HandleFunc("/spotifyLogin", authenticator.spotifyLoginHandler).Methods("POST")
	authenticator.router.HandleFunc("/spotifyGetFreshToken", authenticator.GetSpotifyRefreshAccessTokenHandler).Methods("GET")

	authenticator.router.HandleFunc("/getTwitterRequestToken", authenticator.geTwitterRequestToken).Methods("GET")

	authenticator.router.HandleFunc("/getTwitterAccessToken", authenticator.getTwitterAccessTokenHandler).Methods("GET")

	authenticator.router.HandleFunc("/login", authenticator.Login).Methods("POST")
}

func (authenticator *Authenticator)Login(response http.ResponseWriter, req *http.Request){
	var loginData map[string]interface{}
	loginDataBytes, _ := ioutil.ReadAll(req.Body)
	parsingError := json.Unmarshal(loginDataBytes, &loginData)
	if parsingError != nil {
		log.Printf("error : %v",parsingError.Error())
	}
	log.Printf("%v", loginData)

	spotifyLoginData := loginData["spotifyLoginData"].(map[string]interface{})
	twitterLoginData := loginData["twitterLoginData"].(map[string]interface{})

		_ = twitterLoginData


		log.Printf("trying to log in : \n%v",loginData)


	userEmail := spotifyLoginData["user_email"].(string)
	userRecord, _ := authenticator.authClient.GetUserByEmail(context.Background(), userEmail)
	if userRecord == nil {
		userRecord, _ = authenticator.createNewUser(userEmail)
	}

	_, writeError := authenticator.fireStoreDB.Collection("users").Doc(userRecord.UID).Set(context.Background(), map[string]interface{}{
		"spotify_account": map[string]interface{}{
			"access_token": spotifyLoginData["access_token"],
			"refresh_token": spotifyLoginData["refresh_token"],
			"token_time_stamp": int64(spotifyLoginData["token_time_stamp"].(float64)),
		},
		"twitter_account": twitterLoginData,
	}, firestore.MergeAll)

	if writeError != nil{
		log.Fatal(writeError)
	}

	customToken, _ := authenticator.authClient.CustomToken(authenticator.ctx, userRecord.UID)

	loginResponse, _ := json.Marshal(map[string]interface{}{
		"status_code": 200,
		"custom_token": customToken,
	})

	response.Write(loginResponse)
}

func (authenticator *Authenticator) spotifyLoginHandler(response http.ResponseWriter, req *http.Request) {

	log.Printf("new login request from %v", req.RemoteAddr)

	var loginBody LoginBody
	rawBody, _ := ioutil.ReadAll(req.Body)

	json.Unmarshal(rawBody, &loginBody)

	tokenReqRes, tokenReqErr := authenticator.getSpotifyAccessToken(loginBody.Code)
	accessToken := tokenReqRes["access_token"].(string)
	userInfo, _ := authenticator.getSpotifyUserInfo(accessToken)

	if tokenReqErr != nil {
		log.Fatal(tokenReqErr)
	}

	loginResponse := map[string]interface{}{
		"status_code": 200,
		"access_token": tokenReqRes["access_token"].(string),
		"refresh_token": tokenReqRes["refresh_token"].(string),
		"expires_in":   tokenReqRes["expires_in"].(float64),
		"display_name": userInfo["display_name"].(string),
		"token_time_stamp": tokenReqRes["token_time_stamp"],
		"user_email": userInfo["email"].(string),
		//"picture": (userInfo["images"].(map[string]interface{}))["url"].(string),
		"profile": userInfo["href"].(string),
	}
	loginResponseJSONText, _ := json.Marshal(loginResponse)

	response.Header().Add("Content-Type", "application/json")
	response.Header().Add("Content-Length", strconv.Itoa(len(loginResponseJSONText)))

	response.WriteHeader(200)
	response.Write(loginResponseJSONText)

}

func (authenticator *Authenticator) GetSpotifyRefreshAccessTokenHandler(response http.ResponseWriter, req *http.Request) {

	log.Printf("new fresh token request %v", req.RemoteAddr)

	userUUID := req.URL.Query().Get("userUUID")


	_, refreshedAccessTokenRes := authenticator.GetSpotifyRefreshedAccessToken(userUUID)


	refreshTokenByte, _ := json.Marshal(refreshedAccessTokenRes)


	log.Printf("sending refreshed token to client \n %v", refreshedAccessTokenRes)

	response.Write(refreshTokenByte)

}

func (authenticator *Authenticator) createNewUser(userEmail string) (*auth.UserRecord, error) {

	var user = new(auth.UserToCreate)
	user.Email(userEmail).EmailVerified(true)
	return authenticator.authClient.CreateUser(context.Background(), user)

}
func (authenticator *Authenticator) getSpotifyAccessToken(code string) (map[string]interface{}, error) {
	tokenReqURL := "https://accounts.spotify.com/api/token"
	print(code)
	tokenReqBody := url.Values{}
	tokenReqBody.Add("code", code)
	tokenReqBody.Add("grant_type", "authorization_code")
	tokenReqBody.Add("redirect_uri", RedirectUri)
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


func (authenticator *Authenticator) refreshAccessToken(refreshToken string) (map[string]interface{}, error) {
	tokenReqURL := "https://accounts.spotify.com/api/token"
	print(refreshToken)

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
func(authenticator *Authenticator) GetSpotifyRefreshedAccessToken(uuid string) (string,map[string]interface{}) {
	userDoc := authenticator.fireStoreDB.Collection("users").Doc(uuid)
	userDocSnapShot, _ := userDoc.Get(authenticator.ctx)
	userDocData := userDocSnapShot.Data()
	spotifyAccountData := userDocData["spotify_account"].(map[string]interface{})

	refreshToken := spotifyAccountData["refresh_token"].(string)

	tokenTimestamp := spotifyAccountData["token_time_stamp"].(int64)

	lastFetched := time.Unix(tokenTimestamp/100,0)

	var refreshedToken map[string]interface{}

	if (time.Now().Unix() - lastFetched.Unix()) > (55*60) {
		refreshedAccessTokenMap, refreshedTokenErr := authenticator.refreshAccessToken(refreshToken)

		if refreshedTokenErr != nil{
			log.Fatal(refreshedTokenErr)
		}

		userDoc.Update(authenticator.ctx,[]firestore.Update{
			{
				Path:  "spotify_account.access_token",
				Value: refreshedAccessTokenMap["access_token"].(string),
			},
			{
				Path:  "spotify_account.token_time_stamp",
				Value: refreshedAccessTokenMap["token_time_stamp"].(int64),
			},
		})

		return refreshedAccessTokenMap["access_token"].(string), refreshedAccessTokenMap

	}

	return spotifyAccountData["access_token"].(string),refreshedToken
}


func (authenticator *Authenticator) getSpotifyUserInfo(accessToken string) (map[string]interface{}, error) {
	url := "https://api.spotify.com/v1/me"
	authorizationHeader := "Bearer " + accessToken

	userProfileReq, _ := http.NewRequest("GET", url, nil)
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


func (authenticator *Authenticator) geTwitterRequestToken(response http.ResponseWriter, req *http.Request) {

	var reqResponse []byte

	twitterRequestTokenURL := "https://api.twitter.com/oauth/request_token"
	twitterOauthCallBackURL := "https://127.0.0.1/twitterCallback/"

	tokenRequestParams := url.Values{}

	tokenRequestParams.Add("oauth_callback", twitterOauthCallBackURL)

	tokenRequest, _ := http.NewRequest("POST", twitterRequestTokenURL, strings.NewReader(tokenRequestParams.Encode()))

	tokenRequest.Header.Add("Content-Type", "application/x-www-form-urlencoded")


	oauthParams := url.Values{
		"oauth_consumer_key":     []string{TwitterApiKey},
		"oauth_nonce":             []string{strconv.FormatInt(time.Now().Unix(),10)},
		"oauth_version":           []string{"1.0"},
		"oauth_signature_method":  []string{"HMAC-SHA1"},
		"oauth_timestamp":         []string{strconv.FormatInt(time.Now().Unix(),10)},
	}

	signature,oauthHeader := OauthSignature("POST",twitterRequestTokenURL,TwitterSecretKey,"",tokenRequestParams,oauthParams)

	tokenRequest.Header.Add("Authorization", oauthHeader)
	requestedToken, _ := authenticator.httpClient.Do(tokenRequest)


	log.Printf("signature : %s ,\nheader : %s",signature,oauthHeader)

	requestedTokenRes, _ := ioutil.ReadAll(requestedToken.Body)


	log.Printf("%v", requestedToken.StatusCode)
	log.Printf("%v", requestedToken.Status)
	log.Printf("%v byte read ; %v",string(requestedTokenRes), requestedToken.ContentLength)

	if requestedToken.StatusCode == 200 {
		parsedResponse, _ := url.ParseQuery(string(requestedTokenRes))

		reqResponse, _ = json.Marshal(map[string]string{
			"status_code": strconv.Itoa(requestedToken.StatusCode),
			"oauth_token": parsedResponse.Get("oauth_token"),
			"oauth_token_secret": parsedResponse.Get("oauth_token_secret"),
			"oauth_callback_confirmed": parsedResponse.Get("oauth_callback_confirmed"),
		})

	}else{
		reqResponse, _ = json.Marshal(map[string]string{
			"error": "could not get request token",
			"status_code": strconv.Itoa(requestedToken.StatusCode),
		})
	}

	response.Write(reqResponse)

}

//TODO : must checks if params exist
func (authenticator *Authenticator) getTwitterAccessTokenHandler(response http.ResponseWriter, req *http.Request){
	log.Printf("getting access token")
	var reqResponse []byte

	req.ParseForm()
	twitterAccessTokenURL := "https://api.twitter.com/oauth/access_token"
	tokenRequest, _ := http.NewRequest("POST", twitterAccessTokenURL, strings.NewReader(req.Form.Encode()))



	tokenRequest.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	requestedAccessToken, _ := authenticator.httpClient.Do(tokenRequest)
	log.Printf("status code for accessToken : %v",requestedAccessToken.StatusCode)

	if requestedAccessToken.StatusCode == 200 {
		requestedAccessTokenRes, _ := ioutil.ReadAll(requestedAccessToken.Body)

		parsedResponse, _ := url.ParseQuery(string(requestedAccessTokenRes))


		reqResponse, _ = json.Marshal(map[string]interface{}{
			"status_code": requestedAccessToken.StatusCode,
			"oauth_token": parsedResponse.Get("oauth_token"),
			"oauth_token_secret": parsedResponse.Get("oauth_token_secret"),
			"user_id": parsedResponse.Get("user_id"),
			"screen_name": parsedResponse.Get("screen_name"),
		})

	}else{
		reqResponse, _ = json.Marshal(map[string]interface{}{
			"error": "could not get access token",
			"status_code": requestedAccessToken.StatusCode,
		})
	}


	response.Write(reqResponse)

}

// TODO : must check that the token exist
func (authenticator *Authenticator) GetTwitterAccessToken(uuid string) (string,string) {
	userDoc := authenticator.fireStoreDB.Collection("users").Doc(uuid)
	userDocSnapShot, _ := userDoc.Get(authenticator.ctx)
	userDocData := userDocSnapShot.Data()
	spotifyAccountData := userDocData["twitter_account"].(map[string]interface{})
	return spotifyAccountData["oauth_token"].(string),spotifyAccountData["oauth_token_secret"].(string)
}
