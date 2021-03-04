package helpers

import (
	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"
	"log"
)
const algoliaToken = "947dbe7381b65d1cff9ba87081814258"
const appID = "9VD06SK8X2"
type AlgoliaSearch struct {
	client *search.Client
	userIndex *search.Index
}

func NewAlgoliaSearch() *AlgoliaSearch{
	newAlgoliaSearch := new(AlgoliaSearch)
	newAlgoliaSearch.client = search.NewClient(appID,algoliaToken)
	newAlgoliaSearch.userIndex = newAlgoliaSearch.client.InitIndex("yay_users")
	return newAlgoliaSearch
}


func (a *AlgoliaSearch) IndexUser(user interface{}) error {
	_, indexingErr := a.userIndex.SaveObject(user)

	return indexingErr
}

func(a *AlgoliaSearch) SearchUsers(query string) ([]map[string]interface{},error){
	res, resErr := a.userIndex.Search(query)
	if resErr == nil{
		log.Printf("alogia res , %v",res.Hits)

	}



	if resErr != nil{
		return nil, resErr
	}else {

		return res.Hits,nil
	}
}
