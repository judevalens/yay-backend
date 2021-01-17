package api

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"yaybackEnd/app"
)

type AuthApi struct {
	router *mux.Router
	authManager *app.AuthManager

}

func NewAuthApi(router *mux.Router,manager *app.AuthManager) *AuthApi{
	newAuthApi := new(AuthApi)
	s, _ := router.Path("/auth").GetPathTemplate()
	log.Printf("route name : %v",s )
	newAuthApi.router = router.PathPrefix("/auth").Subrouter()
	newAuthApi.authManager = manager
	newAuthApi.setRoutes()
	return newAuthApi
}
func(authApi *AuthApi) setRoutes(){
	authApi.router.HandleFunc("/spotifyLogin", authApi.spotifyLoginHandler).Methods("POST")
	authApi.router.HandleFunc("/spotifyGetFreshToken", authApi.refreshSpotifyToken).Methods("GET")

	authApi.router.HandleFunc("/getTwitterRequestToken", authApi.geTwitterRequestToken).Methods("GET")

	authApi.router.HandleFunc("/getTwitterAccessToken", authApi.getTwitterAccessToken).Methods("GET")

	authApi.router.HandleFunc("/getUserProfile", authApi.getUserProfile).Methods("GET")

	r := authApi.router.HandleFunc("/login", authApi.Login).Methods("POST")

	path , _ := r.GetPathTemplate()

	log.Printf("route : %v", path)
}


func (authApi *AuthApi) Login(res http.ResponseWriter,req *http.Request){

	var loginData map[string]interface{}
	loginDataBytes, _ := ioutil.ReadAll(req.Body)
	parsingError := json.Unmarshal(loginDataBytes, &loginData)
	if parsingError != nil {
		log.Printf("error : %v",parsingError.Error())
	}
	log.Printf("%v", loginData)

	spotifyLoginData := loginData["spotifyLoginData"].(map[string]interface{})
	twitterLoginData := loginData["twitterLoginData"].(map[string]interface{})

	loginAnswer, newUser, loginAnswerErr := authApi.authManager.AuthenticateUser(spotifyLoginData, twitterLoginData)
	if loginAnswerErr != nil{
		// TODO need to handle error
		log.Print(loginAnswerErr)

		resByte, _ := json.Marshal(map[string]interface{}{
				"status": 400,
				"error": loginAnswerErr,
		})

		_, _ = res.Write(resByte)
		return
	}

	go func() {
		authApi.authManager.UpdateUserProfile(newUser)
	}()

	loginAnswerByte, loginAnswerMarshalErr := json.Marshal(loginAnswer)

	if loginAnswerMarshalErr != nil{
		// TODO need to handle error
		log.Fatal(loginAnswerMarshalErr)
	}

	_, _ = res.Write(loginAnswerByte)
}

func (authApi *AuthApi) spotifyLoginHandler(res http.ResponseWriter,req *http.Request)  {
	log.Printf("new spotify login request from %v", req.RemoteAddr)
	var loginBody interface{}
	rawBody, _ := ioutil.ReadAll(req.Body)
	json.Unmarshal(rawBody, &loginBody)


	spotifyAccessCode := loginBody.(map[string]interface{})["access_code"].(string)
	loginAnswer , spotifyLoginErr := authApi.authManager.LoginWithSpotify(spotifyAccessCode)

	loginAnswerByte, loginMarshalErr := json.Marshal(loginAnswer)

	if spotifyLoginErr != nil{
		log.Fatal(spotifyLoginErr)
	}
	if loginMarshalErr != nil{
		log.Fatal(spotifyLoginErr)
	}

	_, _ = res.Write(loginAnswerByte)
}

func (authApi *AuthApi) refreshSpotifyToken(res http.ResponseWriter,req *http.Request){
	// TODO should create a separate handle for soft login and refreshing token
	log.Printf("new fresh token request %v", req.RemoteAddr)

	userUUID := req.URL.Query().Get("user_uuid")

	user, userErr := authApi.authManager.GetUserByUUID(userUUID)

	if userErr != nil{
		log.Fatal(userErr)
	}

	refreshedAccessTokenRes, refreshedAccessTokenResErr := authApi.authManager.GetAccessTokenMap(user)

	if refreshedAccessTokenResErr != nil{
		// TODO must handle error
		log.Fatal(refreshedAccessTokenResErr)
	}

	// Whenever the user
	go func() {
		authApi.authManager.UpdateUserProfile(user)
	}()

	log.Printf("result %v",refreshedAccessTokenRes)
	refreshTokenByte, refreshTokenMarshalErr := json.Marshal(refreshedAccessTokenRes)

	if refreshTokenMarshalErr != nil{
		log.Fatal(refreshTokenMarshalErr)
	}

	log.Printf("sending refreshed token to client \n %v", refreshedAccessTokenRes)

	_, _ = res.Write(refreshTokenByte)
}

func (authApi *AuthApi) geTwitterRequestToken(res http.ResponseWriter,req *http.Request){
	requestTokenAnswer, requestTokenAnswerErr := authApi.authManager.RequestTwitterRequestToken()

	if requestTokenAnswerErr != nil{
		//TODO must handle error properly
		log.Fatal()
	}

	requestTokenAnswerByte, requestTokenAnswerMarshallErr := json.Marshal(requestTokenAnswer)

	if requestTokenAnswerMarshallErr != nil{
		//TODO must handle error gracefully
		log.Fatal(requestTokenAnswerMarshallErr)
	}

	res.Write(requestTokenAnswerByte)
}

func (authApi *AuthApi) getTwitterAccessToken(res http.ResponseWriter,req *http.Request){
	req.ParseForm()

	oauthToken := req.Form.Get("oauth_token")
	oauthVerifier := req.Form.Get("oauth_verifier")
	accessTokenAnswer, accessTokenAnswerErr := authApi.authManager.RequestTwitterAccessToken(oauthToken, oauthVerifier)

	if accessTokenAnswerErr != nil{
		//TODO must handle error gracefully
		log.Fatal(accessTokenAnswerErr)
	}

	accessTokenAnswerByte , accessTokenAnswerMarshalErr := json.Marshal(accessTokenAnswer)

	if accessTokenAnswerMarshalErr != nil {
		//TODO must handle error gracefully
		log.Fatal(accessTokenAnswerMarshalErr)
	}

	res.Write(accessTokenAnswerByte)
}

func (authApi *AuthApi) getUserProfile(res http.ResponseWriter,req *http.Request){
	var err error
	var userProfile map[string]interface{}

	_ = req.ParseForm()

	log.Print("retrieving user profile")

	userID := req.Form.Get("user_id")
	user, err := authApi.authManager.GetUserByUUID(userID)

	if err != nil{
		log.Fatal(err)
	}

	userProfile, err = authApi.authManager.GetUserProfile(user)
	log.Printf("user complete profile, \n%v",userProfile)

	if err != nil{
		log.Fatal(err)

	}

	userProfileByte, err := json.Marshal(userProfile)

	if err != nil{
		log.Fatal(err)
	}

	_, _ = res.Write(userProfileByte)

}


func(authApi *AuthApi) GetRouter(path string) *mux.Router{
	return authApi.router.Path(path).Subrouter()
}