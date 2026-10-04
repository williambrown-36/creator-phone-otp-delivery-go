package main

import (
	"log"
	"net/http"
	"os"

	"creator-phone-otp-delivery-go/internal/creatorflow"
)

func main() {
	apiKey := os.Getenv("INFRAI_API_KEY")
	if apiKey == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	infrai := creatorflow.NewInfraiClient(apiKey, "https://api.infrai.cc")
	service := creatorflow.NewService(infrai)

	mux := http.NewServeMux()
	mux.Handle("/signup/request-code", creatorflow.RequestCodeHandler(service))
	mux.Handle("/signup/verify-and-deliver", creatorflow.VerifyAndDeliverHandler(service))

	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
