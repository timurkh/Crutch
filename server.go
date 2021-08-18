package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// trick to conver my functions to http.Handler
type appHandler func(http.ResponseWriter, *http.Request) error

func (fn appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e := fn(w, r); e != nil {
		log.Println("appHandler error: " + e.Error())
	}
}

func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	port := getEnv("PORT", "3001")
	baseUrl := getEnv("BASEURL", "crutchdev")
	elastic := getEnv("ELASTIC", "http://10.130.0.21:9400")
	sh := initSearchHelper(elastic)

	router := mux.NewRouter().StrictSlash(true)
	crutch := router.PathPrefix("/" + baseUrl).Subrouter()

	crutch.Methods("POST").Path("/api/searchProducts").Handler(appHandler(sh.searchProductsHandler))
	fs := http.FileServer(http.Dir("./frontend/dist"))
	crutch.PathPrefix("/").Handler(http.StripPrefix("/"+baseUrl, fs))

	http.Handle("/", handlers.CombinedLoggingHandler(os.Stdout, router))

	log.Printf("Server listening on port %s, base url %s\n", port, baseUrl)
	log.Panic(
		http.ListenAndServe(":"+port, nil),
	)
}
