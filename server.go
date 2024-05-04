package main

import (
	"log"
	api "mf-backend/api"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func init() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}
}

// Start service on port APP_PORT
func server() {

	router := mux.NewRouter()

	api.ApiRoutes(router)

	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		log.Println("APP_PORT can't be empty, check .env")
		return
	}

	log.Println("Server started listening on port :" + appPort)

	log.Fatal(http.ListenAndServe(":"+appPort, router))
}
