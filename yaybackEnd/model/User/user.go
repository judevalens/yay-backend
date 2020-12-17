package User

import "yaybackEnd/model/FeedContent"
type User struct {
	SpotifyID string
	TwitterID string
}

func NewUser()*User{
	return nil
}

func(user *User) Authenticate()  {

}

func (user *User) followArtist(Artist *Artist)  {
}

func (user *User) GetSpotifyToken()  {
}

func (user *User) addContentToFeed(content FeedContent.Content)  {

}
