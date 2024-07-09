package main

import (
	"fmt"
	"log"
	api "mf-backend/api"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
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

	fmt.Println("...")
	log.Println("Server started listening on port :" + appPort)

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(router)
	log.Fatal(http.ListenAndServe(":"+appPort, handler))
}
