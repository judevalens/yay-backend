package model

type TwitterFeed struct {
	ownerID string
	TweetFeedRepositoryI
}

func NewTweetFeed(tweetFeedRepository TweetFeedRepositoryI) *TwitterFeed {
	newTweetFeed := new (TwitterFeed)
	newTweetFeed.TweetFeedRepositoryI = tweetFeedRepository
	return newTweetFeed
}

type TweetFeedRepositoryI interface {
	 GetFeed(feedID string)
	 AddTweet(tweet map[string]interface{})
	 GreatestID(greatestID string)
	 GetTweets()
}


