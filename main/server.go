package main

import (
	"log"
	"net/http"
)

func main() {

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	sh := initSearchHelper()

	fs := http.FileServer(http.Dir("../frontend/dist"))
	http.Handle("/", fs)
	http.HandleFunc("/api/searchProducts", sh.searchProductsHandler)

	log.Println("Server listening on port 3000")
	log.Panic(
		http.ListenAndServe(":3000", nil),
	)
}
