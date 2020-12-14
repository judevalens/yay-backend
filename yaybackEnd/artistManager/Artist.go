package artistManager

import (
	"cloud.google.com/go/firestore"
	"context"
	"encoding/json"
	"firebase.google.com/go/auth"
	"firebase.google.com/go/db"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
	auth2 "yaybackEnd/auth"
	"yaybackEnd/job_queue"
	"yaybackEnd/misc"
)

const (
	pullArtistsID      = 0
	retrieveArtistFeed = 1
)

type ArtistSelectionWorker struct {
	jobType          int
	oldestTimeStamp  int
	artistFeedPuller *job_queue.WorkerPool
	stopPulling      bool
	stopPollingChan  chan bool
	artistManager *ArtistManager
}

func (a ArtistSelectionWorker) startSelection() {
	i := 0
	a.artistFeedPuller.Start()
	for i < 1 {
		a.artistFeedPuller.AddJob(true)
		a.stopPollingChan <- true
		i++
	}

}
func  NewArtistSelectionWorker(manager *ArtistManager) *ArtistSelectionWorker {
	artistSelectionWorker := ArtistSelectionWorker{}
	artistSelectionWorker.artistFeedPuller = job_queue.NewWorkerPool(&artistSelectionWorker, 1, 2)
	artistSelectionWorker.stopPollingChan = make(chan bool, 100)
	log.Printf("manager %v\n",manager.fireStoreDB)
	artistSelectionWorker.artistManager = manager
	return &artistSelectionWorker
}

func (a *ArtistSelectionWorker) Worker(id int, job interface{}) {
	for {
		select {
		case poll := <-a.stopPollingChan:
			if poll {
				a.selectArtists()
			}
		case <-time.After(time.Second * 30):
			a.selectArtists()
		}
	}
}
// TODO : there is a better to select the feeds that need to be retrieved !
func (a *ArtistSelectionWorker) selectArtists() {
	log.Printf("pulling feed : %v...",a.artistManager)
//	a.stopPollingChan <- false
	log.Printf("pulling feed done...")

		artistsRef := a.artistManager.fireStoreDB.Collection("artists_feed_retrieval_queue").Where("state", "==", "done").Where("last_fetch", "<=", time.Now().Unix()-int64((time.Second*120).Seconds())).Limit(50)
		_ = a.artistManager.fireStoreDB.RunTransaction(a.artistManager.ctx, func(ctx context.Context, transaction *firestore.Transaction) error {
			artistsFeedQueue := transaction.Documents(artistsRef)

			selectedArtist, selectedArtistErr := artistsFeedQueue.GetAll()

			if selectedArtistErr != nil{
				log.Fatal(selectedArtistErr)
			}

			nSelectedArtist := len(selectedArtist)

			log.Printf("%v artists were retrived",nSelectedArtist)

			a.artistManager.artistSelectionWorker.stopPollingChan <- nSelectedArtist >= 50



			for _, artistFetchStateSnapShot := range selectedArtist{
				addedToQueue := a.artistManager.workerPool.AddJob(artistFetchStateSnapShot)

				if addedToQueue {
					log.Printf("updating state....")
					_ = transaction.Update(artistFetchStateSnapShot.Ref, []firestore.Update{
						{
							Path:  "last_fetch",
							Value: time.Now().Unix(),
						},
						{
							Path:  "state",
							Value: "queued",
						},
					})
				}else{
					log.Printf("failed to update state")
				}
			}

			return nil
		})
}

type ArtistManager struct {
	authClient            *auth.Client
	db                    *db.Client
	fireStoreDB           *firestore.Client
	router                *mux.Router
	httpClient            http.Client
	ctx                   context.Context
	artistSelectionWorker *ArtistSelectionWorker
	workerPool            *job_queue.WorkerPool
	authenticator         *auth2.Authenticator
}

func (artistManager *ArtistManager) Worker(id int, job interface{}) {

}

func GetArtistManger(_authClient *auth.Client, _db *db.Client, _fireStoreDB *firestore.Client, _ctx context.Context, _authenticator *auth2.Authenticator, router *mux.Router) *ArtistManager {
	newArtistManager := ArtistManager{}
	newArtistManager.authClient = _authClient
	newArtistManager.fireStoreDB = _fireStoreDB
	newArtistManager.ctx = _ctx
	newArtistManager.router = router
	newArtistManager.authenticator = _authenticator
	newArtistManager.workerPool = job_queue.NewWorkerPool(&newArtistManager, 50, 1000)
	newArtistManager.artistSelectionWorker =NewArtistSelectionWorker(&newArtistManager)
	newArtistManager.workerPool.Start()
	newArtistManager.artistSelectionWorker.startSelection()
	newArtistManager.setRoutes()

	return &newArtistManager
}

func (artistManager *ArtistManager) setRoutes() {

	artistManager.router.HandleFunc("/artist/updateFollowedArtistList", artistManager.updateFollowedArtistListHandler)
}

func (artistManager *ArtistManager) updateFollowedArtistListHandler(res http.ResponseWriter, req *http.Request) {

	//paramsByte, _ := ioutil.ReadAll()

	_ = req.ParseForm()

	params, _ := url.ParseQuery(string(req.Form.Encode()))

	log.Printf("user id %v", params)

	artistManager.updateFollowedArtistList(params.Get("userID"))
}

