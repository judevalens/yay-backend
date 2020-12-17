package model

import "cloud.google.com/go/firestore"

type TweetFeed struct {
	ownerID string
	db firestore.Client
}

func NewTweetFeed(db firestore.Client) *TweetFeed {
	newTweetFeed := new (TweetFeed)
	newTweetFeed.db = db
	return newTweetFeed
}

type TweetFeedRepository interface {
	 GetFeed(feedID string)
	 SetGreatestID(greatestID string)
}


