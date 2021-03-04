package main

import (
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"google.golang.org/api/option"
	"log"
	"net/http"
	"os"
	"yaybackEnd/api"
	app2 "yaybackEnd/app"
	"yaybackEnd/helpers"
	"yaybackEnd/repository"

	firebase "firebase.google.com/go"
	_ "firebase.google.com/go/auth"

	_ "google.golang.org/api/option"
)

var port = os.Getenv("PORT")

var addr string = ":" + port

var ctx = context.Background()
var conf = &firebase.Config{
	DatabaseURL: "https://yay-music.firebaseio.com/",
}

type App struct {
	router    mux.Router
	fireStore firestore.Client
}

func (a App) getSubRouter(nameSpace string) *mux.Router {
	panic("implement me")
}

func (a App) setHandler(path string, handlerFunc http.HandlerFunc) {
	panic("implement me")
}

func main() {
	port = os.Getenv("PORT")
	fmt.Printf("add : %s\n", port)

	opt := option.WithCredentialsFile("yay-music-firebase-adminsdk-c31yg-77d2819dfa.json")
	app, err := firebase.NewApp(context.Background(), conf, opt)
	if err != nil {
		log.Fatalf("error initializing app: %v", err)
	}
	authClient, authClientErr := app.Auth(ctx)

	_, dbError := app.Database(ctx)

	fireStoreDB, fireStoreDBErr := app.Firestore(ctx)

	if dbError != nil && fireStoreDBErr != nil {
		log.Fatal(dbError)
	}

	log.Printf("getting auth client , authClientErr : %v", authClientErr)

	searchService := helpers.NewAlgoliaSearch()

	var router = mux.NewRouter()
	authManagerRepository := repository.NewUserFireStoreRepository(fireStoreDB, ctx)
	authManager := app2.NewAuthManager(authClient, http.Client{}, ctx, authManagerRepository, searchService)

	relationManagerRepository := repository.NewRelationsFireStoreRepository(fireStoreDB, ctx)
	relationManager := app2.NewRelationManager(http.Client{}, authManager, relationManagerRepository, searchService)

	contentManagerRepository := repository.NewContentManagerFireStoreRepository(fireStoreDB, ctx)

	contentManager := app2.NewContentManager(contentManagerRepository, http.Client{}, authManager)

	_ = api.NewAuthApi(router, authManager)

	_ = api.NewRelationApi(router, relationManager)

	_ = api.NewContentApi(router, contentManager)

	router.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		log.Printf("test")
	})

	err = http.ListenAndServeTLS(addr, "cert.pem", "key.pem", router)
	log.Fatal(err)
}
