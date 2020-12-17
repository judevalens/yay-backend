package user

import (
	"os/user"
	"yaybackEnd/model/FeedContent"
)
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

type UsersRepositoryI interface {
	AuthenticateUSer(user user.User) user.User
	GetUserBySpotifyID(spotifyID string)user.User
	GetUserByTwitterID(twitterID string)user.User
	GetUserSpotifyAccessToken(twitterID string)string
	GetUserTwitterOauth(twitterID string)(string,string)
}
