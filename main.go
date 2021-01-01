package main

import (
	"cloud.google.com/go/firestore"
	"context"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"yaybackEnd/api"
	app2 "yaybackEnd/app"
	"yaybackEnd/repository"

	firebase "firebase.google.com/go"
	_ "firebase.google.com/go/auth"

	_ "google.golang.org/api/option"
)

var addr string = ":8000"

var ctx = context.Background()
var conf = &firebase.Config{
DatabaseURL: "https://yay-music.firebaseio.com/",
}

type App struct {
	router mux.Router
	fireStore firestore.Client
}

func (a App) getSubRouter(nameSpace string) *mux.Router {
	panic("implement me")
}

func (a App) setHandler(path string, handlerFunc http.HandlerFunc) {
	panic("implement me")
}

func main() {

	app, err := firebase.NewApp(context.Background(), conf)
	if err != nil {
		log.Fatalf("authClientErr initializing app: %v\n", err)
	}
	authClient, authClientErr := app.Auth(ctx)

	_, dbError := app.Database(ctx)

	fireStoreDB, fireStoreDBErr := app.Firestore(ctx)

	if dbError != nil && fireStoreDBErr != nil{
		log.Fatal(dbError)
	}

	log.Printf("getting auth client , authClientErr : %v", authClientErr)

	var router = mux.NewRouter()
	authManagerRepository := repository.NewUserFireStoreRepository(fireStoreDB,ctx)
	authManager := app2.NewAuthManager(authClient,http.Client{},ctx,authManagerRepository)

	relationManagerRepository := repository.NewRelationsFireStoreRepository(fireStoreDB,ctx)
	relationManager := app2.NewRelationManager(http.Client{},authManager,relationManagerRepository)

	contentManagerRepository := repository.NewContentManagerFireStoreRepository(fireStoreDB,ctx)

	_ = app2.NewContentManager(contentManagerRepository,http.Client{},authManager)

	_ = api.NewAuthApi(router,authManager)

	_ = api.NewRelationApi(router,relationManager)

	router.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
	log.Printf("test")
	})


	err = http.ListenAndServeTLS(addr, "cert.pem", "key.pem", router)
	log.Fatal(err)
}