func (artistManager *ArtistManager) getFollowedArtistOnSpotify(accessToken string) map[string]interface{} {

	followedArtistUrl := "https://api.spotify.com/v1/me/following?type=artist"

	followedArtistReq, _ := http.NewRequest("GET", followedArtistUrl, nil)

	followedArtistReq.Header.Set("Authorization", "Bearer "+accessToken)

	followedArtistRes, _ := artistManager.httpClient.Do(followedArtistReq)

	followedArtistResBody, followedArtistReqErr := ioutil.ReadAll(followedArtistRes.Body)

	if followedArtistReqErr != nil {
		log.Fatal(followedArtistReqErr)
	}

	log.Printf("followed artist : %v", string(followedArtistResBody))

	var followedArtistJson map[string]interface{}

	_ = json.Unmarshal(followedArtistResBody, &followedArtistJson)

	return followedArtistJson
}

func (artistManager *ArtistManager) updateFollowedArtistList(userID string) {
	accessToken, _ := artistManager.authenticator.GetSpotifyRefreshedAccessToken(userID)
	followedArtistSpotify := (artistManager.getFollowedArtistOnSpotify(accessToken)["artists"]).(map[string]interface{})["items"].([]interface{})

	userDoc := artistManager.fireStoreDB.Collection("users").Doc(userID)

	userDocSnapShot, _ := userDoc.Get(artistManager.ctx)
	followedArtist, _ := userDocSnapShot.DataAt("followed_artist")

	if followedArtist == nil {
		userDoc.Set(artistManager.ctx, map[string]interface{}{
			"followed_artist": []interface{}{},
		}, firestore.MergeAll)
	}

	// add the followed artist
	for i, _ := range followedArtistSpotify {
		artist := followedArtistSpotify[i].(map[string]interface{})
		artistID := artist["id"].(string)

		artistQuerySnapShot, artistQueryErr := artistManager.fireStoreDB.Collection("users").Where("id", "==", userID).Where("followed_artist", "array-contains", artistID).Documents(artistManager.ctx).GetAll()

		if artistQueryErr != nil {
			log.Fatal(artistQueryErr)
		}

		if len(artistQuerySnapShot) > 0 {
			continue
		}

		artistManager.followArtist(artist, artist["name"].(string), userID)

		userDoc.Update(artistManager.ctx, []firestore.Update{
			{Path: "followed_artist",
				Value: firestore.ArrayUnion(artistID),
			},
		})
	}

}

func (artistManager *ArtistManager) followArtist(artistSpotifyObject map[string]interface{}, artistSpotifyName, userID string) bool {

	artistSpotifyID := artistSpotifyObject["id"].(string)

	artistCollection := artistManager.fireStoreDB.Collection("artists").Doc(artistSpotifyID)
	artistCollectionSnapShot, _ := artistCollection.Get(artistManager.ctx)
	artistExist := artistCollectionSnapShot.Exists()

	if artistExist {
		log.Printf("artistExist")
		return true
	}

	twitterUserOauthToken, twitterUserOauthTokenSecret := artistManager.authenticator.GetTwitterAccessToken(userID)

	twitterArtistSearchUrl := "https://api.twitter.com/1.1/users/search.json"

	params := url.Values{}

	params.Set("q", artistSpotifyName)
	params.Set("count", "1")

	oauthParams := url.Values{}

	oauthParams.Add("oauth_consumer_key", auth2.TwitterApiKey)
	oauthParams.Add("oauth_nonce", strconv.FormatInt(time.Now().Unix(), 10))
	oauthParams.Add("oauth_version", "1.0")
	oauthParams.Add("oauth_signature_method", "HMAC-SHA1")
	oauthParams.Add("oauth_version", "1.0")
	oauthParams.Add("oauth_token", twitterUserOauthToken)
	oauthParams.Add("oauth_timestamp", strconv.FormatInt(time.Now().Unix(), 10))

	_, oauthHeader := misc.OauthSignature("GET", twitterArtistSearchUrl, auth2.TwitterSecretKey, twitterUserOauthTokenSecret, params, oauthParams)

	artistReq, _ := http.NewRequest("GET", twitterArtistSearchUrl, nil)

	artistReq.URL.RawQuery = params.Encode()
	artistReq.Header.Add("Authorization", oauthHeader)

	searchedArtistResponse, searchedArtistErr := artistManager.httpClient.Do(artistReq)

	if searchedArtistErr != nil {
		log.Fatal(searchedArtistErr.Error())
	}

	var searchedArtistBytes []byte
	var searchedArtistJson interface{}

	searchedArtistBytes, _ = ioutil.ReadAll(searchedArtistResponse.Body)

	json.Unmarshal(searchedArtistBytes, &searchedArtistJson)

	twitterArtistObject := searchedArtistJson.([]interface{})[0].(map[string]interface{})

	artistDocs := artistManager.fireStoreDB.Collection("artists").Doc(artistSpotifyID)
	artistDocSnapShot, _ := artistDocs.Get(artistManager.ctx)
	entryExists := artistDocSnapShot.Exists()





	if !entryExists {

		_, _, addToQueueErr := artistManager.fireStoreDB.Collection("artists_feed_retrieval_queue").Add(artistManager.ctx,
			map[string]interface{}{

				"id" : artistSpotifyID,
				"last_fetch": time.Now().Unix(),
				"state": "done",
				"twitterID":twitterArtistObject["id"],

			})

		if addToQueueErr != nil{
			log.Fatal(addToQueueErr)
		}

		artistDocs.Set(artistManager.ctx, map[string]interface{}{
			"spotify_account": artistSpotifyObject,
			"twitter_account": twitterArtistObject,
			"followers":       []interface{}{},
		})
	}


	artistDocs.Update(artistManager.ctx, []firestore.Update{
		{
			Path:  "followers",
			Value: firestore.ArrayUnion(userID),
		},
	})

	log.Printf("new artist found: \n%v\n\n", string(searchedArtistBytes))

	return true
}