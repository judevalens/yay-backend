package main

import (
	"context"
	"encoding/json"
	"firebase.google.com/go/auth"
	"firebase.google.com/go/db"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
)

type CSW struct {
	authClient *auth.Client
	db *db.Client
	router *mux.Router
	httpClient http.Client
}


func newCSW(_authClient *auth.Client, _db *db.Client, router *mux.Router) *CSW {
	var authenticator = new(CSW)
	authenticator.authClient = _authClient
	authenticator.router = router
	authenticator.httpClient  = http.Client{}
	authenticator.db = _db

	return authenticator
}

func (csw *CSW) init()  {
	csw.router.HandleFunc("/login", csw.startStreamHandler).Methods("POST")
}


func (csw *CSW) startStreamHandler(response http.ResponseWriter,req *http.Request){
	log.Printf("new login request from %v", req.RemoteAddr)

	var params map[string]interface{}
	rawBody , _ := ioutil.ReadAll(req.Body)

	 roomID := params["room_id"].(string)

	 userID := params["user_id"].(string)

	json.Unmarshal(rawBody, &params)

	var currentRoom map[string]interface{}
	userRefErr := csw.db.NewRef("users/"+userID).Get(context.Background(),&currentRoom)

	if userRefErr != nil{
		log.Fatal(userRefErr)
	}

	// removes the user from it's current room
	if currentRoom != nil {
		endStream()
	}

	roomToStartRef := csw.db.NewRef("rooms").Child(roomID)

	roomToStartRef.Update(context.Background(), map[string]interface{}{
		"is_active": true,
	})
}


func endStream(){

}





