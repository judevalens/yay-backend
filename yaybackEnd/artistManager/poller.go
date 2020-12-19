package artistManager

import (
	"cloud.google.com/go/firestore"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"time"
	auth2 "yaybackEnd/auth"
	"yaybackEnd/helpers"
	"yaybackEnd/job_queue"
)

type ArtistSelectionWorker struct {
	jobType          int
	oldestTimeStamp  int
	artistFeedPuller *job_queue.WorkerPool
	stopPulling      bool
	stopPollingChan  chan bool
	artistManager    *ArtistManager
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
func NewArtistSelectionWorker(manager *ArtistManager) *ArtistSelectionWorker {
	artistSelectionWorker := ArtistSelectionWorker{}
	artistSelectionWorker.artistFeedPuller = job_queue.NewWorkerPool(&artistSelectionWorker, 1, 2)
	artistSelectionWorker.stopPollingChan = make(chan bool, 100)
	log.Printf("manager %v\n", manager.fireStoreDB)
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
		case <-time.After(time.Minute * 5):
			a.selectArtists()
		}
	}
}

// TODO : there is a better to select the feeds that need to be retrieved !
func (a *ArtistSelectionWorker) selectArtists() {
	log.Printf("pulling feed : %v...", a.artistManager)
	//	a.stopPollingChan <- false
	log.Printf("pulling feed done...")

	artistsRef := a.artistManager.fireStoreDB.Collection("artists_feed_retrieval_queue").Where("state", "==", "done").Where("last_fetch", "<=", time.Now().Unix()-int64((time.Minute).Seconds())).Limit(50)
	_ = a.artistManager.fireStoreDB.RunTransaction(a.artistManager.ctx, func(ctx context.Context, transaction *firestore.Transaction) error {
		artistsFeedQueue := transaction.Documents(artistsRef)

		selectedArtist, selectedArtistErr := artistsFeedQueue.GetAll()

		if selectedArtistErr != nil {
			log.Fatal(selectedArtistErr)
		}

		nSelectedArtist := len(selectedArtist)

		log.Printf("%v artists were retrived", nSelectedArtist)

		a.artistManager.artistSelectionWorker.stopPollingChan <- nSelectedArtist >= 50

		for _, artistFetchStateSnapShot := range selectedArtist {
			addedToQueue := a.artistManager.FeedPoller.Poller.AddJob(artistFetchStateSnapShot.Data())

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
			} else {
				log.Printf("failed to update state")
			}
		}

		return nil
	})
}

type FeedPoller struct {
	Poller *job_queue.WorkerPool
	*ArtistManager
}

func (f *FeedPoller) Worker(id int, job interface{}) {
	artistObj, timelines := f.poll(job)
	if f.processTimeline(artistObj, timelines) {
		artistSpotifyID := artistObj["spotifyID"].(string)

		go func() { f.updateSingleArtistFetchState(artistSpotifyID) }()
	}
}

func NewFeedPoller(manager *ArtistManager) *FeedPoller {
	newPoller := new(FeedPoller)

	newPoller.Poller = job_queue.NewWorkerPool(newPoller, 3, 10)
	newPoller.ArtistManager = manager

	newPoller.Poller.Start()
	return newPoller
}

func (f *FeedPoller) updateSingleArtistFetchState(artistSpotifyID string) {

	var stateUpdateErr error = errors.New("")
	var retryCounter time.Duration = time.Second * 2

	for stateUpdateErr != nil {
		time.Sleep(retryCounter)
		_, stateUpdateErr = f.fireStoreDB.Collection("artists_feed_retrieval_queue").Doc(artistSpotifyID).Update(f.ctx, []firestore.Update{
			{
				Path:  "last_fetch",
				Value: time.Now().Unix(),
			},
			{
				Path:  "state",
				Value: "done",
			},
		})
		if stateUpdateErr != nil {
			log.Printf("state update failed: \n%v", stateUpdateErr)

			retryCounter = time.Duration(retryCounter.Nanoseconds() + time.Second.Nanoseconds())

			log.Printf("duration is %v", retryCounter.String())
		}

	}

}

