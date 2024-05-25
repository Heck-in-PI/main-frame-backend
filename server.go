package main

import (
	"fmt"
	"log"
	api "mf-backend/api"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func init() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}
}

func sheesh() {
	for retry := 0; ; retry++ {
		log.Println(runtime.NumGoroutine())
		time.Sleep(1 * time.Second)
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
	var wg sync.WaitGroup

	log.Println("here")
	wg.Add(1)
	go sheesh()
	defer wg.Wait()
	log.Fatal(http.ListenAndServe(":"+appPort, router))
}
