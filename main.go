package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	setupApi()
}

func setupApi() {

	manager := NewManager()
	manager.RegisterEventHandler(EventSendMessage, SendMessage)
	http.Handle("GET /", http.FileServer(http.Dir("./frontend")))
	http.HandleFunc("GET /otp", getOTP)
	http.HandleFunc("GET /ws", manager.serverWebsocket)
	log.Fatal(http.ListenAndServeTLS(":8080", "cert.crt", "cert.key", nil))
}

func getOTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	otp, err := issueOTP("admin")
	if err != nil {
		log.Printf("failed to generate otp: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to generate otp"})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"otp": otp})
}
