package repository

import (
	"cloud.google.com/go/firestore"
	"context"
	"errors"
	"math/big"
	"yaybackEnd/model"

	"log"
	"time"
	_ "yaybackEnd/auth"
	_ "yaybackEnd/helpers"
	"yaybackEnd/job_queue"
)

type ContentManagerFireStoreRepository struct {
	db  *firestore.Client
	ctx context.Context
	//app.AuthManagerRepository
}

func NewContentManagerFireStoreRepository (db *firestore.Client,ctx context.Context )*ContentManagerFireStoreRepository{

	newContentManagerFireStoreRepository := new(ContentManagerFireStoreRepository)
	newContentManagerFireStoreRepository.db = db
	newContentManagerFireStoreRepository.ctx =ctx

	return newContentManagerFireStoreRepository

}

func (c ContentManagerFireStoreRepository) SelectArtistBatch(pollingStateChan chan bool, feedRetrievalQueue *job_queue.WorkerPool, max int) {

	artistsRef := c.db.Collection("artists_feed_retrieval_queue").Where("state", "==", "done").Where("last_fetch", "<=", time.Now().Unix()-int64((time.Minute).Seconds())).Limit(max)
	_ = c.db.RunTransaction(c.ctx, func(ctx context.Context, transaction *firestore.Transaction) error {
		artistsFeedQueue := transaction.Documents(artistsRef)

		selectedArtist, selectedArtistErr := artistsFeedQueue.GetAll()

		if selectedArtistErr != nil {
			log.Fatal(selectedArtistErr)
		}

		nSelectedArtist := len(selectedArtist)

		log.Printf("%v artists were retrived", nSelectedArtist)

		pollingStateChan <- nSelectedArtist >= 50

		for _, artistFetchStateSnapShot := range selectedArtist {
			log.Print(artistFetchStateSnapShot.Data())

			artist := model.NewArtistFeedQueue(artistFetchStateSnapShot.Data())
			addedToQueue := feedRetrievalQueue.AddJob(artist)

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

func (c ContentManagerFireStoreRepository) UpdateArtistTwitterFeed(artistQueueItem model.ArtistFeedQueue, timeLines []map[string]interface{},dispatcher *job_queue.WorkerPool) (int, error) {
	var greatestID = big.NewInt(-1)
	feedDoc := c.db.Collection("artists_twitter_feeds").Doc(artistQueueItem.TwitterID)
	tweetsCollection := feedDoc.Collection("tweets")

	var tweetsInfo []map[string]interface{}

	nTweet := 0

	for _, tweet := range timeLines {
		nTweet++
		tweetID := tweet["id_str"].(string)
		tweetIDInt := big.NewInt(-1)
		tweetIDInt, tweetIDIntErr := tweetIDInt.SetString(tweetID, 10)

		if !tweetIDIntErr {
			log.Fatal("converting id to int failed")
		}

		if tweetIDInt.Cmp(greatestID) > 0 {
			greatestID = greatestID.Set(tweetIDInt)
		}
		_, twitterFeedUpdateErr := tweetsCollection.Doc(tweetID).Set(c.ctx, map[string]interface{}{
			"tweet": tweet,
		}, firestore.MergeAll)

		if twitterFeedUpdateErr != nil {
			nTweet--
		}else{
			tweetsInfo = append(tweetsInfo, map[string]interface{}{
				"tweet_id":  tweetID,
				"from_twitter_id": artistQueueItem.TwitterID,
				"from_twitter_spotify_id": artistQueueItem.TwitterID,
				"retrieved_time": time.Now().Unix(),
			})
		}

	}

	dispatcher.AddJob(map[string]interface{}{
		"from": artistQueueItem,
		"content": tweetsInfo,
		// TODO we should probably use an enum here
		"content_type": "tweet",
	})


	if greatestID.Cmp(big.NewInt(-1)) > 0 {
		_, greatestIDUpdateErr := feedDoc.Set(c.ctx, map[string]interface{}{
			"greatest_id": greatestID.String(),
		}, firestore.MergeAll)

		if greatestIDUpdateErr != nil {
			//TODO not the right way to handle this :)
			return nTweet, greatestIDUpdateErr
		}

	}

	return nTweet, nil
}

func (c ContentManagerFireStoreRepository) GetFeedGreatestTweetID(artistTwitterID string) (string, error) {

	artistFeedDoc := c.db.Collection("artists_twitter_feeds").Doc(artistTwitterID)

	artistFeedDocSnapShot, artistFeedDocErr := artistFeedDoc.Get(c.ctx)
	if artistFeedDocErr != nil {
		log.Print(artistFeedDocErr)
	}

	maxID, maxIDErr := artistFeedDocSnapShot.DataAt("greatest_id")

	if maxIDErr != nil {
		return "", maxIDErr
	}

	return maxID.(string), nil

}

func (c ContentManagerFireStoreRepository) UpdateArtistQueueState(artistSpotifyID string) {
	var stateUpdateErr error = errors.New("")
	var retryCounter time.Duration = time.Second * 2

	for stateUpdateErr != nil {
		time.Sleep(retryCounter)
		_, stateUpdateErr = c.db.Collection("artists_feed_retrieval_queue").Doc(artistSpotifyID).Update(c.ctx, []firestore.Update{
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
			log.Printf("artist queue state failed to update: \n%v", stateUpdateErr)

			retryCounter = time.Duration(retryCounter.Nanoseconds() + time.Second.Nanoseconds())

			log.Printf("duration is %v", retryCounter.String())
		}

	}
}
