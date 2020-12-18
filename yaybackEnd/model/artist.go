package model

type Artist struct {
	spotifyAccount map[string]interface{}
	twitterAccount map[string]interface{}
	followers interface{}
}

func NewArtist(artistAccountData map[string]interface{}) Artist {
	newArtist := Artist{}
	newArtist.spotifyAccount = artistAccountData["spotify_account"].(map[string]interface{})
	newArtist.twitterAccount = artistAccountData["spotify_account"].(map[string]interface{})

	return newArtist
}
