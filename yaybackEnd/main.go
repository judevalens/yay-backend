package main
import (
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"yaybackEnd/artistManager"
	"yaybackEnd/auth"

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
		log.Fatalf("error initializing app: %v\n", err)
	}
	authClient, error := app.Auth(ctx)

	db, dbError := app.Database(ctx)

	fireStoreDB, fireStoreDBErr := app.Firestore(ctx)

	if dbError != nil && fireStoreDBErr != nil{
		log.Fatal(dbError)
	}

	log.Printf("getting auth client , error : %v", error)

	var router = mux.NewRouter()

	authenticator := auth.NewAuthenticator(authClient, db,fireStoreDB, ctx, router)

	_ = artistManager.GetArtistManger(authClient, db,fireStoreDB, ctx, authenticator, router)

	//_ = http.ListenAndServe(addr, router)

	err = http.ListenAndServeTLS(addr, "cert.pem", "key.pem", router)
	log.Fatal(err)


	fmt.Printf("hello %v",app)

}


