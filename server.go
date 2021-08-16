package main

import (
	"io"
	"log"
	"log/syslog"
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

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	var (
		logwriter io.Writer
		err       error
	)
	logwriter, err = syslog.New(syslog.LOG_USER|syslog.LOG_INFO, "crutch")
	if err == nil {
		log.SetOutput(logwriter)
	} else {
		log.Printf("Failed to redirect logging to syslog, %s\n", err.Error())
		logwriter = os.Stdout
	}

	sh := initSearchHelper()

	router := mux.NewRouter().StrictSlash(true)
	crutch := router.PathPrefix("/crutch").Subrouter()

	crutch.Methods("POST").Path("/api/searchProducts").Handler(appHandler(sh.searchProductsHandler))
	fs := http.FileServer(http.Dir("./frontend/dist"))
	crutch.PathPrefix("/").Handler(http.StripPrefix("/crutch", fs))

	http.Handle("/", handlers.CombinedLoggingHandler(logwriter, router))

	log.Println("Server listening on port 3000")
	log.Panic(
		http.ListenAndServe(":3000", nil),
	)
}
