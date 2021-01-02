package api

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"yaybackEnd/app"
)



type ContentApi struct {
	router *mux.Router
	contentManager *app.ContentManager
}

func NewContentApi(router *mux.Router,manager *app.ContentManager) *ContentApi{
	newContentApi :=  new(ContentApi)
	s, _ := router.Path("/relation").GetPathTemplate()
	log.Printf("route name : %v",s )
	newContentApi.router = router.PathPrefix("/content").Subrouter()
	newContentApi.contentManager = manager
	newContentApi.setRoutes()
	return newContentApi
}
func(contentApi *ContentApi) setRoutes(){
	r := contentApi.router.HandleFunc("/tweetFlow", contentApi.HandleTweetFlow).Methods("POST")

	path , _ := r.GetPathTemplate()

	log.Printf("content routes : %v", path)
}
func (contentApi *ContentApi) HandleTweetFlow(res http.ResponseWriter,req *http.Request)  {

	var params map[string]interface{}
	var resByte []byte

	// TODO need to handle errors
	paramsBytes, _ := ioutil.ReadAll(req.Body)

	paramsUnmarshalErr := json.Unmarshal(paramsBytes, &params)

	if paramsUnmarshalErr != nil{

		resByte, _ = json.Marshal(map[string]string{
			"status": "400",
			"error":  paramsUnmarshalErr.Error(),
		})

		_, _ = res.Write(resByte)

	}

	userID := params["user_id"].(string)
	trackInfo := params["track_info"].(map[string]interface{})

	user, userErr := contentApi.contentManager.AuthManager.GetUserByUUID(userID)

	if userErr != nil {
		//TODO must handle error
		log.Fatal(userErr)
	}
	tweetFlow, tweetFlowErr := contentApi.contentManager.TweetFlow(user,trackInfo)

	if tweetFlowErr != nil{
		resByte, _ = json.Marshal(map[string]interface{}{
			"status": 400,
			"error":  tweetFlowErr.Error(),
		})
	}else{
		resByte , _ = json.Marshal(map[string]interface{}{
			"status": 200,
			"tweetFlow":  tweetFlow,
		})
	}

	// I dont really know what to send yet
	_, _ = res.Write(resByte)
}
