package main

import (
	"log"
	"net/http"
)

func Root(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello 民生物联网"))
}

func main() {
	log.Printf("hello")

	http.HandleFunc("/", Root)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("%+v", err)
	}
}
