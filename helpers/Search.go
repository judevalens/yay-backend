package helpers

import (
	"github.com/algolia/algoliasearch-client-go/algolia/search"
)
type AlgoliaSearch struct {
	client *search.Client
	userIndex *search.Index
}

func NewAlgoliaSearch() *AlgoliaSearch{
	newAlgoliaSearch := new(AlgoliaSearch)
	newAlgoliaSearch.client = search.NewClient("","")
	newAlgoliaSearch.userIndex = newAlgoliaSearch.client.InitIndex("users")
	return newAlgoliaSearch
}


func (a *AlgoliaSearch) IndexUser(user interface{}) error {
	_, indexingErr := a.userIndex.SaveObject(user)

	return indexingErr
}

func(a *AlgoliaSearch) SearchUsers(query string) ([]interface{},error){
	res, resErr := a.userIndex.Search(query)
	if resErr != nil{
		return nil, resErr
	}else {
		return res.UserData,nil
	}
}
