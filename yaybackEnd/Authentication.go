package main

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
)

const EmailParam = "userEmail"
const RedirectUri = "http://com.example.yay/"
const ClientId = "c32f7f7b46e14062ba2aea1b462415c9"
const ClientSecret = "4bf8bb4cb9964ec8bb9d900bc9bc5fb3"

type  Authenticator struct{
	authClient *auth.Client
	db *db.Client
	router *mux.Router
	httpClient http.Client
}

type LoginBody struct {
	UserEmail string
	Code string
}

 func newAuthenticator(_authClient *auth.Client, _db *db.Client, router *mux.Router) *Authenticator {
	var authenticator = new(Authenticator)
	authenticator.authClient = _authClient
	authenticator.router = router
	authenticator.httpClient  = http.Client{}
	authenticator.db = _db
	authenticator.setRoutes()

	return authenticator

}

func(authenticator *Authenticator) setRoutes(){
	authenticator.router.HandleFunc("/login", authenticator.loginHandler).Methods("POST")
	authenticator.router.HandleFunc("/getFreshToken", authenticator.getFreshTokenHandler).Methods("GET")
}

func (authenticator *Authenticator) loginHandler(response http.ResponseWriter,req *http.Request){

	log.Printf("new login request from %v", req.RemoteAddr)

	var loginBody LoginBody
	rawBody , _ := ioutil.ReadAll(req.Body)

	json.Unmarshal(rawBody, &loginBody)

	tokenReqRes, tokenReqErr := authenticator.getToken(loginBody.Code)
	accessToken := tokenReqRes["access_token"].(string)

	if tokenReqErr != nil {
		log.Fatal(tokenReqErr)
	}


	userInfo, _ := authenticator.getUserInfo(accessToken)

	userEmail :=  userInfo["email"].(string)


	userRecord, _ := authenticator.authClient.GetUserByEmail(context.Background(),userEmail)

	if userRecord == nil {
		userRecord,_ = authenticator.createNewUser(userEmail)
	}



	userCustomToken , _ :=  authenticator.authClient.CustomToken(ctx, userRecord.UID)

	userRef := authenticator.db.NewRef("users")



	r := userRef.Child(userRecord.UID).Child("user_token").Set(context.Background(),map[string]interface{}{
		"access_token": tokenReqRes["access_token"],
		"refresh_token": tokenReqRes["refresh_token"],
		"timeStamp": time.Now().UTC().Unix(),
		"expires_in": tokenReqRes["expires_in"],
		"display_name": userInfo["display_name"],
		"picture": userInfo["images"],
		"profile": userInfo["href"],
	})

	print(r)

	loginResponse := map[string]string{
		"access_token" : tokenReqRes["access_token"].(string),
		"expires_in": strconv.Itoa(int(tokenReqRes["expires_in"].(float64))),
		"custom_token": userCustomToken,
		"display_name": userInfo["display_name"].(string),
		//"picture": (userInfo["images"].(map[string]interface{}))["url"].(string),
		"profile": userInfo["href"].(string),
	}
	loginResponseJSONText, _ := json.Marshal(loginResponse)

	response.Header().Add("Content-Type","application/json")
	response.Header().Add("Content-Length", strconv.Itoa(len(loginResponseJSONText)))

	response.WriteHeader(200)
	response.Write(loginResponseJSONText)

}
func (authenticator *Authenticator) getFreshTokenHandler(response http.ResponseWriter,req *http.Request){

	log.Printf("new fresh token request %v", req.RemoteAddr)

	userUUID := req.URL.Query().Get("userUUID")

	var userToken map[string]interface{}

	userTokenErr := authenticator.db.NewRef("users").Child(userUUID).Child("user_token").Get(context.Background(), &userToken)

	if userTokenErr != nil{
		log.Fatal(userTokenErr)
	}
	log.Printf("refresh token - user was found, UUID : %v", userToken)


	refreshToken := userToken["refresh_token"].(string)

	refreshTokenMap , _ := authenticator.getRefreshToken(refreshToken)



	refreshTokenByte, _ := json.Marshal(refreshTokenMap)


	log.Printf("sending refreshed token to client \n %v", len(refreshTokenByte))

	response.Write(refreshTokenByte)

}
func (authenticator *Authenticator) createNewUser(userEmail string) (*auth.UserRecord, error){

	var user = new (auth.UserToCreate)
	user.Email(userEmail).EmailVerified(true)
	return  authenticator.authClient.CreateUser(context.Background(),user)

}
func (authenticator *Authenticator) getToken(code string) (map[string]interface{} , error) {
	tokenReqURL := "https://accounts.spotify.com/api/token"
	print(code)
	tokenReqBody := url.Values{}
	tokenReqBody.Add("code", code)
	tokenReqBody.Add("grant_type", "authorization_code")
	tokenReqBody.Add("redirect_uri", RedirectUri)
	tokenReqBody.Add("client_id", ClientId)
	tokenReqBody.Add("client_secret", ClientSecret)

	tokenReq, _ := http.NewRequest("POST",tokenReqURL, strings.NewReader(tokenReqBody.Encode()))
	tokenReq.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	tokenReq.Header.Add("Content-Length", strconv.Itoa(len(tokenReqBody.Encode())))
	log.Printf("body :\n%v",tokenReqBody.Encode())

	tokenReqResponse, _ := authenticator.httpClient.Do(tokenReq)

	resBody, _ := ioutil.ReadAll(tokenReqResponse.Body)

	log.Printf("token response , status code %v,  body : %v", tokenReqResponse.StatusCode,string(resBody))
	if tokenReqResponse.StatusCode == 200{
		var tokenResponse interface{}
		json.Unmarshal(resBody,&tokenResponse)
		tokenResponseMap := tokenResponse.(map[string]interface{})
		return tokenResponseMap,nil
	}else{
		return nil, errors.New("failed")
	}


}
func (authenticator *Authenticator) getRefreshToken(refreshToken string) (map[string]interface{} , error) {
	tokenReqURL := "https://accounts.spotify.com/api/token"
	print(refreshToken)

	tokenReqBody := url.Values{}
	tokenReqBody.Add("refresh_token", refreshToken)
	tokenReqBody.Add("grant_type", "refresh_token")
	tokenReqBody.Add("client_id", ClientId)
	tokenReqBody.Add("client_secret", ClientSecret)



	tokenReq, _ := http.NewRequest("POST",tokenReqURL, strings.NewReader(tokenReqBody.Encode()))

	tokenReq.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	tokenReq.Header.Add("Content-Length", strconv.Itoa(len(tokenReqBody.Encode())))

	authorizationCode := ClientId + ":" + ClientSecret
	authorizationCodeB64 := "Basic " +base64.StdEncoding.EncodeToString([]byte(authorizationCode))
	tokenReq.Header.Add("Authorization", authorizationCodeB64)

	log.Printf("body :\n%v",tokenReqBody.Encode())

	tokenReqResponse, _ := authenticator.httpClient.Do(tokenReq)

	resBody, _ := ioutil.ReadAll(tokenReqResponse.Body)

	log.Printf("token response , status code %v,  body : %v", tokenReqResponse.StatusCode,string(resBody))
	if tokenReqResponse.StatusCode == 200{
		var tokenResponse interface{}
		json.Unmarshal(resBody,&tokenResponse)
		tokenResponseMap := tokenResponse.(map[string]interface{})
		return tokenResponseMap,nil
	}else{
		return nil, errors.New("failed")
	}

}
func (authenticator *Authenticator) getUserInfo(accessToken string) (map[string]interface{} , error){
	url := "https://api.spotify.com/v1/me"
	authorizationHeader := "Bearer "+accessToken

	userProfileReq, _ := http.NewRequest("GET", url,nil)
	userProfileReq.Header.Add("Authorization", authorizationHeader)

	userprofileRes, userProfileReqError := authenticator.httpClient.Do(userProfileReq)


	if userProfileReqError != nil || userprofileRes.StatusCode != 200{
		return nil, userProfileReqError
		log.Fatal("request failed")
	}

	userProfileReqByte , _ := ioutil.ReadAll(userprofileRes.Body)

	var userProfileMap interface{}

	json.Unmarshal(userProfileReqByte,&userProfileMap)
	userProfileMap = userProfileMap.(map[string]interface{})

	return userProfileMap.(map[string]interface{}),nil


}

