package main
import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"

	firebase "firebase.google.com/go"
	_ "firebase.google.com/go/auth"

	_ "google.golang.org/api/option"
)

var addr string = ":8000"

var ctx = context.Background()
var conf = &firebase.Config{
DatabaseURL: "https://yay-music.firebaseio.com/",
}
func main() {



	app, err := firebase.NewApp(context.Background(), conf)
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}
	authClient, error := app.Auth(ctx)

	db, dbError := app.Database(ctx)

	if dbError != nil{
		log.Fatal(dbError)
	}

	log.Printf("getting auth client , error : %v", error)

	var router = mux.NewRouter()

	_ = newAuthenticator(authClient, db, router)

	http.ListenAndServe(addr,router)

fmt.Printf("hello %v",app)

}


