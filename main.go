package main

import (
	"log"
	"os"
	"net/http"

	"ws/callgo"
	
	"github.com/gorilla/mux"
)

func main() {
	// init
	router := mux.NewRouter()
	router.Use(callgo.EnableCORS)
	
	// get port
	port := os.Getenv("PORT")
    if port == "" {
        port = "4321"
    }

	// Handle endpoints
	callgo.HandleEndpoint(router)

	// log and start
	log.Printf("Server starting on port %s.", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
