package FeedContent

type Tweet struct {
	id string
	ownerID string
	rawTweet map[string]interface{}
}

func NewTweet(rawTweet map[string]interface{}) *Tweet{
	newTweet := new(Tweet)
	newTweet.id  = rawTweet["id_str"].(string)
	newTweet.rawTweet = rawTweet
	return newTweet
}
