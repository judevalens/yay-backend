package auth

import (
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
	"yaybackEnd/misc"
)

const EmailParam = "userEmail"
const RedirectUri = "http://com.example.yay/"
const SpotifyClientId = "c32f7f7b46e14062ba2aea1b462415c9"
const SpotifyClientSecret = "4bf8bb4cb9964ec8bb9d900bc9bc5fb3"

const TwitterApiKey = "9OTPANWh0sX2TmBqSz3IG1WQa"
const TwitterSecretKey = "16bwPwW05wwar36QlivkiEwSEePvbCurMxwPoC5faRXSS4YnRb"
const TwitterBearerToken = "AAAAAAAAAAAAAAAAAAAAANPdKQEAAAAAXkg3AHAmFBW8OpZlV1LNy5nDHbg%3D7Spe4lgo4cFqabdV12XHKPn28H4k9esKo9znQkXDeZoNSBOGWz"
const TwitterAccessToken = "2919369773-YRHJGm5S5IqXqN6S61k9jU5f5oIqDXazARzEYFo"
const TwitterAccessSecret = "B7P71NG4Bk5oVfVlYL5wpU8VShwRDoddYEKGz1FX3KReE"

type Authenticator struct {
	authClient *auth.Client
	db         *db.Client
	router     *mux.Router
	httpClient http.Client
	ctx        context.Context
}

type LoginBody struct {
	UserEmail string
	Code      string
}

func NewAuthenticator(_authClient *auth.Client, _db *db.Client, router *mux.Router) *Authenticator {
	var authenticator = new(Authenticator)
	authenticator.authClient = _authClient
	authenticator.router = router
	authenticator.httpClient = http.Client{}
	authenticator.db = _db
	authenticator.setRoutes()

	return authenticator

}

func (authenticator *Authenticator) setRoutes() {
	authenticator.router.HandleFunc("/login", authenticator.loginHandler).Methods("POST")

	authenticator.router.HandleFunc("/getFreshToken", authenticator.getFreshTokenHandler).Methods("GET")

	authenticator.router.HandleFunc("/getTwitterRequestToken", authenticator.geTwitterRequestToken).Methods("GET")
}

func (authenticator *Authenticator) loginHandler(response http.ResponseWriter, req *http.Request) {

	log.Printf("new login request from %v", req.RemoteAddr)

	var loginBody LoginBody
	rawBody, _ := ioutil.ReadAll(req.Body)

	json.Unmarshal(rawBody, &loginBody)

	tokenReqRes, tokenReqErr := authenticator.getToken(loginBody.Code)
	accessToken := tokenReqRes["access_token"].(string)

	if tokenReqErr != nil {
		log.Fatal(tokenReqErr)
	}

	userInfo, _ := authenticator.getUserInfo(accessToken)

	userEmail := userInfo["email"].(string)

	userRecord, _ := authenticator.authClient.GetUserByEmail(context.Background(), userEmail)

	if userRecord == nil {
		userRecord, _ = authenticator.createNewUser(userEmail)
	}

	userCustomToken, _ := authenticator.authClient.CustomToken(authenticator.ctx, userRecord.UID)

	userRef := authenticator.db.NewRef("users")

	r := userRef.Child(userRecord.UID).Child("user_token").Set(context.Background(), map[string]interface{}{
		"access_token":  tokenReqRes["access_token"],
		"refresh_token": tokenReqRes["refresh_token"],
		"timeStamp":     time.Now().UTC().Unix(),
		"expires_in":    tokenReqRes["expires_in"],
		"display_name":  userInfo["display_name"],
		"picture":       userInfo["images"],
		"profile":       userInfo["href"],
	})

	print(r)

	loginResponse := map[string]string{
		"access_token": tokenReqRes["access_token"].(string),
		"expires_in":   strconv.Itoa(int(tokenReqRes["expires_in"].(float64))),
		"custom_token": userCustomToken,
		"display_name": userInfo["display_name"].(string),
		//"picture": (userInfo["images"].(map[string]interface{}))["url"].(string),
		"profile": userInfo["href"].(string),
	}
	loginResponseJSONText, _ := json.Marshal(loginResponse)

	response.Header().Add("Content-Type", "application/json")
	response.Header().Add("Content-Length", strconv.Itoa(len(loginResponseJSONText)))

	response.WriteHeader(200)
	response.Write(loginResponseJSONText)

}

