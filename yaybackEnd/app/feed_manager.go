package app

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
	auth2 "yaybackEnd/auth"
	"yaybackEnd/helpers"
	"yaybackEnd/job_queue"
	"yaybackEnd/model"
)

type ContentManager struct {
	httpClient       http.Client
	jobType          int
	oldestTimeStamp  int
	artistFeedPuller *job_queue.WorkerPool
	feedPoller            *FeedPoller
	stopPulling      bool
	stopPollingChan  chan bool
	ContentManagerRepository
}

func NewContentManager(repo ContentManagerRepository,httpClient http.Client) *ContentManager {
	contentManager := new(ContentManager)
	contentManager.artistFeedPuller = job_queue.NewWorkerPool(contentManager, 1, 2)
	contentManager.stopPollingChan = make(chan bool, 100)
	contentManager.feedPoller = NewFeedPoller(contentManager)
	contentManager.ContentManagerRepository = repo
	contentManager.httpClient = httpClient
	contentManager.startSelection()
	return contentManager
}

func (a *ContentManager) startSelection() {
	i := 0
	a.artistFeedPuller.Start()
	for i < 1 {
		a.artistFeedPuller.AddJob(true)
		a.stopPollingChan <- true
		i++
	}

}

func (a *ContentManager) Worker(id int, job interface{}) {
	for {
		select {
		case poll := <-a.stopPollingChan:
			if poll {
				a.ContentManagerRepository.SelectArtistBatch(a.stopPollingChan, a.feedPoller.Poller, 25)
			}
		case <-time.After(time.Second * 20):
			a.ContentManagerRepository.SelectArtistBatch(a.stopPollingChan, a.feedPoller.Poller, 25)
		}
	}
}

// TODO : there is a better to select the feeds that need to be retrieved !
func (a *ContentManager) selectArtists() {
	log.Printf("pulling feed :...")
	//	a.stopPollingChan <- false
	a.ContentManagerRepository.SelectArtistBatch(a.stopPollingChan, a.artistFeedPuller, 25)
	log.Printf("pulling feed done...")

}

type FeedPoller struct {
	Poller *job_queue.WorkerPool
	*ContentManager
}

func NewFeedPoller(manager *ContentManager) *FeedPoller {
	newPoller := new(FeedPoller)

	newPoller.Poller = job_queue.NewWorkerPool(newPoller, 3, 10)
	newPoller.ContentManager = manager

	newPoller.Poller.Start()
	return newPoller
}

func (f *FeedPoller) Worker(id int, job interface{}) {
	artist := job.(model.ArtistFeedQueue)
	timelines, _ := f.poll(artist)
	_, _ = f.processTimeline(artist, timelines)
	go func() { f.updateSingleArtistFetchState(artist.SpotifyID) }()
}
func (f *FeedPoller) poll(artistQueueItem model.ArtistFeedQueue) ([]map[string]interface{}, error) {
	twitterTimeLineUrl := "https://api.twitter.com/1.1/statuses/user_timeline.json"
	params := url.Values{}

	artistTwitterID := artistQueueItem.TwitterID

	maxID, maxIDErr := f.GetFeedGreatestTweetID(artistTwitterID)

	if maxIDErr == nil {
		params.Add("since_id", maxID)
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

	artistTimeLineReS, artistTimeLineReSErr := f.ContentManager.httpClient.Do(artistTimeLineReq)

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

	return artistTimeLineBody, nil
}

func (f *FeedPoller) processTimeline(artistQueueItem model.ArtistFeedQueue, timeLines []map[string]interface{}) (int, error) {

	n, updateErr := f.UpdateArtistTwitterFeed(artistQueueItem, timeLines)

	log.Printf("%v tweets were process. \nerror %v", n, updateErr)

	return n, updateErr

}

func (f *FeedPoller) updateSingleArtistFetchState(artistSpotifyID string) {

	f.UpdateArtistQueueState(artistSpotifyID)

}

type ContentManagerRepository interface {
	SelectArtistBatch(pollingStateChan chan bool, feedRetrievalQueue *job_queue.WorkerPool, max int)
	GetFeedGreatestTweetID(artistSpotifyID string) (string, error)
	UpdateArtistTwitterFeed(artistQueueItem model.ArtistFeedQueue, timeLines []map[string]interface{}) (int, error)
	UpdateArtistQueueState(artistSpotifyID string)
}
