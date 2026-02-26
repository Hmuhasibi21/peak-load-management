package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello dari API ElasticSix! Infrastruktur Docker Ready 🚀"))
	})

	fmt.Println("Server jalan di port 8080...")
	http.ListenAndServe(":8080", nil)
}