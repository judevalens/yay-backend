package user

import (
	"yaybackEnd/model/FeedContent"
)
type User struct {
	uuid string
	SpotifyID string
	TwitterID string
	SpotifyAccount map[string]interface{}
	TwitterAccount map[string]interface{}
	UsersRepositoryI
}

func NewUser(uuid string, spotifyAccountData,twitterAccountData map[string]interface{})*User{
	newUser := new(User)
	newUser.uuid = uuid
	newUser.SpotifyAccount = spotifyAccountData
	newUser.TwitterAccount = twitterAccountData
	return  newUser
}

func NewUserByUUID(uuid string)*User{
	return nil
}

func(user *User) Authenticate()  {
}
func (user *User) followArtist(Artist *Artist)  {
}
func (user *User) GetSpotifyToken()  {
}


func (user *User) getSpotifyUserInfo(){
}

func (user *User) GetUserUUID() string {
	return user.uuid
}
func (user *User) addContentToFeed(content FeedContent.Content)  {
}

func (user *User) UpdateSpotifyOauthInfo(accessToken string, accessTokenTimeStamp int64){

}

func (user *User) GetAccessTokenTimeStamp() int64{
	return 0
}

func (user *User) GetSpotifyAccount() map[string]interface{}{
	return user.SpotifyAccount
}

func (user *User) GetUserTwitterOauth()(string,string){
	// TODO check if values exists
	return user.SpotifyAccount["oauth_token"].(string),user.SpotifyAccount["oauth_secret"].(string)
}


type UsersRepositoryI interface {

	GetID()string
}
