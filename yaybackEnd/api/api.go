package api

import "github.com/gorilla/mux"

type Router interface {
	GetRouter () *mux.Router
}