func (authenticator *Authenticator) getFreshTokenHandler(response http.ResponseWriter, req *http.Request) {

	log.Printf("new fresh token request %v", req.RemoteAddr)

	userUUID := req.URL.Query().Get("userUUID")

	var userToken map[string]interface{}

	userTokenErr := authenticator.db.NewRef("users").Child(userUUID).Child("user_token").Get(context.Background(), &userToken)

	if userTokenErr != nil {
		log.Fatal(userTokenErr)
	}
	log.Printf("refresh token - user was found, UUID : %v", userToken)

	refreshToken := userToken["refresh_token"].(string)

	refreshTokenMap, _ := authenticator.getRefreshToken(refreshToken)

	refreshTokenByte, _ := json.Marshal(refreshTokenMap)

	log.Printf("sending refreshed token to client \n %v", len(refreshTokenByte))

	response.Write(refreshTokenByte)

}
func (authenticator *Authenticator) createNewUser(userEmail string) (*auth.UserRecord, error) {

	var user = new(auth.UserToCreate)
	user.Email(userEmail).EmailVerified(true)
	return authenticator.authClient.CreateUser(context.Background(), user)

}
func (authenticator *Authenticator) getToken(code string) (map[string]interface{}, error) {
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
		return tokenResponseMap, nil
	} else {
		return nil, errors.New("failed")
	}

}
func (authenticator *Authenticator) getRefreshToken(refreshToken string) (map[string]interface{}, error) {
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
		return tokenResponseMap, nil
	} else {
		return nil, errors.New("failed")
	}

}
func (authenticator *Authenticator) getUserInfo(accessToken string) (map[string]interface{}, error) {
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

	twitterRequestToken := "https://api.twitter.com/oauth/request_token"
	twitterOauthCallBack := "https://127.0.0.1/twitterCallback/"

	log.Printf("%s", twitterOauthHeader())

	tokenRequestParam := url.Values{}
	tokenRequestParam.Add("oauth_callback", twitterOauthCallBack)

	log.Printf("PARAMS 1: %s", url.QueryEscape("https://127.0.0.1/twitterCallback/"))
	log.Printf("PARAMS : %s", tokenRequestParam.Encode())

	tokenRequest, _ := http.NewRequest("POST", twitterRequestToken, strings.NewReader(tokenRequestParam.Encode()))
	tokenRequest.Header.Add("Authorization", twitterOauthHeader())
	tokenRequest.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	requestedToken, _ := authenticator.httpClient.Do(tokenRequest)

	params := map[string]string{
		"include_entities": "true",
		"status":           "Hello Ladies + Gentlemen, a signed OAuth request!",
	}

	oauthParams := map[string]string{
		"oauth_consumer_key":     "xvz1evFS4wEEPTGEFPHBog",
		"oauth_nonce":            "kYjzVBB8Y0ZFabxSWbWovY3uYSQ2pTgmZeNu2VS4cg", //strconv.FormatInt(time.Now().Unix(),10),
		"oauth_version":          "1.0",
		"oauth_signature_method": "HMAC-SHA1",
		"oauth_token":            "370773112-GmHxMAgYyLbNEtIKZeRNFsMKPR9EyMZeS9weJAEb",
		"oauth_timestamp":        "1318622958",
	}

	signature , oauthHeader:= misc.OauthSignature("POST", "https://api.twitter.com/1.1/statuses/update.json", "kAcSOqF21Fu85e7zjz7ZN2U4ZRhfV3WpwPAoE3Z7kBw", "LswwdoUaIvS8ltyTt5jkRh4J50vUPVVHtR2YPi5kE",
		params, oauthParams)


	log.Printf("signature : %s ,\nheader : %s",signature,oauthHeader)


	log.Printf("%v", requestedToken.StatusCode)
	log.Printf("%v", requestedToken.Status)

	response.Write([]byte("hello"))

}

func twitterOauthHeader() string {
	return "OAuth oauth_consumer_key=\"9OTPANWh0sX2TmBqSz3IG1WQa\",oauth_signature_method=\"HMAC-SHA1\",oauth_timestamp=\"1607611378\",oauth_nonce=\"TZ04lL\",oauth_version=\"1.0\",oauth_signature=\"HAD6ktd%2BU%2FkXBqYzmzdhWV9pJFg%3D\""
}