func (f *FeedPoller) poll(artist interface{}) (map[string]interface{}, []map[string]interface{}) {
	twitterTimeLineUrl := "https://api.twitter.com/1.1/statuses/user_timeline.json"
	params := url.Values{}

	artistObj := artist.(map[string]interface{})

	artistTwitterID := artistObj["twitterID"].(string)

	artistFeedDoc := f.fireStoreDB.Collection("artists_twitter_feeds").Doc(artistTwitterID)

	artistFeedDocSnapShot, artistFeedDocErr := artistFeedDoc.Get(f.ctx)
	if artistFeedDocErr != nil {
		log.Print(artistFeedDocErr)
	}

	maxID, maxIDErr := artistFeedDocSnapShot.DataAt("greatest_id")

	if maxIDErr != nil {
		log.Printf("set since_ID")
	}

	if maxID != nil {
		params.Add("since_id", maxID.(string))
	}

	params.Add("count", "50")

	params.Add("user_id", artistTwitterID)

	_, oauthHeader := helpers.OauthSignature("GET", twitterTimeLineUrl, auth2.TwitterSecretKey, "", params, helpers.GetAuthParams(nil))

	artistTimeLineReq, artistTimeLineReqErr_ := http.NewRequest("GET", twitterTimeLineUrl, nil)
	if artistTimeLineReqErr_ != nil {
		log.Fatal(artistTimeLineReqErr_)
	}
	artistTimeLineReq.Header.Add("Authorization", oauthHeader)
	artistTimeLineReq.URL.RawQuery = params.Encode()
	log.Printf("retrieving tweeets for :  %v", artistTwitterID)

	artistTimeLineReS, artistTimeLineReSErr := f.ArtistManager.httpClient.Do(artistTimeLineReq)

	if artistTimeLineReSErr != nil {
		log.Fatal(artistTimeLineReSErr)
	}

	var artistTimeLineBody []map[string]interface{}

	artistTimeLineBodyBytes, artistTimeLineBodyErr := ioutil.ReadAll(artistTimeLineReS.Body)

	if artistTimeLineBodyErr != nil {
		log.Fatal(artistTimeLineBodyErr)
	}
	jsonParsingErr := json.Unmarshal(artistTimeLineBodyBytes, &artistTimeLineBody)

	if jsonParsingErr != nil {
		log.Printf("%v", string(artistTimeLineBodyBytes))
		log.Fatal(jsonParsingErr)
	}

	log.Printf("retrieved %v tweeets , status code : \n%v", len(artistTimeLineBody), artistTimeLineReS.Status)

	return artistObj, artistTimeLineBody
}

func (f *FeedPoller) processTimeline(artist map[string]interface{}, timeLines []map[string]interface{}) bool {
	var greatestID = big.NewInt(-1)
	artistTwitterID := artist["twitterID"].(string)
	feedDoc := f.ArtistManager.fireStoreDB.Collection("artists_twitter_feeds").Doc(artistTwitterID)
	tweetsCollection := feedDoc.Collection("tweets")

	for _, tweet := range timeLines {
		tweetID := tweet["id_str"].(string)
		tweetIDInt := big.NewInt(-1)
		tweetIDInt, tweetIDIntErr := tweetIDInt.SetString(tweetID, 10)

		if !tweetIDIntErr {
			log.Fatal("converting id to int failed")
		}

		if tweetIDInt.Cmp(greatestID) > 0 {
			greatestID = greatestID.Set(tweetIDInt)
		}
		tweetsCollection.Doc(tweetID).Set(f.ArtistManager.ctx, map[string]interface{}{
			"tweet": tweet,
		}, firestore.MergeAll)

		f.ContentDispatcher.ContentDispatcher.Dispatcher.AddJob(tweet)
	}

	if greatestID.Cmp(big.NewInt(-1)) > 0 {
		feedDoc.Set(f.ArtistManager.ctx, map[string]interface{}{
			"greatest_id": greatestID.String(),
		}, firestore.MergeAll)

	}

	return true

}

type ContentDispatcher struct {
	Dispatcher *job_queue.WorkerPool
	*ArtistManager
}

func (c ContentDispatcher) Worker(id int, job interface{}) {
	panic("implement me")
}

func NewContentDispatcher(manager *ArtistManager) *ContentDispatcher {
	newContentDispatcher := new(ContentDispatcher)
	newContentDispatcher.ArtistManager = manager
	newContentDispatcher.Dispatcher = job_queue.NewWorkerPool(newContentDispatcher, 10, 50)

	return newContentDispatcher
}

func (c *ContentDispatcher) contentDispatcher() {

}
