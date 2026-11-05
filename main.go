package main

import (
	"log"
	"net/http"
)

func main() {
	handler := http.NewServeMux()
	
	server := http.Server{
		Addr: ":8080",
		Handler: handler,
	}
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal("Server is broken...")
	}
	defer server.Close()
}
