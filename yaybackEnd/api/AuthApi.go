package api

import (
	"github.com/gorilla/mux"
	"yaybackEnd/app"
)

type AuthApi struct {
	r *mux.Router
	authManager app.AuthManager
}

func NewAuthApi(router mux.Router){
	newAuthApi := new(AuthApi)
	newAuthApi.r = router.Path("/auth").Subrouter()
}
func(authApi *AuthApi) setRoutes(){

}
func(authApi *AuthApi) GetRouter(path string) *mux.Router{
	return authApi.r.Path(path).Subrouter()
}