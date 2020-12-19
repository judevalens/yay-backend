package model

type Artist struct {

	// it's
	id	string
	spotifyAccount map[string]interface{}
	twitterAccount map[string]interface{}
	followers interface{}
}

type ArtistFeedQueue struct {
	LastFetch int64
	SpotifyID string
	TwitterID string
	State string
}

func NewArtistFeedQueue(artistFeedQueueItem map[string]interface{}) ArtistFeedQueue{
	newArtistFeedQueue := ArtistFeedQueue{}
	newArtistFeedQueue.LastFetch = artistFeedQueueItem["last_fetch"].(int64)
	newArtistFeedQueue.SpotifyID = artistFeedQueueItem["spotify_id"].(string)
	newArtistFeedQueue.TwitterID = artistFeedQueueItem["twitter_id"].(string)
	newArtistFeedQueue.State = artistFeedQueueItem["state"].(string)

	return newArtistFeedQueue
}


func NewArtist(artistAccountData map[string]interface{}) *Artist {
	newArtist := new(Artist)
	newArtist.spotifyAccount = artistAccountData["spotify_account"].(map[string]interface{})
	newArtist.twitterAccount = artistAccountData["twitter_account"].(map[string]interface{})
	newArtist.id = artistAccountData["id"].(string)
	return newArtist
}

func (a *Artist) GetTwitterAccount() map[string]interface{}{
	return  a.twitterAccount
}

func (a *Artist) GetSpotifyAccount() map[string]interface{}{
	return  a.spotifyAccount
}

func (a *Artist) GetID() string{
	return  a.id
}
