package repository

import (
	"cloud.google.com/go/firestore"
	"context"
	"log"
	"time"
	"yaybackEnd/app"
	"yaybackEnd/model"
)

type RelationsFireStoreRepository struct {
	db  *firestore.Client
	ctx context.Context
	app.AuthManagerRepository
}

func (r RelationsFireStoreRepository) IsFollowingUser(userA, userB *model.User) (bool, error) {

	_,  isFollowingErr := r.db.Collection("users").Doc(userA.GetUserUUID()).Collection("followed_users").Doc(userB.GetUserUUID()).Get(r.ctx)

	if isFollowingErr != nil{
		return false,isFollowingErr
	}

	return true, nil

}

func (r RelationsFireStoreRepository) FollowUser(userA, userB *model.User) error {

	_, followUserErr := r.db.Collection("users").Doc(userB.GetUserUUID()).Collection("followed_users").Doc(userA.GetUserUUID()).Set(r.ctx,
		map[string]interface{}{
			"following": true,
		})

	return followUserErr
}

func (r RelationsFireStoreRepository) GetArtistBySpotifyID(spotifyID string) (*model.Artist, error) {
	artistDoc := r.db.Collection("artists").Doc(spotifyID)

	artistDocSnapShot, artistDocSnapShotErr := artistDoc.Get(r.ctx)

	if artistDocSnapShotErr != nil {
		return nil, artistDocSnapShotErr
	}

	artistDocData := artistDocSnapShot.Data()

	return model.NewArtist(artistDocData), nil
}

func (r RelationsFireStoreRepository) GetArtistByTwitterID(spotifyID string) *model.Artist {
	panic("implement me")
}

func (r RelationsFireStoreRepository) AddArtist(artistAccountData map[string]interface{}, spotifyID string) error {
	artistDoc := r.db.Collection("artists").Doc(spotifyID)
	artistQueueDoc := r.db.Collection("artists_feed_retrieval_queue").Doc(spotifyID)
	return r.db.RunTransaction(r.ctx, func(ctx context.Context, transaction *firestore.Transaction) error {

		_, addToArtistErr := artistDoc.Set(r.ctx, artistAccountData)

		if addToArtistErr != nil {
			log.Printf("addToArtistErr 1 \n%v", artistAccountData)
			return addToArtistErr
		}
		addToQueueErr := transaction.Set(artistQueueDoc, map[string]interface{}{

			"spotify_id": spotifyID,
			"last_fetch": time.Now().Unix(),
			"state":      "done",
			"twitter_id": artistAccountData["twitter_account"].(map[string]interface{})["id_str"].(string),
		})

		return addToQueueErr
	})

}

func (r RelationsFireStoreRepository) GetFollowedArtist(user *model.User) []*model.Artist {
	userDoc := r.db.Collection("users").Doc(user.GetUserUUID())

	userDocSnapShot, _ := userDoc.Get(r.ctx)
	followedArtistsIdJSON, _ := userDocSnapShot.DataAt("followed_artist")

	if followedArtistsIdJSON == nil {
		_, _ = userDoc.Set(r.ctx, map[string]interface{}{
			"followed_artists": []interface{}{},
		}, firestore.MergeAll)
	}

	var followedArtist []*model.Artist
	followedArtistIDList := followedArtistsIdJSON.([]string)

	for _, artistSpotifyID := range followedArtistIDList {
		artist, _ := r.GetArtistBySpotifyID(artistSpotifyID)
		// TODO must handle error
		if artist != nil {
			followedArtist = append(followedArtist, artist)
		}
	}

	return followedArtist

}

func (r RelationsFireStoreRepository) IsFollowingArtist(user *model.User, artist *model.Artist) bool {

	//log.Printf("artist %v",artist)

	artistQuerySnapShot, artistQueryErr := r.db.Collection("users").Where("id", "==", user.GetUserUUID()).Where("followed_artists", "array-contains", artist.GetID()).Documents(r.ctx).GetAll()
	log.Printf("artists ID : %v, len : %v", artist.GetID(), len(artistQuerySnapShot))

	if artistQueryErr != nil {
		log.Print(artistQueryErr)
	}

	return len(artistQuerySnapShot) > 0

}

func (r RelationsFireStoreRepository) FollowArtist(user *model.User, artist *model.Artist) error {
	userDoc := r.db.Collection("users").Doc(user.GetUserUUID())
	artistDoc := r.db.Collection("artists").Doc(artist.GetID())
	followArtistErr := r.db.RunTransaction(r.ctx, func(ctx context.Context, transaction *firestore.Transaction) error {
		userArtistListUpdateErr := transaction.Update(userDoc, []firestore.Update{
			{Path: "followed_artists",
				Value: firestore.ArrayUnion(artist.GetID()),
			},
		})

		if userArtistListUpdateErr != nil {
			return userArtistListUpdateErr
		}

		artistFollowerUpdateErr := transaction.Update(artistDoc, []firestore.Update{
			{Path: "followers",
				Value: firestore.ArrayUnion(user.GetUserUUID()),
			},
		})

		if artistFollowerUpdateErr != nil {
			return artistFollowerUpdateErr
		}

		return nil

	})

	return followArtistErr
}

func NewRelationsFireStoreRepository(db *firestore.Client, ctx context.Context) *RelationsFireStoreRepository {
	newUserFireStoreRepository := new(RelationsFireStoreRepository)
	newUserFireStoreRepository.db = db
	newUserFireStoreRepository.ctx = ctx
	return newUserFireStoreRepository
}
