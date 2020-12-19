package model

type Artist struct {

	// it's
	id	string
	spotifyAccount map[string]interface{}
	twitterAccount map[string]interface{}
	followers interface{}
}

func NewArtist(artistAccountData map[string]interface{}) *Artist {
	newArtist := new(Artist)
	newArtist.spotifyAccount = artistAccountData["spotify_account"].(map[string]interface{})
	newArtist.twitterAccount = artistAccountData["spotify_account"].(map[string]interface{})
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
