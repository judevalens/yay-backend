package app

import "yaybackEnd/model/user"
type AuthManager struct {

}

// TODO need to figure out which params we need
func (authenticator *AuthManager) getTwitterAccessTokenHandler(){

}

func (authenticator *AuthManager) GetTwitterAccessToken(user user.User) (string,string) {
	/*
	userDoc := authenticator.fireStoreDB.Collection("users").Doc(uuid)
	userDocSnapShot, _ := userDoc.Get(authenticator.ctx)
	userDocData := userDocSnapShot.Data()
	spotifyAccountData := userDocData["twitter_account"].(map[string]interface{})
	return spotifyAccountData["oauth_token"].(string),spotifyAccountData["oauth_token_secret"].(string)
	*
	 */

	return "",""
}

