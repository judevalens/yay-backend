
package app

import (
	"log"
	"time"
	"yaybackEnd/job_queue"
)

type ArtistSelectionWorker struct {
jobType          int
oldestTimeStamp  int
artistFeedPuller *job_queue.WorkerPool
stopPulling      bool
stopPollingChan  chan bool
	ContentManagerRepository
}

func NewArtistSelectionWorker() *ArtistSelectionWorker {
	artistSelectionWorker := ArtistSelectionWorker{}
	artistSelectionWorker.artistFeedPuller = job_queue.NewWorkerPool(&artistSelectionWorker, 1, 2)
	artistSelectionWorker.stopPollingChan = make(chan bool, 100)

	return &artistSelectionWorker
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
	log.Printf("pulling feed :...", )
	//	a.stopPollingChan <- false
	a.ContentManagerRepository.SelectArtistBatch(a.stopPollingChan,a.artistFeedPuller,25)
	log.Printf("pulling feed done...")

}

type ContentManagerRepository interface {
	SelectArtistBatch(pollingStateChan chan bool, feedRetrievalQueue *job_queue.WorkerPool, max int)}