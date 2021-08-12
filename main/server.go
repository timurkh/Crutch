package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	fs := http.FileServer(http.Dir("../frontend/dist"))
	http.Handle("/", fs)
	http.HandleFunc("/api/searchProducts", searchProductsHandler)

	fmt.Println("Server listening on port 3000")
	log.Panic(
		http.ListenAndServe(":3000", nil),
	)
}

func searchProductsHandler(w http.ResponseWriter, r *http.Request) {

	time.Sleep(8 * time.Second)

	products := []string{"краска", "гаечный ключ"}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(products)
	if err != nil {
		log.Println(err.Error())
	}
}
