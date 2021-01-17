package api

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"yaybackEnd/app"
)

type RelationApi struct {
	router          *mux.Router
	relationManager *app.RelationManager
}

func NewRelationApi(router *mux.Router, manager *app.RelationManager) *RelationApi {
	newRelationApi := new(RelationApi)
	s, _ := router.Path("/relation").GetPathTemplate()
	log.Printf("route name : %v", s)
	newRelationApi.router = router.PathPrefix("/relation").Subrouter()
	newRelationApi.relationManager = manager
	newRelationApi.setRoutes()
	return newRelationApi
}
func (relationApi *RelationApi) setRoutes() {
	r := relationApi.router.HandleFunc("/updateFollowedArtistList", relationApi.updateFollowedArtists).Methods("POST")

	relationApi.router.HandleFunc("/searchUser", relationApi.searchUsers).Methods("GET")
	relationApi.router.HandleFunc("/isFollowingUser", relationApi.isFollowingUser).Methods("GET")
	relationApi.router.HandleFunc("/friendUsers", relationApi.friendUsers).Methods("GET")

	path, _ := r.GetPathTemplate()

	log.Printf("relation route : %v", path)
}
func (relationApi *RelationApi) updateFollowedArtists(res http.ResponseWriter, req *http.Request) {

	_ = req.ParseForm()

	userID := req.Form.Get("user_id")

	user, userErr := relationApi.relationManager.AuthManager.GetUserByUUID(userID)

	if userErr != nil {
		//TODO must handle error
		log.Fatal(userErr)
	}
	relationApi.relationManager.UpdateFollowedArtistList(user)

	// I dont really know what to send yet
	_, _ = res.Write([]byte(""))
}
func (relationApi *RelationApi) searchUsers(res http.ResponseWriter, req *http.Request) {
	var resByte []byte
	_ = req.ParseForm()

	query := req.Form.Get("query")

	log.Printf("got search request : %v", query)

	searchRes, searchResErr := relationApi.relationManager.SearchUsers(query)

	log.Printf("search res \n%v", searchRes)
	log.Printf("search err \n%v", searchResErr)

	if searchResErr != nil {
		resByte, _ = json.Marshal(map[string]interface{}{
			"status": 400,
			"error":  searchResErr,
		})
	} else {
		resByte, _ = json.Marshal(map[string]interface{}{
			"status": 200,
			"users":  searchRes,
		})
	}

	_, _ = res.Write(resByte)

}
func (relationApi *RelationApi) isFollowingUser(res http.ResponseWriter, req *http.Request) {
	var resByte []byte
	var err error
	_ = req.ParseForm()

	userIdA := req.Form.Get("user_a_id")
	userIdB := req.Form.Get("user_b_id")

	UserA, err := relationApi.relationManager.GetUserByUUID(userIdA)

	if err != nil {
		resByte, err = json.Marshal(map[string]interface{}{
			"status": 400,
			"error":  err,
		})

		_, _ = res.Write(resByte)
		return
	}

	UserB, err := relationApi.relationManager.GetUserByUUID(userIdB)

	if err != nil {
		resByte, err = json.Marshal(map[string]interface{}{
			"status": 400,
			"error":  err,
		})

		_, _ = res.Write(resByte)
		return
	}

	isFollowing, err := relationApi.relationManager.IsFollowingUser(UserA, UserB)

	if err != nil {
		resByte, err = json.Marshal(map[string]interface{}{
			"status": 400,
			"error":  err,
		})

		_, _ = res.Write(resByte)
		return
	}

	resByte, err = json.Marshal(map[string]interface{}{
		"status":      200,
		"isFollowing": isFollowing,
	})

	_, _ = res.Write(resByte)

}
func (relationApi *RelationApi) friendUsers(res http.ResponseWriter, req *http.Request) {
	var resByte []byte
	var err error
	_ = req.ParseForm()

	userIdA := req.Form.Get("user_a_id")
	userIdB := req.Form.Get("user_b_id")
	

	UserA, err := relationApi.relationManager.GetUserByUUID(userIdA)

	if err != nil {
		resByte, err = json.Marshal(map[string]interface{}{
			"status": 400,
			"error":  err,
		})

		_, _ = res.Write(resByte)
		return
	}

	UserB, err := relationApi.relationManager.GetUserByUUID(userIdB)

	if err != nil {
		resByte, err = json.Marshal(map[string]interface{}{
			"status": 400,
			"error":  err,
		})

		_, _ = res.Write(resByte)
		return
	}
	
	log.Printf("friending users %v and %v",UserA.GetUserUUID(),UserB.GetUserUUID())

	 err = relationApi.relationManager.FollowUser(UserA, UserB)

	if err != nil {
		resByte, err = json.Marshal(map[string]interface{}{
			"status": 400,
			"error":  err,
		})

		_, _ = res.Write(resByte)
		return
	}

	resByte, err = json.Marshal(map[string]interface{}{
		"status":      200,
	})

	_, _ = res.Write(resByte)

}
