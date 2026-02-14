package main

import (
	"log"
	"net/http"
)

func main() {
	setupApi()
}

func setupApi() {
	manager := NewManager()
	http.Handle("GET /", http.FileServer(http.Dir("./frontend")))
	http.HandleFunc("GET /ws", manager.serverWebsocket)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
