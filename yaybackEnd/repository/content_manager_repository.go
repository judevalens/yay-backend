package repository

import (
	"cloud.google.com/go/firestore"
	"context"
	"log"
	"time"
	"yaybackEnd/app"
	"yaybackEnd/job_queue"
)

type ContentManagerFireStoreRepo struct {
	db  *firestore.Client
	ctx context.Context
	app.AuthManagerRepository
}

func (c ContentManagerFireStoreRepo) SelectArtistBatch(pollingStateChan chan bool, feedRetrievalQueue *job_queue.WorkerPool, max int) {

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
			addedToQueue := feedRetrievalQueue.AddJob(artistFetchStateSnapShot.Data())

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
