package api

import (
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"yaybackEnd/app"
)

type RelationApi struct {
	router *mux.Router
	relationManager *app.RelationManager
}

func NewRelationApi(router *mux.Router,manager *app.RelationManager) *RelationApi{
	newRelationApi :=  new(RelationApi)
	s, _ := router.Path("/relation").GetPathTemplate()
	log.Printf("route name : %v",s )
	newRelationApi.router = router.PathPrefix("/relation").Subrouter()
	newRelationApi.relationManager = manager
	newRelationApi.setRoutes()
	return newRelationApi
}
func(relationApi *RelationApi) setRoutes(){
	r := relationApi.router.HandleFunc("/updateFollowedArtistList", relationApi.updateFollowedArtists).Methods("POST")

	path , _ := r.GetPathTemplate()

	log.Printf("relation route : %v", path)
}
func (relationApi *RelationApi) updateFollowedArtists(res http.ResponseWriter,req *http.Request)  {

	_ = req.ParseForm()

	userID := req.Form.Get("user_id")

	user, userErr := relationApi.relationManager.AuthManager.GetUserByUUID(userID)

	if userErr != nil {
		//TODO must handle error
		log.Fatal(userErr)
	}
	relationApi.relationManager.UpdateFollowedArtistList(user)

	// I dont really know what to send yet
	res.Write([]byte(""))
}
