package repository

import (
	"cloud.google.com/go/firestore"
	"context"
	"errors"
	"math/big"
	"yaybackEnd/model"

	"log"
	"time"
	_ "yaybackEnd/helpers"
	"yaybackEnd/job_queue"
)

type ContentManagerFireStoreRepository struct {
	db  *firestore.Client
	ctx context.Context
	//app.AuthManagerRepository
}

func (c ContentManagerFireStoreRepository) GetTweetFlow(trackID string) (map[string]interface{}, error) {

	var tweets []map[string]interface{}

	tweetFlowDocSnapShot := c.db.Collection("tweetFlows").Doc(trackID).Collection("tweets").Limit(50).Documents(c.ctx)

	tweetSnapShots, tweetFlowDocErr := tweetFlowDocSnapShot.GetAll()

	if tweetFlowDocErr != nil {
		return nil, tweetFlowDocErr
	}

	for _, tweetSnapShot := range tweetSnapShots {
		tweets = append(tweets,tweetSnapShot.Data())
	}

	tweetFlowSnapShot, tweetFlowSnapShotErr := c.db.Collection("tweetFlows").Doc(trackID).Get(c.ctx)
	if tweetFlowSnapShotErr != nil{
		return nil, tweetFlowSnapShotErr
	}

	tweetFlow  := tweetFlowSnapShot.Data()
	tweetFlow["tweets"] = tweets

		log.Printf("tweet flow : %v",tweetFlow)
	return tweetFlow, nil
}

func (c ContentManagerFireStoreRepository) UpdateTweetFlow(trackID string, tweets []interface{}) (map[string]interface{},error) {
	tweetFlowDoc := c.db.Collection("tweetFlows").Doc(trackID)
	nProcessedTweet := 0
	for _, tweetI := range tweets {
		tweet := tweetI.(map[string]interface{})
		tweetID := tweet["id_str"].(string)

		tweetDate := tweet["created_at"].(string)
		tweetTimeStamp, tweetTimeStampErr := time.Parse(time.RubyDate, tweetDate)

		if tweetTimeStampErr != nil {
			log.Print(tweetTimeStamp)
			continue
		}
		tweet["sorting_timestamp"] = tweetTimeStamp.Unix()
		_, addingTweetErr := tweetFlowDoc.Collection("tweets").Doc(tweetID).Set(c.ctx, tweet,firestore.MergeAll)

		if addingTweetErr != nil {
			log.Print(addingTweetErr)
		}
		nProcessedTweet++
	}

	if nProcessedTweet > 0 {
		_, _ = tweetFlowDoc.Set(c.ctx, map[string]interface{}{
			"last_fetched": time.Now().Unix(),
			"test": "test",
		},firestore.MergeAll)
	}



	return c.GetTweetFlow(trackID)

}

func (c ContentManagerFireStoreRepository) GetArtistFollowers(artistSpotifyID string) ([]string, error) {
	log.Printf("retrieving followers for : %v", artistSpotifyID)
	artistDoc, artistDocErr := c.db.Collection("artists").Doc(artistSpotifyID).Get(c.ctx)
	if artistDocErr != nil {
		return nil, artistDocErr
	}

	artistFollowersI, _ := artistDoc.DataAt("followers")

	var followers []string

	for _, follower := range artistFollowersI.([]interface{}) {
		followers = append(followers, follower.(string))
	}

	return followers, nil
}

func (c ContentManagerFireStoreRepository) UpdateFollowerFeed(userSpotifyID string, feedItems []map[string]interface{}) (int, error) {

	userFeedDoc := c.db.Collection("users_feed").Doc(userSpotifyID)
	addedItemCounter := 0
	for _, item := range feedItems {
		addedItemCounter++
		item["sever_timestamp"] = firestore.ServerTimestamp
		_, addItemToUserFeedErr := userFeedDoc.Collection("items").Doc(item["item_id"].(string)).Set(c.ctx, item)

		if addItemToUserFeedErr != nil {
			addedItemCounter--
		}
	}
	return addedItemCounter, nil
}

func NewContentManagerFireStoreRepository(db *firestore.Client, ctx context.Context) *ContentManagerFireStoreRepository {

	newContentManagerFireStoreRepository := new(ContentManagerFireStoreRepository)
	newContentManagerFireStoreRepository.db = db
	newContentManagerFireStoreRepository.ctx = ctx

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

		log.Printf("%v artists were retrieved", nSelectedArtist)

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

func (c ContentManagerFireStoreRepository) UpdateArtistTwitterFeed(artistQueueItem model.ArtistFeedQueue, timeLines []map[string]interface{}, dispatcher *job_queue.WorkerPool) (int, error) {
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
		} else {
			tweetDate := tweet["created_at"].(string)
			tweetTimeStamp, tweetTimeStampErr := time.Parse(time.RubyDate, tweetDate)

			if tweetTimeStampErr != nil {
				log.Printf("err")
				nTweet--
			} else {
				tweetsInfo = append(tweetsInfo, map[string]interface{}{
					"item_id":               tweetID,
					"creator_type":          "artist",
					"sorting_timestamp":     tweetTimeStamp.Unix(),
					"created_by_twitter_id": artistQueueItem.TwitterID,
					"created_by_spotify_id": artistQueueItem.SpotifyID,
					"timestamp":             firestore.ServerTimestamp,
					"content_type":          "tweet",
					"seen":                  false,
				})
			}

		}

	}

	if nTweet > 0 {
		dispatcher.AddJob(map[string]interface{}{
			"created_by": artistQueueItem,
			"content":    tweetsInfo,
			// TODO we should probably use an enum here
			"content_type": "tweet",
		})
	}

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
